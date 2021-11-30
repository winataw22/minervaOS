package primitives

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	// this probably need a way to update. for now just hard code it
	cloudContainerFlist = "https://hub.grid.tf/azmy.3bot/cloud-container.flist"
	cloudContainerName  = "cloud-container"
)

// ZMachine type
type ZMachine = zos.ZMachine

// ZMachineResult type
type ZMachineResult = zos.ZMachineResult

// FListInfo virtual machine details
type FListInfo struct {
	Container bool
	Initrd    string
	Kernel    string
	ImagePath string
}

func (p *Primitives) virtualMachineProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, wl)
}

func (p *Primitives) vmMounts(ctx context.Context, deployment gridtypes.Deployment, mounts []zos.MachineMount, format bool, vm *pkg.VM) error {
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

func (p *Primitives) mountDisk(ctx context.Context, wl *gridtypes.WorkloadWithID, mount zos.MachineMount, format bool, vm *pkg.VM) error {
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

func (p *Primitives) mountQsfs(wl *gridtypes.WorkloadWithID, mount zos.MachineMount, vm *pkg.VM) error {

	var info zos.QuatumSafeFSResult
	if err := wl.Result.Unmarshal(&info); err != nil {
		return fmt.Errorf("invalid qsfs result '%s': %w", mount.Name, err)
	}
	wlID := wl.ID.String()
	vm.Shared = append(vm.Shared, pkg.SharedDir{ID: strings.ReplaceAll(wlID, "-", ""), Path: info.Path, Target: mount.Mountpoint})
	return nil
}

func (p *Primitives) virtualMachineProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.ZMachineResult, err error) {
	var (
		storage = stubs.NewStorageModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config ZMachine
	)
	if vm.Exists(ctx, wl.ID.String()) {
		return result, provision.ErrDidNotChange
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}
	machine := pkg.VM{
		Name:        wl.ID.String(),
		CPU:         config.ComputeCapacity.CPU,
		Memory:      config.ComputeCapacity.Memory,
		Environment: config.Env,
	}
	// Should config.Vaid() be called here?

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

	deployment := provision.GetDeployment(ctx)

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
		ifs = append(ifs, tapNameFromName(wl.ID, string(nic.Network)))
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
		pubIf = tapNameFromName(ipWl.ID, "pub")
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
	}

	if config.Network.Planetary {
		inf, err := p.newYggNetworkInterface(ctx, wl)
		if err != nil {
			return result, err
		}

		log.Debug().Msgf("Planetary: %+v", inf)
		ifs = append(ifs, tapNameFromName(wl.ID, "ygg"))
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
		result.YggIP = inf.IPs[0].IP.String()
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

	var boot pkg.Boot

	// "root=/dev/vda rw console=ttyS0 reboot=k panic=1"
	cmd := pkg.KernelArgs{
		"rw":      "",
		"console": "ttyS0",
		"reboot":  "k",
		"panic":   "1",
		"root":    "/dev/vda",
	}
	var entrypoint string
	if imageInfo.Container {
		// - if Container, remount RW
		// prepare for container
		if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
			return result, errors.Wrapf(err, "failed to unmount flist: %s", wl.ID.String())
		}
		rootfsSize := config.Size
		if rootfsSize < 250*gridtypes.Megabyte {
			rootfsSize = 250 * gridtypes.Megabyte
		}
		// remounting in RW mode
		mnt, err = flist.Mount(ctx, wl.ID.String(), config.FList, pkg.MountOptions{
			ReadOnly: false,
			Limit:    rootfsSize,
		})
		if err != nil {
			return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
		}

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
		// inject container kernel and init
		imageInfo.Kernel = filepath.Join(cloudImage, "kernel")
		imageInfo.Initrd = filepath.Join(cloudImage, "initramfs-linux.img")

		boot = pkg.Boot{
			Type: pkg.BootVirtioFS,
			Path: mnt,
		}

		if err := fListStartup(&config, filepath.Join(mnt, ".startup.toml")); err != nil {
			return result, errors.Wrap(err, "failed to apply startup config from flist")
		}

		cmd["host"] = string(wl.Name)
		entrypoint = config.Entrypoint
		if err := p.vmMounts(ctx, deployment, config.Mounts, true, &machine); err != nil {
			return result, err
		}
	} else {
		// if a VM the vm has to have at least one mount
		if len(config.Mounts) == 0 {
			err = fmt.Errorf("at least one mount has to be attached for Vm mode")
			return result, err
		}

		var disk *gridtypes.WorkloadWithID
		disk, err = deployment.Get(config.Mounts[0].Name)
		if err != nil {
			return result, err
		}

		if disk.Type != zos.ZMountType {
			return result, fmt.Errorf("mount is not not a valid disk workload")
		}

		if disk.Result.State != gridtypes.StateOk {
			return result, fmt.Errorf("boot disk was not deployed correctly")
		}
		var info pkg.VDisk
		info, err = storage.DiskLookup(ctx, disk.ID.String())
		if err != nil {
			return result, errors.Wrap(err, "disk does not exist")
		}

		//TODO: this should not happen if disk image was written before !!
		// fs detection must be done here
		if err = storage.DiskWrite(ctx, disk.ID.String(), imageInfo.ImagePath); err != nil {
			return result, errors.Wrap(err, "failed to write image to disk")
		}

		boot = pkg.Boot{
			Type: pkg.BootDisk,
			Path: info.Path,
		}
		if err := p.vmMounts(ctx, deployment, config.Mounts[1:], false, &machine); err != nil {
			return result, err
		}
	}

	// - Attach mounts
	// - boot
	machine.Network = networkInfo
	machine.KernelImage = imageInfo.Kernel
	machine.InitrdImage = imageInfo.Initrd
	machine.KernelArgs = cmd
	machine.Boot = boot
	machine.Entrypoint = entrypoint

	if err = vm.Run(ctx, machine); err != nil {
		// attempt to delete the vm, should the process still be lingering
		log.Error().Err(err).Msg("cleaning up vm deployment duo to an error")
		_ = vm.Delete(ctx, wl.ID.String())
	}

	return result, err
}

func (p *Primitives) vmDecomission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		flist   = stubs.NewFlisterStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

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

	for _, inf := range cfg.Network.Interfaces {
		tapName := tapNameFromName(wl.ID, string(inf.Network))

		if err := network.RemoveTap(ctx, tapName); err != nil {
			return errors.Wrap(err, "could not clean up tap device")
		}
	}

	if cfg.Network.Planetary {
		tapName := tapNameFromName(wl.ID, "ygg")
		if err := network.RemoveTap(ctx, tapName); err != nil {
			return errors.Wrap(err, "could not clean up tap device")
		}
	}

	if len(cfg.Network.PublicIP) > 0 {
		deployment := provision.GetDeployment(ctx)
		ipWl, err := deployment.Get(cfg.Network.PublicIP)
		if err != nil {
			return err
		}
		ifName := tapNameFromName(ipWl.ID, "pub")
		if err := network.RemovePubTap(ctx, ifName); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	return nil
}

func tapNameFromName(id gridtypes.WorkloadID, network string) string {
	m := md5.New()

	fmt.Fprintf(m, "%s:%s", id.String(), network)

	h := m.Sum(nil)
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return string(b)
}
