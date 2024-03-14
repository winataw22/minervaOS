package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	cloudContainerFlist = "https://hub.grid.tf/tf-autobuilder/cloud-container-9dba60e.flist"
	cloudContainerName  = "cloud-container"
)

// ZMachine type
type ZMachine = zos.ZMachine

var (
	_ provision.Manager     = (*Manager)(nil)
	_ provision.Initializer = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (m *Manager) Initialize(ctx context.Context) error {
	return m.initGPUs()
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, wl)
}

func (p *Manager) vmMounts(ctx context.Context, deployment *gridtypes.Deployment, mounts []zos.MachineMount, format bool, vm *pkg.VM) error {
	for _, mount := range mounts {
		wl, err := deployment.Get(mount.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to get mount '%s' workload", mount.Name)
		}
		if wl.Result.State != gridtypes.StateOk {
			return fmt.Errorf("invalid disk '%s' state", mount.Name)
		}
		switch wl.Type {
		case zos.ZMountType:
			if err := p.mountDisk(ctx, wl, mount, format, vm); err != nil {
				return err
			}
		case zos.QuantumSafeFSType:
			if err := p.mountQsfs(wl, mount, vm); err != nil {
				return err
			}
		default:
			return fmt.Errorf("expecting a reservation of type '%s' or '%s' for disk '%s'", zos.ZMountType, zos.QuantumSafeFSType, mount.Name)
		}
	}
	return nil
}

func (p *Manager) mountDisk(ctx context.Context, wl *gridtypes.WorkloadWithID, mount zos.MachineMount, format bool, vm *pkg.VM) error {
	storage := stubs.NewStorageModuleStub(p.zbus)

	info, err := storage.DiskLookup(ctx, wl.ID.String())
	if err != nil {
		return errors.Wrapf(err, "failed to inspect disk '%s'", mount.Name)
	}

	if format {
		if err := storage.DiskFormat(ctx, wl.ID.String()); err != nil {
			return errors.Wrap(err, "failed to prepare mount")
		}
	}

	vm.Disks = append(vm.Disks, pkg.VMDisk{Path: info.Path, Target: mount.Mountpoint})

	return nil
}

func (p *Manager) mountQsfs(wl *gridtypes.WorkloadWithID, mount zos.MachineMount, vm *pkg.VM) error {

	var info zos.QuatumSafeFSResult
	if err := wl.Result.Unmarshal(&info); err != nil {
		return fmt.Errorf("invalid qsfs result '%s': %w", mount.Name, err)
	}

	vm.Shared = append(vm.Shared, pkg.SharedDir{ID: wl.Name.String(), Path: info.Path, Target: mount.Mountpoint})
	return nil
}

