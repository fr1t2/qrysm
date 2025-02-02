package beacon_api

import (
	"math/big"
	"strconv"

	"github.com/pkg/errors"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/qrysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/theQRL/qrysm/v4/consensus-types/primitives"
	"github.com/theQRL/qrysm/v4/encoding/bytesutil"
	enginev1 "github.com/theQRL/qrysm/v4/proto/engine/v1"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
)

type beaconBlockConverter interface {
	ConvertRESTPhase0BlockToProto(block *apimiddleware.BeaconBlockJson) (*zondpb.BeaconBlock, error)
	ConvertRESTAltairBlockToProto(block *apimiddleware.BeaconBlockAltairJson) (*zondpb.BeaconBlockAltair, error)
	ConvertRESTBellatrixBlockToProto(block *apimiddleware.BeaconBlockBellatrixJson) (*zondpb.BeaconBlockBellatrix, error)
	ConvertRESTCapellaBlockToProto(block *apimiddleware.BeaconBlockCapellaJson) (*zondpb.BeaconBlockCapella, error)
}

type beaconApiBeaconBlockConverter struct{}

// ConvertRESTPhase0BlockToProto converts a Phase0 JSON beacon block to its protobuf equivalent
func (c beaconApiBeaconBlockConverter) ConvertRESTPhase0BlockToProto(block *apimiddleware.BeaconBlockJson) (*zondpb.BeaconBlock, error) {
	blockSlot, err := strconv.ParseUint(block.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse slot `%s`", block.Slot)
	}

	blockProposerIndex, err := strconv.ParseUint(block.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse proposer index `%s`", block.ProposerIndex)
	}

	parentRoot, err := hexutil.Decode(block.ParentRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode parent root `%s`", block.ParentRoot)
	}

	stateRoot, err := hexutil.Decode(block.StateRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode state root `%s`", block.StateRoot)
	}

	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	randaoReveal, err := hexutil.Decode(block.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode randao reveal `%s`", block.Body.RandaoReveal)
	}

	if block.Body.Eth1Data == nil {
		return nil, errors.New("eth1 data is nil")
	}

	depositRoot, err := hexutil.Decode(block.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode deposit root `%s`", block.Body.Eth1Data.DepositRoot)
	}

	depositCount, err := strconv.ParseUint(block.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse deposit count `%s`", block.Body.Eth1Data.DepositCount)
	}

	blockHash, err := hexutil.Decode(block.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode block hash `%s`", block.Body.Eth1Data.BlockHash)
	}

	graffiti, err := hexutil.Decode(block.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode graffiti `%s`", block.Body.Graffiti)
	}

	proposerSlashings, err := convertProposerSlashingsToProto(block.Body.ProposerSlashings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get proposer slashings")
	}

	attesterSlashings, err := convertAttesterSlashingsToProto(block.Body.AttesterSlashings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attester slashings")
	}

	attestations, err := convertAttestationsToProto(block.Body.Attestations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestations")
	}

	deposits, err := convertDepositsToProto(block.Body.Deposits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposits")
	}

	voluntaryExits, err := convertVoluntaryExitsToProto(block.Body.VoluntaryExits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get voluntary exits")
	}

	return &zondpb.BeaconBlock{
		Slot:          primitives.Slot(blockSlot),
		ProposerIndex: primitives.ValidatorIndex(blockProposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &zondpb.BeaconBlockBody{
			RandaoReveal: randaoReveal,
			Eth1Data: &zondpb.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			Deposits:          deposits,
			VoluntaryExits:    voluntaryExits,
		},
	}, nil
}

// ConvertRESTAltairBlockToProto converts an Altair JSON beacon block to its protobuf equivalent
func (c beaconApiBeaconBlockConverter) ConvertRESTAltairBlockToProto(block *apimiddleware.BeaconBlockAltairJson) (*zondpb.BeaconBlockAltair, error) {
	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	// Call convertRESTPhase0BlockToProto to set the phase0 fields because all the error handling and the heavy lifting
	// has already been done
	phase0Block, err := c.ConvertRESTPhase0BlockToProto(&apimiddleware.BeaconBlockJson{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
		Body: &apimiddleware.BeaconBlockBodyJson{
			RandaoReveal:      block.Body.RandaoReveal,
			Eth1Data:          block.Body.Eth1Data,
			Graffiti:          block.Body.Graffiti,
			ProposerSlashings: block.Body.ProposerSlashings,
			AttesterSlashings: block.Body.AttesterSlashings,
			Attestations:      block.Body.Attestations,
			Deposits:          block.Body.Deposits,
			VoluntaryExits:    block.Body.VoluntaryExits,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the phase0 fields of the altair block")
	}

	if block.Body.SyncAggregate == nil {
		return nil, errors.New("sync aggregate is nil")
	}

	syncCommitteeBits, err := hexutil.Decode(block.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode sync committee bits `%s`", block.Body.SyncAggregate.SyncCommitteeBits)
	}

	syncCommitteeSignature, err := hexutil.Decode(block.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode sync committee signature `%s`", block.Body.SyncAggregate.SyncCommitteeSignature)
	}

	return &zondpb.BeaconBlockAltair{
		Slot:          phase0Block.Slot,
		ProposerIndex: phase0Block.ProposerIndex,
		ParentRoot:    phase0Block.ParentRoot,
		StateRoot:     phase0Block.StateRoot,
		Body: &zondpb.BeaconBlockBodyAltair{
			RandaoReveal:      phase0Block.Body.RandaoReveal,
			Eth1Data:          phase0Block.Body.Eth1Data,
			Graffiti:          phase0Block.Body.Graffiti,
			ProposerSlashings: phase0Block.Body.ProposerSlashings,
			AttesterSlashings: phase0Block.Body.AttesterSlashings,
			Attestations:      phase0Block.Body.Attestations,
			Deposits:          phase0Block.Body.Deposits,
			VoluntaryExits:    phase0Block.Body.VoluntaryExits,
			SyncAggregate: &zondpb.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSignature,
			},
		},
	}, nil
}

// ConvertRESTBellatrixBlockToProto converts a Bellatrix JSON beacon block to its protobuf equivalent
func (c beaconApiBeaconBlockConverter) ConvertRESTBellatrixBlockToProto(block *apimiddleware.BeaconBlockBellatrixJson) (*zondpb.BeaconBlockBellatrix, error) {
	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	// Call convertRESTAltairBlockToProto to set the altair fields because all the error handling and the heavy lifting
	// has already been done
	altairBlock, err := c.ConvertRESTAltairBlockToProto(&apimiddleware.BeaconBlockAltairJson{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
		Body: &apimiddleware.BeaconBlockBodyAltairJson{
			RandaoReveal:      block.Body.RandaoReveal,
			Eth1Data:          block.Body.Eth1Data,
			Graffiti:          block.Body.Graffiti,
			ProposerSlashings: block.Body.ProposerSlashings,
			AttesterSlashings: block.Body.AttesterSlashings,
			Attestations:      block.Body.Attestations,
			Deposits:          block.Body.Deposits,
			VoluntaryExits:    block.Body.VoluntaryExits,
			SyncAggregate:     block.Body.SyncAggregate,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the altair fields of the bellatrix block")
	}

	if block.Body.ExecutionPayload == nil {
		return nil, errors.New("execution payload is nil")
	}

	parentHash, err := hexutil.Decode(block.Body.ExecutionPayload.ParentHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload parent hash `%s`", block.Body.ExecutionPayload.ParentHash)
	}

	feeRecipient, err := hexutil.Decode(block.Body.ExecutionPayload.FeeRecipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload fee recipient `%s`", block.Body.ExecutionPayload.FeeRecipient)
	}

	stateRoot, err := hexutil.Decode(block.Body.ExecutionPayload.StateRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload state root `%s`", block.Body.ExecutionPayload.StateRoot)
	}

	receiptsRoot, err := hexutil.Decode(block.Body.ExecutionPayload.ReceiptsRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload receipts root `%s`", block.Body.ExecutionPayload.ReceiptsRoot)
	}

	logsBloom, err := hexutil.Decode(block.Body.ExecutionPayload.LogsBloom)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload logs bloom `%s`", block.Body.ExecutionPayload.LogsBloom)
	}

	prevRandao, err := hexutil.Decode(block.Body.ExecutionPayload.PrevRandao)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload prev randao `%s`", block.Body.ExecutionPayload.PrevRandao)
	}

	blockNumber, err := strconv.ParseUint(block.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload block number `%s`", block.Body.ExecutionPayload.BlockNumber)
	}

	gasLimit, err := strconv.ParseUint(block.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload gas limit `%s`", block.Body.ExecutionPayload.GasLimit)
	}

	gasUsed, err := strconv.ParseUint(block.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload gas used `%s`", block.Body.ExecutionPayload.GasUsed)
	}

	timestamp, err := strconv.ParseUint(block.Body.ExecutionPayload.TimeStamp, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload timestamp `%s`", block.Body.ExecutionPayload.TimeStamp)
	}

	extraData, err := hexutil.Decode(block.Body.ExecutionPayload.ExtraData)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload extra data `%s`", block.Body.ExecutionPayload.ExtraData)
	}

	baseFeePerGas := new(big.Int)
	if _, ok := baseFeePerGas.SetString(block.Body.ExecutionPayload.BaseFeePerGas, 10); !ok {
		return nil, errors.Errorf("failed to parse execution payload base fee per gas `%s`", block.Body.ExecutionPayload.BaseFeePerGas)
	}

	blockHash, err := hexutil.Decode(block.Body.ExecutionPayload.BlockHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload block hash `%s`", block.Body.ExecutionPayload.BlockHash)
	}

	transactions, err := convertTransactionsToProto(block.Body.ExecutionPayload.Transactions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get execution payload transactions")
	}

	return &zondpb.BeaconBlockBellatrix{
		Slot:          altairBlock.Slot,
		ProposerIndex: altairBlock.ProposerIndex,
		ParentRoot:    altairBlock.ParentRoot,
		StateRoot:     altairBlock.StateRoot,
		Body: &zondpb.BeaconBlockBodyBellatrix{
			RandaoReveal:      altairBlock.Body.RandaoReveal,
			Eth1Data:          altairBlock.Body.Eth1Data,
			Graffiti:          altairBlock.Body.Graffiti,
			ProposerSlashings: altairBlock.Body.ProposerSlashings,
			AttesterSlashings: altairBlock.Body.AttesterSlashings,
			Attestations:      altairBlock.Body.Attestations,
			Deposits:          altairBlock.Body.Deposits,
			VoluntaryExits:    altairBlock.Body.VoluntaryExits,
			SyncAggregate:     altairBlock.Body.SyncAggregate,
			ExecutionPayload: &enginev1.ExecutionPayload{
				ParentHash:    parentHash,
				FeeRecipient:  feeRecipient,
				StateRoot:     stateRoot,
				ReceiptsRoot:  receiptsRoot,
				LogsBloom:     logsBloom,
				PrevRandao:    prevRandao,
				BlockNumber:   blockNumber,
				GasLimit:      gasLimit,
				GasUsed:       gasUsed,
				Timestamp:     timestamp,
				ExtraData:     extraData,
				BaseFeePerGas: bytesutil.PadTo(bytesutil.BigIntToLittleEndianBytes(baseFeePerGas), 32),
				BlockHash:     blockHash,
				Transactions:  transactions,
			},
		},
	}, nil
}

// ConvertRESTCapellaBlockToProto converts a Capella JSON beacon block to its protobuf equivalent
func (c beaconApiBeaconBlockConverter) ConvertRESTCapellaBlockToProto(block *apimiddleware.BeaconBlockCapellaJson) (*zondpb.BeaconBlockCapella, error) {
	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	if block.Body.ExecutionPayload == nil {
		return nil, errors.New("execution payload is nil")
	}

	// Call convertRESTBellatrixBlockToProto to set the bellatrix fields because all the error handling and the heavy
	// lifting has already been done
	bellatrixBlock, err := c.ConvertRESTBellatrixBlockToProto(&apimiddleware.BeaconBlockBellatrixJson{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
		Body: &apimiddleware.BeaconBlockBodyBellatrixJson{
			RandaoReveal:      block.Body.RandaoReveal,
			Eth1Data:          block.Body.Eth1Data,
			Graffiti:          block.Body.Graffiti,
			ProposerSlashings: block.Body.ProposerSlashings,
			AttesterSlashings: block.Body.AttesterSlashings,
			Attestations:      block.Body.Attestations,
			Deposits:          block.Body.Deposits,
			VoluntaryExits:    block.Body.VoluntaryExits,
			SyncAggregate:     block.Body.SyncAggregate,
			ExecutionPayload: &apimiddleware.ExecutionPayloadJson{
				ParentHash:    block.Body.ExecutionPayload.ParentHash,
				FeeRecipient:  block.Body.ExecutionPayload.FeeRecipient,
				StateRoot:     block.Body.ExecutionPayload.StateRoot,
				ReceiptsRoot:  block.Body.ExecutionPayload.ReceiptsRoot,
				LogsBloom:     block.Body.ExecutionPayload.LogsBloom,
				PrevRandao:    block.Body.ExecutionPayload.PrevRandao,
				BlockNumber:   block.Body.ExecutionPayload.BlockNumber,
				GasLimit:      block.Body.ExecutionPayload.GasLimit,
				GasUsed:       block.Body.ExecutionPayload.GasUsed,
				TimeStamp:     block.Body.ExecutionPayload.TimeStamp,
				ExtraData:     block.Body.ExecutionPayload.ExtraData,
				BaseFeePerGas: block.Body.ExecutionPayload.BaseFeePerGas,
				BlockHash:     block.Body.ExecutionPayload.BlockHash,
				Transactions:  block.Body.ExecutionPayload.Transactions,
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the bellatrix fields of the capella block")
	}

	withdrawals, err := convertWithdrawalsToProto(block.Body.ExecutionPayload.Withdrawals)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get withdrawals")
	}

	dilithiumToExecutionChanges, err := convertDilithiumToExecutionChangesToProto(block.Body.DilithiumToExecutionChanges)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dilithium to execution changes")
	}

	return &zondpb.BeaconBlockCapella{
		Slot:          bellatrixBlock.Slot,
		ProposerIndex: bellatrixBlock.ProposerIndex,
		ParentRoot:    bellatrixBlock.ParentRoot,
		StateRoot:     bellatrixBlock.StateRoot,
		Body: &zondpb.BeaconBlockBodyCapella{
			RandaoReveal:      bellatrixBlock.Body.RandaoReveal,
			Eth1Data:          bellatrixBlock.Body.Eth1Data,
			Graffiti:          bellatrixBlock.Body.Graffiti,
			ProposerSlashings: bellatrixBlock.Body.ProposerSlashings,
			AttesterSlashings: bellatrixBlock.Body.AttesterSlashings,
			Attestations:      bellatrixBlock.Body.Attestations,
			Deposits:          bellatrixBlock.Body.Deposits,
			VoluntaryExits:    bellatrixBlock.Body.VoluntaryExits,
			SyncAggregate:     bellatrixBlock.Body.SyncAggregate,
			ExecutionPayload: &enginev1.ExecutionPayloadCapella{
				ParentHash:    bellatrixBlock.Body.ExecutionPayload.ParentHash,
				FeeRecipient:  bellatrixBlock.Body.ExecutionPayload.FeeRecipient,
				StateRoot:     bellatrixBlock.Body.ExecutionPayload.StateRoot,
				ReceiptsRoot:  bellatrixBlock.Body.ExecutionPayload.ReceiptsRoot,
				LogsBloom:     bellatrixBlock.Body.ExecutionPayload.LogsBloom,
				PrevRandao:    bellatrixBlock.Body.ExecutionPayload.PrevRandao,
				BlockNumber:   bellatrixBlock.Body.ExecutionPayload.BlockNumber,
				GasLimit:      bellatrixBlock.Body.ExecutionPayload.GasLimit,
				GasUsed:       bellatrixBlock.Body.ExecutionPayload.GasUsed,
				Timestamp:     bellatrixBlock.Body.ExecutionPayload.Timestamp,
				ExtraData:     bellatrixBlock.Body.ExecutionPayload.ExtraData,
				BaseFeePerGas: bellatrixBlock.Body.ExecutionPayload.BaseFeePerGas,
				BlockHash:     bellatrixBlock.Body.ExecutionPayload.BlockHash,
				Transactions:  bellatrixBlock.Body.ExecutionPayload.Transactions,
				Withdrawals:   withdrawals,
			},
			DilithiumToExecutionChanges: dilithiumToExecutionChanges,
		},
	}, nil
}
