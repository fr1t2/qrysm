package validator

import (
	"bytes"
	"context"
	"sort"

	"github.com/pkg/errors"
	dilithium2 "github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/qrysm/v4/beacon-chain/core/feed"
	opfeed "github.com/theQRL/qrysm/v4/beacon-chain/core/feed/operation"
	"github.com/theQRL/qrysm/v4/config/params"
	"github.com/theQRL/qrysm/v4/consensus-types/primitives"
	"github.com/theQRL/qrysm/v4/crypto/dilithium"
	"github.com/theQRL/qrysm/v4/encoding/bytesutil"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetSyncMessageBlockRoot retrieves the sync committee block root of the beacon chain.
func (vs *Server) GetSyncMessageBlockRoot(
	ctx context.Context, _ *emptypb.Empty,
) (*zondpb.SyncMessageBlockRootResponse, error) {
	// An optimistic validator MUST NOT participate in sync committees
	// (i.e., sign across the DOMAIN_SYNC_COMMITTEE, DOMAIN_SYNC_COMMITTEE_SELECTION_PROOF or DOMAIN_CONTRIBUTION_AND_PROOF domains).
	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, err
	}

	r, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
	}

	return &zondpb.SyncMessageBlockRootResponse{
		Root: r,
	}, nil
}

// SubmitSyncMessage submits the sync committee message to the network.
// It also saves the sync committee message into the pending pool for block inclusion.
func (vs *Server) SubmitSyncMessage(ctx context.Context, msg *zondpb.SyncCommitteeMessage) (*emptypb.Empty, error) {
	errs, ctx := errgroup.WithContext(ctx)

	headSyncCommitteeIndices, err := vs.HeadFetcher.HeadSyncCommitteeIndices(ctx, msg.ValidatorIndex, msg.Slot)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	// Broadcasting and saving message into the pool in parallel. As one fail should not affect another.
	// This broadcasts for all subnets.
	for _, index := range headSyncCommitteeIndices {
		subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
		subnet := uint64(index) / subCommitteeSize
		errs.Go(func() error {
			return vs.P2P.BroadcastSyncCommitteeMessage(ctx, subnet, msg)
		})
	}

	if err := vs.SyncCommitteePool.SaveSyncCommitteeMessage(msg); err != nil {
		return &emptypb.Empty{}, err
	}

	// Wait for p2p broadcast to complete and return the first error (if any)
	err = errs.Wait()
	return &emptypb.Empty{}, err
}

// GetSyncSubcommitteeIndex is called by a sync committee participant to get
// its subcommittee index for sync message aggregation duty.
func (vs *Server) GetSyncSubcommitteeIndex(
	ctx context.Context, req *zondpb.SyncSubcommitteeIndexRequest,
) (*zondpb.SyncSubcommitteeIndexResponse, error) {
	index, exists := vs.HeadFetcher.HeadPublicKeyToValidatorIndex(bytesutil.ToBytes2592(req.PublicKey))
	if !exists {
		return nil, errors.New("public key does not exist in state")
	}
	indices, err := vs.HeadFetcher.HeadSyncCommitteeIndices(ctx, index, req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee index: %v", err)
	}
	return &zondpb.SyncSubcommitteeIndexResponse{Indices: indices}, nil
}

// GetSyncCommitteeContribution is called by a sync committee aggregator
// to retrieve sync committee contribution object.
func (vs *Server) GetSyncCommitteeContribution(
	ctx context.Context, req *zondpb.SyncCommitteeContributionRequest,
) (*zondpb.SyncCommitteeContribution, error) {
	// An optimistic validator MUST NOT participate in sync committees
	// (i.e., sign across the DOMAIN_SYNC_COMMITTEE, DOMAIN_SYNC_COMMITTEE_SELECTION_PROOF or DOMAIN_CONTRIBUTION_AND_PROOF domains).
	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, err
	}

	msgs, err := vs.SyncCommitteePool.SyncCommitteeMessages(req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee messages: %v", err)
	}
	headRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head root: %v", err)
	}
	aggregatedSig, bits, err := vs.aggregatedSigAndAggregationBits(ctx, msgs, req.Slot, req.SubnetId, headRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get contribution data: %v", err)
	}
	contribution := &zondpb.SyncCommitteeContribution{
		Slot:              req.Slot,
		BlockRoot:         headRoot,
		SubcommitteeIndex: req.SubnetId,
		AggregationBits:   bits,
		Signature:         aggregatedSig,
	}

	return contribution, nil
}

