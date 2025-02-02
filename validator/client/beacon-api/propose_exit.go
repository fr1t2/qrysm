package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/qrysm/v4/beacon-chain/rpc/apimiddleware"
	zondpb "github.com/theQRL/qrysm/v4/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) proposeExit(ctx context.Context, signedVoluntaryExit *zondpb.SignedVoluntaryExit) (*zondpb.ProposeExitResponse, error) {
	if signedVoluntaryExit == nil {
		return nil, errors.New("signed voluntary exit is nil")
	}

	if signedVoluntaryExit.Exit == nil {
		return nil, errors.New("exit is nil")
	}

	jsonSignedVoluntaryExit := apimiddleware.SignedVoluntaryExitJson{
		Exit: &apimiddleware.VoluntaryExitJson{
			Epoch:          strconv.FormatUint(uint64(signedVoluntaryExit.Exit.Epoch), 10),
			ValidatorIndex: strconv.FormatUint(uint64(signedVoluntaryExit.Exit.ValidatorIndex), 10),
		},
		Signature: hexutil.Encode(signedVoluntaryExit.Signature),
	}

	marshalledSignedVoluntaryExit, err := json.Marshal(jsonSignedVoluntaryExit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal signed voluntary exit")
	}

	if _, err := c.jsonRestHandler.PostRestJson(ctx, "/eth/v1/beacon/pool/voluntary_exits", nil, bytes.NewBuffer(marshalledSignedVoluntaryExit), nil); err != nil {
		return nil, errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	exitRoot, err := signedVoluntaryExit.Exit.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute exit root")
	}

	return &zondpb.ProposeExitResponse{
		ExitRoot: exitRoot[:],
	}, nil
}
