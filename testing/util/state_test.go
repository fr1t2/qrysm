package util

import (
	"context"
	"reflect"
	"testing"

	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
	"github.com/theQRL/qrysm/v4/testing/require"
)

func TestNewBeaconState(t *testing.T) {
	st, err := NewBeaconState()
	require.NoError(t, err)
	b, err := st.MarshalSSZ()
	require.NoError(t, err)
	got := &zondpb.BeaconState{}
	require.NoError(t, got.UnmarshalSSZ(b))
	if !reflect.DeepEqual(st.ToProtoUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}

func TestNewBeaconStateAltair(t *testing.T) {
	st, err := NewBeaconStateAltair()
	require.NoError(t, err)
	b, err := st.MarshalSSZ()
	require.NoError(t, err)
	got := &zondpb.BeaconStateAltair{}
	require.NoError(t, got.UnmarshalSSZ(b))
	if !reflect.DeepEqual(st.ToProtoUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}

func TestNewBeaconStateBellatrix(t *testing.T) {
	st, err := NewBeaconStateBellatrix()
	require.NoError(t, err)
	b, err := st.MarshalSSZ()
	require.NoError(t, err)
	got := &zondpb.BeaconStateBellatrix{}
	require.NoError(t, got.UnmarshalSSZ(b))
	if !reflect.DeepEqual(st.ToProtoUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}

func TestNewBeaconStateCapella(t *testing.T) {
	st, err := NewBeaconStateCapella()
	require.NoError(t, err)
	b, err := st.MarshalSSZ()
	require.NoError(t, err)
	got := &zondpb.BeaconStateCapella{}
	require.NoError(t, got.UnmarshalSSZ(b))
	if !reflect.DeepEqual(st.ToProtoUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}

func TestNewBeaconState_HashTreeRoot(t *testing.T) {
	st, err := NewBeaconState()
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	st, err = NewBeaconStateAltair()
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	st, err = NewBeaconStateBellatrix()
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	st, err = NewBeaconStateCapella()
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
}
