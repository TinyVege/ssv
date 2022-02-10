package qbft

import (
	"bytes"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/pkg/errors"
)

// uponCommit returns true if a quorum of commit messages was received.
func uponCommit(state State, signedCommit *SignedMessage, commitMsgContainer MsgContainer) (bool, []byte, error) {
	if state.GetProposalAcceptedForCurrentRound() == nil {
		return false, nil, errors.New("did not receive proposal for this round")
	}

	if err := validateCommit(
		state,
		signedCommit,
		state.GetHeight(),
		state.GetRound(),
		state.GetProposalAcceptedForCurrentRound(),
		state.GetConfig().GetNodes(),
	); err != nil {
		return false, nil, errors.Wrap(err, "commit msg invalid")
	}
	if !commitMsgContainer.AddIfDoesntExist(signedCommit) {
		return false, nil, nil // uponCommit was already called
	}

	value := signedCommit.Message.GetCommitData().GetData()
	if commitQuorumForValue(state, commitMsgContainer, value) {
		return true, value, nil
	}
	return false, nil, nil
}

func commitQuorumForValue(state State, commitMsgContainer MsgContainer, value []byte) bool {
	commitMsgs := commitMsgContainer.MessagesForHeightAndRound(state.GetHeight(), state.GetRound())
	valueFiltered := make([]*SignedMessage, 0)
	for _, msg := range commitMsgs {
		// TODO - not needed as we add msgs to container after validating the value
		if bytes.Equal(msg.Message.GetCommitData().GetData(), value) {
			valueFiltered = append(valueFiltered, msg)
		}
	}

	return state.GetConfig().HasQuorum(valueFiltered)
}

// didSendCommitForHeightAndRound returns true if sent commit msg for specific height and round
func didSendCommitForHeightAndRound(state State, commitMsgContainer MsgContainer) bool {
	/**
	!exists m :: && m in current.messagesReceived
	                            && m.Commit?
	                            && var uPayload := m.commitPayload.unsignedPayload;
	                            && uPayload.height == |current.blockchain|
	                            && uPayload.round == current.round
	                            && recoverSignedCommitAuthor(m.commitPayload) == current.id
	*/

	panic("implement")
}

func createCommit(state State, value []byte) *SignedMessage {
	/**
	Commit(
	                    signCommit(
	                        UnsignedCommit(
	                            |current.blockchain|,
	                            current.round,
	                            signHash(hashBlockForCommitSeal(proposedBlock), current.id),
	                            digest(proposedBlock)),
	                            current.id
	                        )
	                    );
	*/
	panic("implement")
}

func validateCommit(
	state State,
	signedCommit *SignedMessage,
	height uint64,
	round Round,
	proposedMsg *SignedMessage,
	nodes []*types.Node,
) error {
	if signedCommit.Message.MsgType != CommitType {
		return errors.New("commit msg type is wrong")
	}
	if signedCommit.Message.Height != height {
		return errors.New("commit height is wrong")
	}
	if signedCommit.Message.Round != round {
		return errors.New("commit round is wrong")
	}
	if !bytes.Equal(proposedMsg.Message.GetCommitData().GetData(), signedCommit.Message.GetCommitData().GetData()) {
		return errors.New("proposed data different than commit msg data")
	}
	if err := signedCommit.IsValidSignature(state.GetConfig().GetSignatureDomainType(), nodes); err != nil {
		return errors.Wrap(err, "commit msg signature invalid")
	}
	return nil
}
