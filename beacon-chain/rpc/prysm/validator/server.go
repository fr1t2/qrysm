package validator

import (
	"github.com/theQRL/qrysm/v4/beacon-chain/blockchain"
	"github.com/theQRL/qrysm/v4/beacon-chain/sync"
)

// Server defines a server implementation for HTTP endpoints, providing
// access data relevant to the Ethereum Beacon Chain.
type Server struct {
	GenesisTimeFetcher blockchain.TimeFetcher
	SyncChecker        sync.Checker
	HeadFetcher        blockchain.HeadFetcher
}
