package beacon_api

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/theQRL/go-zond/common/hexutil"
	rpcmiddleware "github.com/theQRL/qrysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/theQRL/qrysm/v4/consensus-types/primitives"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
	"github.com/theQRL/qrysm/v4/testing/assert"
	"github.com/theQRL/qrysm/v4/testing/require"
	"github.com/theQRL/qrysm/v4/validator/client/beacon-api/mock"
)

const stringPubKey = "0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13"

func getPubKeyAndURL(t *testing.T) ([]byte, string) {
	baseUrl := "/eth/v1/beacon/states/head/validators"
	url := fmt.Sprintf("%s?id=%s", baseUrl, stringPubKey)

	pubKey, err := hexutil.Decode(stringPubKey)
	require.NoError(t, err)

	return pubKey, url
}

func TestIndex_Nominal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubKey, url := getPubKeyAndURL(t)
	ctx := context.Background()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:  "55293",
					Status: "active_ongoing",
					Validator: &rpcmiddleware.ValidatorJson{
						PublicKey: stringPubKey,
					},
				},
			},
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{
		stateValidatorsProvider: beaconApiStateValidatorsProvider{
			jsonRestHandler: jsonRestHandler,
		},
	}

	validatorIndex, err := validatorClient.ValidatorIndex(
		ctx,
		&zondpb.ValidatorIndexRequest{
			PublicKey: pubKey,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, primitives.ValidatorIndex(55293), validatorIndex.Index)
}

func TestIndex_UnexistingValidator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubKey, url := getPubKeyAndURL(t)
	ctx := context.Background()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{},
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{
		stateValidatorsProvider: beaconApiStateValidatorsProvider{
			jsonRestHandler: jsonRestHandler,
		},
	}

	_, err := validatorClient.ValidatorIndex(
		ctx,
		&zondpb.ValidatorIndexRequest{
			PublicKey: pubKey,
		},
	)

	wanted := "could not find validator index for public key `0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13`"
	assert.ErrorContains(t, wanted, err)
}

func TestIndex_BadIndexError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubKey, url := getPubKeyAndURL(t)
	ctx := context.Background()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:  "This is not an index",
					Status: "active_ongoing",
					Validator: &rpcmiddleware.ValidatorJson{
						PublicKey: stringPubKey,
					},
				},
			},
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{
		stateValidatorsProvider: beaconApiStateValidatorsProvider{
			jsonRestHandler: jsonRestHandler,
		},
	}

	_, err := validatorClient.ValidatorIndex(
		ctx,
		&zondpb.ValidatorIndexRequest{
			PublicKey: pubKey,
		},
	)

	assert.ErrorContains(t, "failed to parse validator index", err)
}

func TestIndex_JsonResponseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pubKey, url := getPubKeyAndURL(t)
	ctx := context.Background()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		errors.New("some specific json error"),
	).Times(1)

	validatorClient := beaconApiValidatorClient{
		stateValidatorsProvider: beaconApiStateValidatorsProvider{
			jsonRestHandler: jsonRestHandler,
		},
	}

	_, err := validatorClient.ValidatorIndex(
		ctx,
		&zondpb.ValidatorIndexRequest{
			PublicKey: pubKey,
		},
	)

	assert.ErrorContains(t, "failed to get state validator", err)
}
