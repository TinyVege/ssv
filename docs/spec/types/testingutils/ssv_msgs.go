package testingutils

import (
	"github.com/bloxapp/ssv/beacon"
	"github.com/bloxapp/ssv/docs/spec/qbft"
	"github.com/bloxapp/ssv/docs/spec/ssv"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/herumi/bls-eth-go-binary/bls"
)

var AttesterMsgID = types.MessageIDForValidatorPKAndRole(TestingValidatorPubKey[:], beacon.RoleTypeAttester)
var ProposerMsgID = types.MessageIDForValidatorPKAndRole(TestingValidatorPubKey[:], beacon.RoleTypeProposer)

var TestAttesterConsensusData = &types.ConsensusData{
	Duty:            TestingAttesterDuty,
	AttestationData: TestingAttestationData,
}
var TestAttesterConsensusDataByts, _ = TestAttesterConsensusData.Encode()

var TestAggregatorConsensusData = &types.ConsensusData{
	Duty:            TestingAggregatorDuty,
	AttestationData: TestingAttestationData,
}
var TestAggregatorConsensusDataByts, _ = TestAggregatorConsensusData.Encode()

var TestProposerConsensusData = &types.ConsensusData{
	Duty:      TestingProposerDuty,
	BlockData: TestingBeaconBlock,
}
var TestProposerConsensusDataByts, _ = TestProposerConsensusData.Encode()

var TestConsensusUnkownDutyTypeData = &types.ConsensusData{
	Duty:            TestingUnknownDutyType,
	AttestationData: TestingAttestationData,
}
var TestConsensusUnkownDutyTypeDataByts, _ = TestConsensusUnkownDutyTypeData.Encode()

var TestConsensusWrongDutyPKData = &types.ConsensusData{
	Duty:            TestingWrongDutyPK,
	AttestationData: TestingAttestationData,
}
var TestConsensusWrongDutyPKDataByts, _ = TestConsensusWrongDutyPKData.Encode()

var SSVMsgAttester = func(qbftMsg *qbft.SignedMessage, postMsg *ssv.SignedPartialSignatureMessage) *types.SSVMessage {
	return ssvMsg(qbftMsg, postMsg, types.MessageIDForValidatorPKAndRole(TestingValidatorPubKey[:], beacon.RoleTypeAttester))
}

var SSVMsgWrongID = func(qbftMsg *qbft.SignedMessage, postMsg *ssv.SignedPartialSignatureMessage) *types.SSVMessage {
	return ssvMsg(qbftMsg, postMsg, types.MessageIDForValidatorPKAndRole(TestingWrongValidatorPubKey[:], beacon.RoleTypeAttester))
}

var SSVMsgProposer = func(qbftMsg *qbft.SignedMessage, postMsg *ssv.SignedPartialSignatureMessage) *types.SSVMessage {
	return ssvMsg(qbftMsg, postMsg, types.MessageIDForValidatorPKAndRole(TestingValidatorPubKey[:], beacon.RoleTypeProposer))
}

var ssvMsg = func(qbftMsg *qbft.SignedMessage, postMsg *ssv.SignedPartialSignatureMessage, msgID types.MessageID) *types.SSVMessage {
	var msgType types.MsgType
	var data []byte
	if qbftMsg != nil {
		msgType = types.SSVConsensusMsgType
		data, _ = qbftMsg.Encode()
	} else if postMsg != nil {
		msgType = types.SSVPartialSignatureMsgType
		data, _ = postMsg.Encode()
	} else {
		panic("msg type undefined")
	}

	return &types.SSVMessage{
		MsgType: msgType,
		MsgID:   msgID,
		Data:    data,
	}
}

var PostConsensusAttestationMsgWithMsgMultiSigners = func(sk *bls.SecretKey, id types.OperatorID, height qbft.Height) *ssv.SignedPartialSignatureMessage {
	return postConsensusAttestationMsg(sk, id, height, false, false, true, false)
}

var PostConsensusAttestationMsgWithNoMsgSigners = func(sk *bls.SecretKey, id types.OperatorID, height qbft.Height) *ssv.SignedPartialSignatureMessage {
	return postConsensusAttestationMsg(sk, id, height, false, false, true, false)
}

var PostConsensusAttestationMsgWithWrongSig = func(sk *bls.SecretKey, id types.OperatorID, height qbft.Height) *ssv.SignedPartialSignatureMessage {
	return postConsensusAttestationMsg(sk, id, height, false, true, false, false)
}

var PostConsensusAttestationMsgWithWrongRoot = func(sk *bls.SecretKey, id types.OperatorID, height qbft.Height) *ssv.SignedPartialSignatureMessage {
	return postConsensusAttestationMsg(sk, id, height, true, false, false, false)
}

