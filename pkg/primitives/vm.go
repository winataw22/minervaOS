package primitives

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"

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

func (p *Primitives) mountsToDisks(ctx context.Context, deployment gridtypes.Deployment, disks []zos.MachineMount, format bool) ([]pkg.VMDisk, error) {
	storage := stubs.NewStorageModuleStub(p.zbus)

	var results []pkg.VMDisk
	for _, disk := range disks {
		wl, err := deployment.Get(disk.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get disk '%s' workload", disk.Name)
		}
		if wl.Type != zos.ZMountType {
			return nil, fmt.Errorf("expecting a reservation of type '%s' for disk '%s'", zos.ZMountType, disk.Name)
		}
		if wl.Result.State != gridtypes.StateOk {
			return nil, fmt.Errorf("invalid disk '%s' state", disk.Name)
		}

		info, err := storage.DiskLookup(ctx, wl.ID.String())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to inspect disk '%s'", disk.Name)
		}

		if format {
			if err := storage.DiskFormat(ctx, wl.ID.String()); err != nil {
				return nil, errors.Wrap(err, "failed to prepare mount")
			}
		}

		results = append(results, pkg.VMDisk{Path: info.Path, Target: disk.Mountpoint})
	}

	return results, nil
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
	var disks []pkg.VMDisk

	// "root=/dev/vda rw console=ttyS0 reboot=k panic=1"
	cmd := pkg.KernelArgs{
		"rw":      "",
		"console": "ttyS0",
		"reboot":  "k",
		"panic":   "1",
		"root":    "/dev/vda",
	}

	if imageInfo.Container {
		// - if Container, remount RW
		// prepare for container
		if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
			return result, errors.Wrapf(err, "failed to unmount flist: %s", wl.ID.String())
		}
		// remounting in RW mode
		mnt, err = flist.Mount(ctx, wl.ID.String(), config.FList, pkg.DefaultMountOptions)
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
		// change the root boot to use the right virtiofs tag
		cmd["init"] = config.Entrypoint

		disks, err = p.mountsToDisks(ctx, deployment, config.Mounts, true)
		if err != nil {
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
		// we don't format disks attached to VMs, it's up to the vm to decide that
		disks, err = p.mountsToDisks(ctx, deployment, config.Mounts[1:], false)
		if err != nil {
			return result, err
		}
	}

	// - Attach mounts
	// - boot

	err = p.vmRun(ctx, wl.ID.String(), &config, boot, disks, imageInfo, cmd, networkInfo)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
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

func (p *Primitives) vmRun(
	ctx context.Context,
	name string,
	config *ZMachine,
	boot pkg.Boot,
	disks []pkg.VMDisk,
	imageInfo FListInfo,
	cmdline pkg.KernelArgs,
	networkInfo pkg.VMNetworkInfo) error {

	vm := stubs.NewVMModuleStub(p.zbus)

	cap := config.ComputeCapacity
	// installed disk
	kubevm := pkg.VM{
		Name:        name,
		CPU:         cap.CPU,
		Memory:      cap.Memory,
		Network:     networkInfo,
		KernelImage: imageInfo.Kernel,
		InitrdImage: imageInfo.Initrd,
		KernelArgs:  cmdline,
		Boot:        boot,
		Environment: config.Env,
		Disks:       disks,
	}

	return vm.Run(ctx, kubevm)
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
