package provisiond

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/primitives"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// build all api handlers and attach them to the rmb router.
func setupApi(router rmb.Router, cl zbus.Client, engine provision.Engine, store provision.Storage, statistics *primitives.Statistics) error {

	// attach statistics module to rmb
	if err := primitives.NewStatisticsMessageBus(router, statistics); err != nil {
		return errors.Wrap(err, "failed to create statistics api")
	}

	setupStorageRmb(router, cl)
	setupGPURmb(router, store)

	_ = mbus.NewDeploymentMessageBus(router, engine)

	return nil
}

func setupStorageRmb(router rmb.Router, cl zbus.Client) {
	storage := router.Subroute("storage")
	storage.WithHandler("pools", func(ctx context.Context, payload []byte) (interface{}, error) {
		stub := stubs.NewStorageModuleStub(cl)
		return stub.Metrics(ctx)
	})
}

func setupGPURmb(router rmb.Router, store provision.Storage) {
	type Info struct {
		ID       string `json:"id"`
		Vendor   string `json:"vendor"`
		Device   string `json:"device"`
		Contract uint64 `json:"contract"`
	}
	gpus := router.Subroute("gpu")
	usedGpus := func() (map[string]uint64, error) {
		gpus := make(map[string]uint64)
		active, err := store.Capacity()
		if err != nil {
			return nil, err
		}
		for _, dl := range active.Deployments {
			for _, wl := range dl.Workloads {
				if wl.Type != zos.ZMachineType {
					continue
				}
				var vm zos.ZMachine
				if err := json.Unmarshal(wl.Data, &vm); err != nil {
					return nil, errors.Wrapf(err, "invalid workload data (%d.%s)", dl.ContractID, wl.Name)
				}

				for _, gpu := range vm.GPU {
					gpus[string(gpu)] = dl.ContractID
				}
			}
		}
		return gpus, nil
	}
	gpus.WithHandler("list", func(ctx context.Context, payload []byte) (interface{}, error) {
		devices, err := capacity.ListPCI(capacity.GPU)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list available devices")
		}

		if err != nil {
			return nil, errors.Wrap(err, "failed to list active deployments")
		}

		used, err := usedGpus()
		if err != nil {
			return nil, errors.Wrap(err, "failed to list used gpus")
		}

		var list []Info
		for _, pciDevice := range devices {
			id := pciDevice.ShortID()
			info := Info{
				ID:       id,
				Vendor:   "unknown",
				Device:   "unknown",
				Contract: used[id],
			}

			vendor, device, ok := pciDevice.GetDevice()
			if ok {
				info.Vendor = vendor.Name
				info.Device = device.Name
			}

			subdevice, ok := pciDevice.GetSubdevice()
			if ok {
				info.Device = subdevice.Name
			}

			list = append(list, info)
		}

		return list, nil
	})
}
