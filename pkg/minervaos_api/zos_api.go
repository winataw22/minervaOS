package zosapi

import (
	"fmt"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/diagnostics"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

type ZosAPI struct {
	oracle                 *capacity.ResourceOracle
	versionMonitorStub     *stubs.VersionMonitorStub
	provisionStub          *stubs.ProvisionStub
	networkerStub          *stubs.NetworkerStub
	statisticsStub         *stubs.StatisticsStub
	storageStub            *stubs.StorageModuleStub
	performanceMonitorStub *stubs.PerformanceMonitorStub
	diagnosticsManager     *diagnostics.DiagnosticsManager
	farmerID               uint32
}

func NewZosAPI(manager substrate.Manager, client zbus.Client, msgBrokerCon string) (ZosAPI, error) {
	sub, err := manager.Substrate()
	if err != nil {
		return ZosAPI{}, err
	}
	defer sub.Close()
	diagnosticsManager, err := diagnostics.NewDiagnosticsManager(msgBrokerCon, client)
	if err != nil {
		return ZosAPI{}, err
	}
	storageModuleStub := stubs.NewStorageModuleStub(client)
	api := ZosAPI{
		oracle:                 capacity.NewResourceOracle(storageModuleStub),
		versionMonitorStub:     stubs.NewVersionMonitorStub(client),
		provisionStub:          stubs.NewProvisionStub(client),
		networkerStub:          stubs.NewNetworkerStub(client),
		statisticsStub:         stubs.NewStatisticsStub(client),
		storageStub:            storageModuleStub,
		performanceMonitorStub: stubs.NewPerformanceMonitorStub(client),
		diagnosticsManager:     diagnosticsManager,
	}
	farm, err := sub.GetFarm(uint32(environment.MustGet().FarmID))
	if err != nil {
		return ZosAPI{}, fmt.Errorf("failed to get farm: %w", err)
	}

	farmer, err := sub.GetTwin(uint32(farm.TwinID))
	if err != nil {
		return ZosAPI{}, err
	}
	api.farmerID = uint32(farmer.ID)
	return api, nil
}
