// Package events defines a gRPC events service implementation,
// following the official API standards https://ethereum.github.io/beacon-apis/#/.
// This package includes the events endpoint.
package events

import (
	"context"

	"github.com/theQRL/qrysm/v4/beacon-chain/blockchain"
	blockfeed "github.com/theQRL/qrysm/v4/beacon-chain/core/feed/block"
	opfeed "github.com/theQRL/qrysm/v4/beacon-chain/core/feed/operation"
	statefeed "github.com/theQRL/qrysm/v4/beacon-chain/core/feed/state"
)

// Server defines a server implementation of the gRPC events service,
// providing RPC endpoints to subscribe to events from the beacon node.
type Server struct {
	Ctx               context.Context
	StateNotifier     statefeed.Notifier
	BlockNotifier     blockfeed.Notifier
	OperationNotifier opfeed.Notifier
	HeadFetcher       blockchain.HeadFetcher
	ChainInfoFetcher  blockchain.ChainInfoFetcher
}