var PostConsensusAttestationMsg = func(sk *bls.SecretKey, id types.OperatorID, height qbft.Height) *ssv.SignedPartialSignatureMessage {
	return postConsensusAttestationMsg(sk, id, height, false, false, false, false)
}

var postConsensusAttestationMsg = func(
	sk *bls.SecretKey,
	id types.OperatorID,
	height qbft.Height,
	wrongRoot bool,
	wrongBeaconSig bool,
	noMsgSigners bool,
	multiMsgSigners bool,
) *ssv.SignedPartialSignatureMessage {
	signer := NewTestingKeyManager()
	signedAtt, root, _ := signer.SignAttestation(TestingAttestationData, TestingAttesterDuty, sk.GetPublicKey().Serialize())

	if wrongBeaconSig {
		signedAtt, _, _ = signer.SignAttestation(TestingAttestationData, TestingAttesterDuty, TestingWrongSK.GetPublicKey().Serialize())
	}

	if wrongRoot {
		root = []byte{1, 2, 3, 4}
	}

	postConsensusMsg := &ssv.PartialSignatureMessage{
		Type:             ssv.PostConsensusPartialSig,
		PartialSignature: signedAtt.Signature[:],
		SigningRoot:      root,
		Signers:          []types.OperatorID{id},
	}

	if noMsgSigners {
		postConsensusMsg.Signers = []types.OperatorID{}
	}
	if multiMsgSigners {
		postConsensusMsg.Signers = []types.OperatorID{id, 5}
	}

	sig, _ := signer.SignRoot(postConsensusMsg, types.PartialSignatureType, sk.GetPublicKey().Serialize())
	return &ssv.SignedPartialSignatureMessage{
		Message:   postConsensusMsg,
		Signature: sig,
		Signers:   []types.OperatorID{id},
	}
}

var PostConsensusProposerMsg = func(sk *bls.SecretKey, id types.OperatorID) *ssv.SignedPartialSignatureMessage {
	return postConsensusBeaconBlockMsg(sk, id, false, false, false, false)
}

var postConsensusBeaconBlockMsg = func(
	sk *bls.SecretKey,
	id types.OperatorID,
	wrongRoot bool,
	wrongBeaconSig bool,
	noMsgSigners bool,
	multiMsgSigners bool,
) *ssv.SignedPartialSignatureMessage {
	signer := NewTestingKeyManager()
	signedAtt, root, _ := signer.SignBeaconBlock(TestingBeaconBlock, TestingProposerDuty, sk.GetPublicKey().Serialize())

	if wrongBeaconSig {
		//signedAtt, _, _ = signer.SignAttestation(TestingAttestationData, TestingAttesterDuty, TestingWrongSK.GetPublicKey().Serialize())
		panic("implement")
	}

	if wrongRoot {
		root = []byte{1, 2, 3, 4}
	}

	postConsensusMsg := &ssv.PartialSignatureMessage{
		Type:             ssv.PostConsensusPartialSig,
		PartialSignature: signedAtt.Signature[:],
		SigningRoot:      root,
		Signers:          []types.OperatorID{id},
	}

	if noMsgSigners {
		postConsensusMsg.Signers = []types.OperatorID{}
	}
	if multiMsgSigners {
		postConsensusMsg.Signers = []types.OperatorID{id, 5}
	}

	sig, _ := signer.SignRoot(postConsensusMsg, types.PartialSignatureType, sk.GetPublicKey().Serialize())
	return &ssv.SignedPartialSignatureMessage{
		Message:   postConsensusMsg,
		Signature: sig,
		Signers:   []types.OperatorID{id},
	}
}

var PreConsensusRandaoMsg = func(sk *bls.SecretKey, id types.OperatorID) *ssv.SignedPartialSignatureMessage {
	return randaoMsg(sk, id, false, false, false, false)
}

var randaoMsg = func(
	sk *bls.SecretKey,
	id types.OperatorID,
	wrongRoot bool,
	wrongBeaconSig bool,
	noMsgSigners bool,
	multiMsgSigners bool,
) *ssv.SignedPartialSignatureMessage {
	signer := NewTestingKeyManager()
	randaoSig, root, _ := signer.SignRandaoReveal(1, sk.GetPublicKey().Serialize())

	randaoMsg := &ssv.PartialSignatureMessage{
		Type:             ssv.RandaoPartialSig,
		PartialSignature: randaoSig[:],
		SigningRoot:      root,
		Signers:          []types.OperatorID{id},
	}

	sig, _ := signer.SignRoot(randaoMsg, types.PartialSignatureType, sk.GetPublicKey().Serialize())
	return &ssv.SignedPartialSignatureMessage{
		Message:   randaoMsg,
		Signature: sig,
		Signers:   []types.OperatorID{id},
	}
}