func (p *Manager) virtualMachineProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.ZMachineResult, err error) {
	var (
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config ZMachine
	)
	if vm.Exists(ctx, wl.ID.String()) {
		return result, provision.ErrNoActionNeeded
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	if len(config.GPU) != 0 && !provision.IsRentedNode(ctx) {
		// you cannot use GPU unless this is a rented node
		return result, fmt.Errorf("usage of GPU is not allowed unless node is rented")
	}

	machine := pkg.VM{
		Name:       wl.ID.String(),
		CPU:        config.ComputeCapacity.CPU,
		Memory:     config.ComputeCapacity.Memory,
		Entrypoint: config.Entrypoint,
		KernelArgs: pkg.KernelArgs{},
	}

	// expand GPUs
	devices, err := p.expandGPUs(config.GPU)
	if err != nil {
		return result, errors.Wrap(err, "failed to prepare requested gpu device(s)")
	}

	for _, device := range devices {
		machine.Devices = append(machine.Devices, device.Slot)
	}

	// the config is validated by the engine. we now only support only one
	// private network
	if len(config.Network.Interfaces) != 1 {
		return result, fmt.Errorf("only one private network is support")
	}
	netConfig := config.Network.Interfaces[0]

	// check if public ipv4 is supported, should this be requested
	if !config.Network.PublicIP.IsEmpty() && !network.PublicIPv4Support(ctx) {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = wl.ID.String()
	result.IP = netConfig.IP.String()

	deployment, err := provision.GetDeployment(ctx)
	if err != nil {
		return result, errors.Wrap(err, "failed to get deployment")
	}
	networkInfo := pkg.VMNetworkInfo{
		Nameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1"), net.ParseIP("2001:4860:4860::8888")},
	}

	var ifs []string
	var pubIf string

	defer func() {
		if err != nil {
			for _, nic := range ifs {
				_ = network.RemoveTap(ctx, nic)
			}
			if pubIf != "" {
				_ = network.DisconnectPubTap(ctx, pubIf)
			}
		}
	}()

	for _, nic := range config.Network.Interfaces {
		inf, err := p.newPrivNetworkInterface(ctx, deployment, wl, nic)
		if err != nil {
			return result, err
		}
		ifs = append(ifs, wl.ID.Unique(string(nic.Network)))
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
	}

	if !config.Network.PublicIP.IsEmpty() {
		// some public access is required, we need to add a public
		// interface to the machine with the right config.
		inf, err := p.newPubNetworkInterface(ctx, deployment, config)
		if err != nil {
			return result, err
		}

		ipWl, _ := deployment.Get(config.Network.PublicIP)
		pubIf = ipWl.ID.Unique("pub")
		ifs = append(ifs, pubIf)
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
	}

	if config.Network.Planetary {
		inf, err := p.newYggNetworkInterface(ctx, wl)
		if err != nil {
			return result, err
		}
		ifs = append(ifs, wl.ID.Unique("ygg"))

		log.Debug().Msgf("Planetary: %+v", inf)
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
		result.PlanetaryIP = inf.IPs[0].IP.String()
	}

	if config.Network.Mycelium != nil {
		inf, err := p.newMyceliumNetworkInterface(ctx, deployment, wl, config.Network.Mycelium)
		if err != nil {
			return result, err
		}
		ifs = append(ifs, wl.ID.Unique("mycelium"))
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
		result.MyceliumIP = inf.IPs[0].IP.String()
	}
	// - mount flist RO
	mnt, err := flist.Mount(ctx, wl.ID.String(), config.FList, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
	}

	var imageInfo FListInfo
	// - detect type (container or VM)
	imageInfo, err = getFlistInfo(mnt)
	if err != nil {
		return result, err
	}

	log.Debug().Msgf("detected flist type: %+v", imageInfo)

	// mount cloud-container flist (or reuse) which has kernel, initrd and also firmware
	hash, err := flist.FlistHash(ctx, cloudContainerFlist)
	if err != nil {
		return zos.ZMachineResult{}, errors.Wrap(err, "failed to get cloud-container flist hash")
	}

	// if the name changes (because flist changed, a new mount will be created)
	name := fmt.Sprintf("%s:%s", cloudContainerName, hash)
	// now mount cloud image also
	cloudImage, err := flist.Mount(ctx, name, cloudContainerFlist, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrap(err, "failed to mount cloud container base image")
	}

	if imageInfo.IsContainer() {
		if err = p.prepContainer(ctx, cloudImage, imageInfo, &machine, &config, &deployment, wl); err != nil {
			return result, err
		}
	} else {
		if err = p.prepVirtualMachine(ctx, cloudImage, imageInfo, &machine, &config, &deployment, wl); err != nil {
			return result, err
		}

	}

	// - Attach mounts
	// - boot
	machine.Network = networkInfo
	machine.Environment = config.Env
	machine.Hostname = wl.Name.String()

	machineInfo, err := vm.Run(ctx, machine)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		log.Error().Err(err).Msg("cleaning up vm deployment duo to an error")
		_ = vm.Delete(ctx, wl.ID.String())
	}
	result.ConsoleURL = machineInfo.ConsoleURL
	return result, err
}

func (p *Manager) copyFile(srcPath string, destPath string, permissions os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrapf(err, "Coludn't find %s on the node", srcPath)
	}
	defer src.Close()
	dest, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, permissions)
	if err != nil {
		return errors.Wrapf(err, "Coludn't create %s file", destPath)
	}
	defer dest.Close()
	_, err = io.Copy(dest, src)
	if err != nil {
		return errors.Wrapf(err, "Couldn't copy to %s", destPath)
	}
	return nil
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		flist   = stubs.NewFlisterStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)
		storage = stubs.NewStorageModuleStub(p.zbus)

		cfg ZMachine
	)

	if err := json.Unmarshal(wl.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(ctx, wl.ID.String()); err == nil {
		if err := vm.Delete(ctx, wl.ID.String()); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", wl.ID)
		}
	}

	if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
		log.Error().Err(err).Msg("failed to unmount machine flist")
	}

	volName := fmt.Sprintf("rootfs:%s", wl.ID.String())
	if err := storage.VolumeDelete(ctx, volName); err != nil {
		log.Error().Err(err).Str("name", volName).Msg("failed to delete rootfs volume")
	}

	for _, inf := range cfg.Network.Interfaces {
		tapName := wl.ID.Unique(string(inf.Network))

		if err := network.RemoveTap(ctx, tapName); err != nil {
			return errors.Wrap(err, "could not clean up tap device")
		}
	}

	if cfg.Network.Planetary {
		var tapName string
		if cfg.Network.Mycelium == nil {
			// yggdrasil network
			tapName = wl.ID.Unique("ygg")
		} else {
			tapName = wl.ID.Unique("mycelium")
		}

		if err := network.RemoveTap(ctx, tapName); err != nil {
			return errors.Wrap(err, "could not clean up tap device")
		}
	}

	if len(cfg.Network.PublicIP) > 0 {
		// TODO: we need to make sure workload status reflects the actual status by the engine
		// this is not the case anymore.
		ipWl, err := provision.GetWorkload(ctx, cfg.Network.PublicIP)
		if err != nil {
			return err
		}
		ifName := ipWl.ID.Unique("pub")
		if err := network.RemovePubTap(ctx, ifName); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	return nil
}