// SubmitSignedContributionAndProof is called by a sync committee aggregator
// to submit signed contribution and proof object.
func (vs *Server) SubmitSignedContributionAndProof(
	ctx context.Context, s *zondpb.SignedContributionAndProof,
) (*emptypb.Empty, error) {
	errs, ctx := errgroup.WithContext(ctx)

	// Broadcasting and saving contribution into the pool in parallel. As one fail should not affect another.
	errs.Go(func() error {
		return vs.P2P.Broadcast(ctx, s)
	})

	if err := vs.SyncCommitteePool.SaveSyncCommitteeContribution(s.Message.Contribution); err != nil {
		return nil, err
	}

	// Wait for p2p broadcast to complete and return the first error (if any)
	err := errs.Wait()

	if err == nil {
		vs.OperationNotifier.OperationFeed().Send(&feed.Event{
			Type: opfeed.SyncCommitteeContributionReceived,
			Data: &opfeed.SyncCommitteeContributionReceivedData{
				Contribution: s,
			},
		})
	}

	return &emptypb.Empty{}, err
}

// AggregatedSigAndAggregationBits returns the aggregated signature and aggregation bits
// associated with a particular set of sync committee messages.
func (vs *Server) AggregatedSigAndAggregationBits(
	ctx context.Context,
	req *zondpb.AggregatedSigAndAggregationBitsRequest,
) (*zondpb.AggregatedSigAndAggregationBitsResponse, error) {
	aggregatedSig, bits, err := vs.aggregatedSigAndAggregationBits(ctx, req.Msgs, req.Slot, req.SubnetId, req.BlockRoot)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &zondpb.AggregatedSigAndAggregationBitsResponse{AggregatedSig: aggregatedSig, Bits: bits}, nil
}

func (vs *Server) aggregatedSigAndAggregationBits(
	ctx context.Context,
	msgs []*zondpb.SyncCommitteeMessage,
	slot primitives.Slot,
	subnetId uint64,
	blockRoot []byte,
) ([]byte, []byte, error) {
	subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
	sigs := make([][]byte, 0, subCommitteeSize)
	bits := zondpb.NewSyncCommitteeAggregationBits()
	syncCommitteeIndicesSigMap := make(map[primitives.CommitteeIndex]*zondpb.SyncCommitteeMessage)
	appendedSyncCommitteeIndices := make([]primitives.CommitteeIndex, 0)

	for _, msg := range msgs {
		if bytes.Equal(blockRoot, msg.BlockRoot) {
			headSyncCommitteeIndices, err := vs.HeadFetcher.HeadSyncCommitteeIndices(ctx, msg.ValidatorIndex, slot)
			if err != nil {
				return []byte{}, nil, errors.Wrapf(err, "could not get sync subcommittee index")
			}
			for _, index := range headSyncCommitteeIndices {
				i := uint64(index)
				subnetIndex := i / subCommitteeSize
				indexMod := i % subCommitteeSize
				if subnetIndex == subnetId && !bits.BitAt(indexMod) {
					bits.SetBitAt(indexMod, true)
					syncCommitteeIndicesSigMap[index] = msg
					appendedSyncCommitteeIndices = append(appendedSyncCommitteeIndices, index)
				}
			}
		}
	}

	sort.Slice(appendedSyncCommitteeIndices, func(i, j int) bool {
		return appendedSyncCommitteeIndices[i] < appendedSyncCommitteeIndices[j]
	})

	for _, syncCommitteeIndex := range appendedSyncCommitteeIndices {
		msg, ok := syncCommitteeIndicesSigMap[syncCommitteeIndex]
		if !ok {
			return []byte{}, nil, errors.Errorf("could not get sync subcommittee index %d "+
				"in syncCommitteeIndicesSigMap", syncCommitteeIndex)
		}
		sigs = append(sigs, msg.Signature)
	}

	aggregatedSig := make([]byte, dilithium2.CryptoBytes)
	aggregatedSig[0] = 0xC0
	if len(sigs) != 0 {
		uncompressedSigs, err := dilithium.MultipleSignaturesFromBytes(sigs)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not decompress signatures")
		}
		aggregatedSig = dilithium.UnaggregatedSignatures(uncompressedSigs)
	}

	return aggregatedSig, bits, nil
}
