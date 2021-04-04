package node

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"strconv"
	"time"

	"github.com/bloxapp/ssv/ibft"
	"github.com/bloxapp/ssv/ibft/proto"
	"github.com/bloxapp/ssv/utils/dataval/bytesval"

	"github.com/bloxapp/ssv/beacon"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.uber.org/zap"
)

// postConsensusDutyExecution signs the eth2 duty after iBFT came to consensus,
// waits for others to sign, collect sigs, reconstruct and broadcast the reconstructed signature to the beacon chain
func (n *ssvNode) postConsensusDutyExecution(
	ctx context.Context,
	logger *zap.Logger,
	identifier []byte,
	inputValue *proto.InputValue,
	signaturesCount int,
	role beacon.Role,
	duty *ethpb.DutiesResponse_Duty,
) error {
	// TODO - closing receive sig channel
	// TODO - id is meaningless
	signaturesChan := n.network.ReceivedSignatureChan(0, identifier)

	// sign input value
	sig, root, err := n.signDuty(ctx, inputValue, role, duty)

	// broadcast
	if err != nil {
		return errors.Wrap(err, "failed to sign input data")
	}
	if err := n.network.BroadcastSignature(identifier, map[uint64][]byte{n.nodeID: sig}); err != nil {
		return errors.Wrap(err, "failed to broadcast signature")
	}

	// Collect signatures from other nodes
	// TODO - waiting timeout, when should we stop waiting for the sigs and just move on?
	signatures := make(map[uint64][]byte, signaturesCount)
	foundCnt := 0
	done := false
	for {
		if done {
			break
		}
		select {
		case sig := <-signaturesChan:
			for index, signature := range sig {
				if _, found := signatures[index]; found {
					continue
				}
				// verify sig
				if err := n.verifyPartialSignature(signature, root, index); err != nil {
					logger.Error("received invalid signature", zap.Error(err))
					continue
				}

				signatures[index] = signature
				foundCnt++
			}
			if foundCnt >= signaturesCount {
				done = true
				break
			}
		case <-time.After(n.signatureCollectionTimeout):
			err = errors.New("timed out waiting for post consensus signatures")
			done = true
			break
		}
	}

	if err != nil {
		return err
	}
	logger.Info("GOT ALL BROADCASTED SIGNATURES", zap.Int("signatures", len(signatures)))

	// Reconstruct signatures
	if err := n.reconstructAndBroadcastSignature(ctx, logger, signatures, root, inputValue, role, duty); err != nil {
		return errors.Wrap(err, "failed to reconstruct and broadcast signature")
	}
	logger.Info("Successfully submitted role!")
	return nil
}

func (n *ssvNode) comeToConsensusOnInputValue(
	ctx context.Context,
	logger *zap.Logger,
	identifier []byte,
	slot uint64,
	role beacon.Role,
	duty *ethpb.DutiesResponse_Duty,
) (int, *proto.InputValue, error) {
	l := logger.With(zap.String("role", role.String()))

	l.Info("new version without aggregator and proposal")

	inputValue := &proto.InputValue{}
	switch role {
	case beacon.RoleAttester:
		attData, err := n.beacon.GetAttestationData(ctx, slot, duty.GetCommitteeIndex())
		if err != nil {
			return 0, nil, errors.Wrap(err, "failed to get attestation data")
		}

		inputValue.Data = &proto.InputValue_AttestationData{
			AttestationData: attData,
		}
	//case beacon.RoleAggregator:
	//	aggData, err := n.beacon.GetAggregationData(ctx, slot, duty.GetCommitteeIndex())
	//	if err != nil {
	//		return 0, nil, errors.Wrap(err, "failed to get aggregation data")
	//	}
	//
	//	inputValue.Data = &proto.InputValue_AggregationData{
	//		AggregationData: aggData,
	//	}
	//case beacon.RoleProposer:
	//	block, err := n.beacon.GetProposalData(ctx, slot)
	//	if err != nil {
	//		return 0, nil, errors.Wrap(err, "failed to get proposal block")
	//	}
	//
	//	inputValue.Data = &proto.InputValue_BeaconBlock{
	//		BeaconBlock: block,
	//	}
	case beacon.RoleUnknown:
		return 0, nil, errors.New("unknown role")
	}

	valBytes, err := json.Marshal(&inputValue)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to marshal input value")
	}

	decided, signaturesCount := n.iBFT.StartInstance(ibft.StartOptions{
		Logger:       l,
		Consensus:    bytesval.New(valBytes),
		PrevInstance: identifier,
		Identifier:   []byte(strconv.Itoa(int(slot))),
		Value:        valBytes,
	})

	if !decided {
		return 0, nil, errors.New("ibft did not decide, not executing role")
	}
	return signaturesCount, inputValue, nil
}

func (n *ssvNode) executeDuty(
	ctx context.Context,
	identifier []byte,
	slot uint64,
	duty *ethpb.DutiesResponse_Duty,
) {
	logger := n.logger.With(zap.Time("start_time", n.getSlotStartTime(slot)),
		zap.Uint64("committee_index", duty.GetCommitteeIndex()),
		zap.Uint64("slot", slot))

	roles, err := n.beacon.RolesAt(ctx, slot, duty)
	if err != nil {
		logger.Error("failed to get roles for duty", zap.Error(err))
		return
	}

	for _, role := range roles {
		go func() {
			l := logger.With(zap.String("role", role.String()))

			signaturesCount, inputValue, err := n.comeToConsensusOnInputValue(ctx, logger, identifier, slot, role, duty)
			if err != nil {
				logger.Error("could not come to consensus", zap.Error(err))
				return
			}

			// Here we ensure at least 2/3 instances got a val so we can sign data and broadcast signatures
			logger.Info("GOT CONSENSUS", zap.Any("inputValue", &inputValue))

			// Sign, aggregate and broadcast signature
			if err := n.postConsensusDutyExecution(
				ctx,
				l,
				identifier,
				inputValue,
				signaturesCount,
				role,
				duty,
			); err != nil {
				logger.Error("could not execute duty", zap.Error(err))
				return
			}

			//identfier = newId // TODO: Fix race condition
		}()
	}
}
