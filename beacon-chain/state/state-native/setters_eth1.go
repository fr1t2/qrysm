package state_native

import (
	"github.com/theQRL/qrysm/v4/beacon-chain/state/state-native/types"
	"github.com/theQRL/qrysm/v4/beacon-chain/state/stateutil"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
)

// SetEth1Data for the beacon state.
func (b *BeaconState) SetEth1Data(val *zondpb.Eth1Data) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.eth1Data = val
	b.markFieldAsDirty(types.Eth1Data)
	return nil
}

// SetEth1DataVotes for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetEth1DataVotes(val []*zondpb.Eth1Data) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.Eth1DataVotes].MinusRef()
	b.sharedFieldReferences[types.Eth1DataVotes] = stateutil.NewRef(1)

	b.eth1DataVotes = val
	b.markFieldAsDirty(types.Eth1DataVotes)
	b.rebuildTrie[types.Eth1DataVotes] = true
	return nil
}

// SetEth1DepositIndex for the beacon state.
func (b *BeaconState) SetEth1DepositIndex(val uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.eth1DepositIndex = val
	b.markFieldAsDirty(types.Eth1DepositIndex)
	return nil
}

// AppendEth1DataVotes for the beacon state. Appends the new value
// to the end of list.
func (b *BeaconState) AppendEth1DataVotes(val *zondpb.Eth1Data) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	votes := b.eth1DataVotes
	if b.sharedFieldReferences[types.Eth1DataVotes].Refs() > 1 {
		// Copy elements in underlying array by reference.
		votes = make([]*zondpb.Eth1Data, len(b.eth1DataVotes))
		copy(votes, b.eth1DataVotes)
		b.sharedFieldReferences[types.Eth1DataVotes].MinusRef()
		b.sharedFieldReferences[types.Eth1DataVotes] = stateutil.NewRef(1)
	}

	b.eth1DataVotes = append(votes, val)
	b.markFieldAsDirty(types.Eth1DataVotes)
	b.addDirtyIndices(types.Eth1DataVotes, []uint64{uint64(len(b.eth1DataVotes) - 1)})
	return nil
}
