package validator_client_factory

import (
	"github.com/theQRL/qrysm/v4/config/features"
	beaconApi "github.com/theQRL/qrysm/v4/validator/client/beacon-api"
	grpcApi "github.com/theQRL/qrysm/v4/validator/client/grpc-api"
	"github.com/theQRL/qrysm/v4/validator/client/iface"
	validatorHelpers "github.com/theQRL/qrysm/v4/validator/helpers"
)

func NewSlasherClient(validatorConn validatorHelpers.NodeConnection) iface.SlasherClient {
	grpcClient := grpcApi.NewSlasherClient(validatorConn.GetGrpcClientConn())
	featureFlags := features.Get()

	if featureFlags.EnableBeaconRESTApi {
		return beaconApi.NewSlasherClientWithFallback(validatorConn.GetBeaconApiUrl(), validatorConn.GetBeaconApiTimeout(), grpcClient)
	} else {
		return grpcClient
	}
}
