package slasher

import (
	"context"
	"testing"

	"github.com/theQRL/qrysm/v4/beacon-chain/slasher/mock"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
	"github.com/theQRL/qrysm/v4/testing/require"
)

func TestServer_IsSlashableAttestation_SlashingFound(t *testing.T) {
	mockSlasher := &mock.MockSlashingChecker{
		AttesterSlashingFound: true,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableAttestation(ctx, &zondpb.IndexedAttestation{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.AttesterSlashings) > 0)
}

func TestServer_IsSlashableAttestation_SlashingNotFound(t *testing.T) {
	mockSlasher := &mock.MockSlashingChecker{
		AttesterSlashingFound: false,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableAttestation(ctx, &zondpb.IndexedAttestation{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.AttesterSlashings) == 0)
}

func TestServer_IsSlashableBlock_SlashingFound(t *testing.T) {
	mockSlasher := &mock.MockSlashingChecker{
		ProposerSlashingFound: true,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableBlock(ctx, &zondpb.SignedBeaconBlockHeader{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.ProposerSlashings) > 0)
}

func TestServer_IsSlashableBlock_SlashingNotFound(t *testing.T) {
	mockSlasher := &mock.MockSlashingChecker{
		ProposerSlashingFound: false,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableBlock(ctx, &zondpb.SignedBeaconBlockHeader{})
	require.NoError(t, err)
	require.Equal(t, true, len(slashing.ProposerSlashings) == 0)
}
