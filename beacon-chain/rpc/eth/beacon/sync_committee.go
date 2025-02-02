package beacon

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/theQRL/qrysm/v4/api/grpc"
	"github.com/theQRL/qrysm/v4/beacon-chain/core/altair"
	"github.com/theQRL/qrysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/theQRL/qrysm/v4/beacon-chain/state"
	"github.com/theQRL/qrysm/v4/config/params"
	"github.com/theQRL/qrysm/v4/consensus-types/primitives"
	"github.com/theQRL/qrysm/v4/encoding/bytesutil"
	zondpbv2 "github.com/theQRL/qrysm/v4/proto/zond/v2"
	zondpbalpha "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
	"github.com/theQRL/qrysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListSyncCommittees retrieves the sync committees for the given epoch.
// If the epoch is not passed in, then the sync committees for the epoch of the state will be obtained.
func (bs *Server) ListSyncCommittees(ctx context.Context, req *zondpbv2.StateSyncCommitteesRequest) (*zondpbv2.StateSyncCommitteesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListSyncCommittees")
	defer span.End()

	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(currentSlot)
	currentPeriodStartEpoch, err := slots.SyncCommitteePeriodStartEpoch(currentEpoch)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not calculate start period for slot %d: %v",
			currentSlot,
			err,
		)
	}

	requestNextCommittee := false
	if req.Epoch != nil {
		reqPeriodStartEpoch, err := slots.SyncCommitteePeriodStartEpoch(*req.Epoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not calculate start period for epoch %d: %v",
				*req.Epoch,
				err,
			)
		}
		if reqPeriodStartEpoch > currentPeriodStartEpoch+params.BeaconConfig().EpochsPerSyncCommitteePeriod {
			return nil, status.Errorf(
				codes.Internal,
				"Could not fetch sync committee too far in the future. Requested epoch: %d, current epoch: %d",
				*req.Epoch, currentEpoch,
			)
		}
		if reqPeriodStartEpoch > currentPeriodStartEpoch {
			requestNextCommittee = true
			req.Epoch = &currentPeriodStartEpoch
		}
	}

	st, err := bs.stateFromRequest(ctx, &stateRequest{
		epoch:   req.Epoch,
		stateId: req.StateId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch beacon state using request: %v", err)
	}

	var committeeIndices []primitives.ValidatorIndex
	var committee *zondpbalpha.SyncCommittee
	if requestNextCommittee {
		// Get the next sync committee and sync committee indices from the state.
		committeeIndices, committee, err = nextCommitteeIndicesFromState(st)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get next sync committee indices: %v", err)
		}
	} else {
		// Get the current sync committee and sync committee indices from the state.
		committeeIndices, committee, err = currentCommitteeIndicesFromState(st)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get current sync committee indices: %v", err)
		}
	}
	subcommittees, err := extractSyncSubcommittees(st, committee)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not extract sync subcommittees: %v", err)
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &zondpbv2.StateSyncCommitteesResponse{
		Data: &zondpbv2.SyncCommitteeValidators{
			Validators:          committeeIndices,
			ValidatorAggregates: subcommittees,
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}, nil
}

func committeeIndicesFromState(st state.BeaconState, committee *zondpbalpha.SyncCommittee) ([]primitives.ValidatorIndex, *zondpbalpha.SyncCommittee, error) {
	committeeIndices := make([]primitives.ValidatorIndex, len(committee.Pubkeys))
	for i, key := range committee.Pubkeys {
		index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes2592(key))
		if !ok {
			return nil, nil, fmt.Errorf(
				"validator index not found for pubkey %#x",
				bytesutil.Trunc(key),
			)
		}
		committeeIndices[i] = index
	}
	return committeeIndices, committee, nil
}

func currentCommitteeIndicesFromState(st state.BeaconState) ([]primitives.ValidatorIndex, *zondpbalpha.SyncCommittee, error) {
	committee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not get sync committee: %v", err,
		)
	}

	return committeeIndicesFromState(st, committee)
}

func nextCommitteeIndicesFromState(st state.BeaconState) ([]primitives.ValidatorIndex, *zondpbalpha.SyncCommittee, error) {
	committee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not get sync committee: %v", err,
		)
	}

	return committeeIndicesFromState(st, committee)
}

func extractSyncSubcommittees(st state.BeaconState, committee *zondpbalpha.SyncCommittee) ([]*zondpbv2.SyncSubcommitteeValidators, error) {
	subcommitteeCount := params.BeaconConfig().SyncCommitteeSubnetCount
	subcommittees := make([]*zondpbv2.SyncSubcommitteeValidators, subcommitteeCount)
	for i := uint64(0); i < subcommitteeCount; i++ {
		pubkeys, err := altair.SyncSubCommitteePubkeys(committee, primitives.CommitteeIndex(i))
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get subcommittee pubkeys: %v", err,
			)
		}
		subcommittee := &zondpbv2.SyncSubcommitteeValidators{Validators: make([]primitives.ValidatorIndex, len(pubkeys))}
		for j, key := range pubkeys {
			index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes2592(key))
			if !ok {
				return nil, fmt.Errorf(
					"validator index not found for pubkey %#x",
					bytesutil.Trunc(key),
				)
			}
			subcommittee.Validators[j] = index
		}
		subcommittees[i] = subcommittee
	}
	return subcommittees, nil
}

// SubmitPoolSyncCommitteeSignatures submits sync committee signature objects to the node.
func (bs *Server) SubmitPoolSyncCommitteeSignatures(ctx context.Context, req *zondpbv2.SubmitPoolSyncCommitteeSignatures) (*empty.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitPoolSyncCommitteeSignatures")
	defer span.End()

	var validMessages []*zondpbalpha.SyncCommitteeMessage
	var msgFailures []*helpers.SingleIndexedVerificationFailure
	for i, msg := range req.Data {
		if err := validateSyncCommitteeMessage(msg); err != nil {
			msgFailures = append(msgFailures, &helpers.SingleIndexedVerificationFailure{
				Index:   i,
				Message: err.Error(),
			})
			continue
		}

		v1alpha1Msg := &zondpbalpha.SyncCommitteeMessage{
			Slot:           msg.Slot,
			BlockRoot:      msg.BeaconBlockRoot,
			ValidatorIndex: msg.ValidatorIndex,
			Signature:      msg.Signature,
		}
		validMessages = append(validMessages, v1alpha1Msg)
	}

	for _, msg := range validMessages {
		_, err := bs.V1Alpha1ValidatorServer.SubmitSyncMessage(ctx, msg)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not submit message: %v", err)
		}
	}

	if len(msgFailures) > 0 {
		failuresContainer := &helpers.IndexedVerificationFailure{Failures: msgFailures}
		err := grpc.AppendCustomErrorHeader(ctx, failuresContainer)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"One or more messages failed validation. Could not prepare detailed failure information: %v",
				err,
			)
		}
		return nil, status.Errorf(codes.InvalidArgument, "One or more messages failed validation")
	}

	return &empty.Empty{}, nil
}

func validateSyncCommitteeMessage(msg *zondpbv2.SyncCommitteeMessage) error {
	if len(msg.BeaconBlockRoot) != 32 {
		return errors.New("invalid block root length")
	}
	if len(msg.Signature) != 96 {
		return errors.New("invalid signature length")
	}
	return nil
}
