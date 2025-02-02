package stateutil

import (
	"github.com/pkg/errors"
	"github.com/theQRL/qrysm/v4/container/trie"
	"github.com/theQRL/qrysm/v4/crypto/hash/htr"
	"github.com/theQRL/qrysm/v4/encoding/ssz"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
)

// SyncCommitteeRoot computes the HashTreeRoot Merkleization of a committee root.
// a SyncCommitteeRoot struct according to the eth2
// Simple Serialize specification.
func SyncCommitteeRoot(committee *zondpb.SyncCommittee) ([32]byte, error) {
	var fieldRoots [][32]byte
	if committee == nil {
		return [32]byte{}, nil
	}

	// Field 1:  Vector[BLSPubkey, SYNC_COMMITTEE_SIZE]
	pubKeyRoots := make([][32]byte, 0)
	for _, pubkey := range committee.Pubkeys {
		r, err := merkleizePubkey(pubkey)
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoots = append(pubKeyRoots, r)
	}
	pubkeyRoot, err := ssz.BitwiseMerkleize(pubKeyRoots, uint64(len(pubKeyRoots)), uint64(len(pubKeyRoots)))
	if err != nil {
		return [32]byte{}, err
	}

	// Field 2: BLSPubkey
	aggregateKeyRoot, err := merkleizePubkey(committee.AggregatePubkey)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots = [][32]byte{pubkeyRoot, aggregateKeyRoot}

	return ssz.BitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func merkleizePubkey(pubkey []byte) ([32]byte, error) {
	if len(pubkey) == 0 {
		return [32]byte{}, errors.New("zero length pubkey provided")
	}
	chunks, err := ssz.PackByChunk([][]byte{pubkey})
	if err != nil {
		return [32]byte{}, err
	}

	depth := ssz.Depth(uint64(len(chunks)))
	for i := uint8(0); i < depth; i++ {
		chunkLength := len(chunks)
		oddChunksLen := chunkLength%2 == 1
		if oddChunksLen {
			chunks = append(chunks, trie.ZeroHashes[i])
		}
		outputLen := len(chunks) / 2
		outputChunk := make([][32]byte, outputLen)
		htr.VectorizedSha256(chunks, outputChunk)
		chunks = outputChunk
	}

	return chunks[0], nil
}
