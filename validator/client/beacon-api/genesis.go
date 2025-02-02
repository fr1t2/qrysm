package beacon_api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/qrysm/v4/api/gateway/apimiddleware"
	rpcmiddleware "github.com/theQRL/qrysm/v4/beacon-chain/rpc/apimiddleware"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
)

type genesisProvider interface {
	GetGenesis(ctx context.Context) (*rpcmiddleware.GenesisResponse_GenesisJson, *apimiddleware.DefaultErrorJson, error)
}

type beaconApiGenesisProvider struct {
	jsonRestHandler jsonRestHandler
}

func (c beaconApiValidatorClient) waitForChainStart(ctx context.Context) (*zondpb.ChainStartResponse, error) {
	genesis, httpError, err := c.genesisProvider.GetGenesis(ctx)

	for err != nil {
		if httpError == nil || httpError.Code != http.StatusNotFound {
			return nil, errors.Wrap(err, "failed to get genesis data")
		}

		// Error 404 means that the chain genesis info is not yet known, so we query it every second until it's ready
		select {
		case <-time.After(time.Second):
			genesis, httpError, err = c.genesisProvider.GetGenesis(ctx)
		case <-ctx.Done():
			return nil, errors.New("context canceled")
		}
	}

	genesisTime, err := strconv.ParseUint(genesis.GenesisTime, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse genesis time: %s", genesis.GenesisTime)
	}

	chainStartResponse := &zondpb.ChainStartResponse{}
	chainStartResponse.Started = true
	chainStartResponse.GenesisTime = genesisTime

	if !validRoot(genesis.GenesisValidatorsRoot) {
		return nil, errors.Errorf("invalid genesis validators root: %s", genesis.GenesisValidatorsRoot)
	}

	genesisValidatorRoot, err := hexutil.Decode(genesis.GenesisValidatorsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode genesis validators root")
	}
	chainStartResponse.GenesisValidatorsRoot = genesisValidatorRoot

	return chainStartResponse, nil
}

// GetGenesis gets the genesis information from the beacon node via the /eth/v1/beacon/genesis endpoint
func (c beaconApiGenesisProvider) GetGenesis(ctx context.Context) (*rpcmiddleware.GenesisResponse_GenesisJson, *apimiddleware.DefaultErrorJson, error) {
	genesisJson := &rpcmiddleware.GenesisResponseJson{}
	errorJson, err := c.jsonRestHandler.GetRestJsonResponse(ctx, "/eth/v1/beacon/genesis", genesisJson)
	if err != nil {
		return nil, errorJson, errors.Wrap(err, "failed to get json response")
	}

	if genesisJson.Data == nil {
		return nil, nil, errors.New("genesis data is nil")
	}

	return genesisJson.Data, nil, nil
}
