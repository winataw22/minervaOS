package vm

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/primitives/pubip"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var networkResourceNet = net.IPNet{
	IP:   net.ParseIP("100.64.0.0"),
	Mask: net.IPv4Mask(0xff, 0xff, 0, 0),
}

// fill up the VM (machine) object with write boot config for a full virtual machine (with a disk image)
func (p *Manager) prepVirtualMachine(
	ctx context.Context,
	cloudImage string,
	imageInfo FListInfo,
	machine *pkg.VM,
	config *zos.ZMachine,
	deployment *gridtypes.Deployment,
	wl *gridtypes.WorkloadWithID,
) error {
	storage := stubs.NewStorageModuleStub(p.zbus)
	// if a VM the vm has to have at least one mount
	if len(config.Mounts) == 0 {
		return fmt.Errorf("at least one mount has to be attached for Vm mode")
	}

	machine.KernelImage = filepath.Join(cloudImage, "hypervisor-fw")
	disk, err := deployment.Get(config.Mounts[0].Name)
	if err != nil {
		return err
	}

	if disk.Type != zos.ZMountType {
		return fmt.Errorf("mount is not a valid disk workload")
	}

	if disk.Result.State != gridtypes.StateOk {
		return fmt.Errorf("boot disk was not deployed correctly")
	}

	info, err := storage.DiskLookup(ctx, disk.ID.String())
	if err != nil {
		return errors.Wrap(err, "disk does not exist")
	}

	// TODO: DiskWrite will not override the disk if it already has a partition table
	// or a filesystem. this means that if later the disk is assigned to a new VM with
	// a different flist it will have the same old operating system copied from previous
	// setup.
	if err = storage.DiskWrite(ctx, disk.ID.String(), imageInfo.ImagePath); err != nil {
		return errors.Wrap(err, "failed to write image to disk")
	}

	machine.Boot = pkg.Boot{
		Type: pkg.BootDisk,
		Path: info.Path,
	}

	return p.vmMounts(ctx, deployment, config.Mounts[1:], false, machine)
}

// prepare the machine and fill it up with proper boot flags for a container VM
func (p *Manager) prepContainer(
	ctx context.Context,
	cloudImage string,
	imageInfo FListInfo,
	machine *pkg.VM,
	config *zos.ZMachine,
	deployment *gridtypes.Deployment,
	wl *gridtypes.WorkloadWithID,
) error {
	// - if Container, remount RW
	// prepare for container
	var (
		storage = stubs.NewStorageModuleStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
	)

	if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
		return errors.Wrapf(err, "failed to unmount flist: %s", wl.ID.String())
	}
	rootfsSize := config.RootSize()
	// create a persisted volume for the vm. we don't do it automatically
	// via the flist, so we have control over when to decomission this volume.
	// remounting in RW mode
	volName := fmt.Sprintf("rootfs:%s", wl.ID.String())

	volumeExists, err := storage.VolumeExists(ctx, volName)
	if err != nil {
		return errors.Wrap(err, "failed to check if vm rootfs exists")
	}

	volume, err := storage.VolumeCreate(ctx, volName, rootfsSize)
	if err != nil {
		return errors.Wrap(err, "failed to create vm rootfs")
	}

	defer func() {
		if err != nil {
			// vm creation failed,
			if err := storage.VolumeDelete(ctx, volName); err != nil {
				log.Error().Err(err).Str("volume", volName).Msg("failed to delete persisted volume")
			}
		}
	}()

	mnt, err := flist.Mount(ctx, wl.ID.String(), config.FList, pkg.MountOptions{
		ReadOnly:        false,
		PersistedVolume: volume.Path,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
	}

	// clean up host keys
	if !volumeExists {
		files, err := filepath.Glob(filepath.Join(mnt, "etc", "ssh", "ssh_host_*"))
		if err != nil {
			log.Debug().Err(err).Msg("failed to list ssh host keys for a vm image")
		}

		for _, file := range files {
			if err := os.Remove(file); err != nil {
				log.Debug().Err(err).Str("file", file).Msg("failed to delete host key file")
			}
		}
	}

	// inject container kernel and init
	machine.KernelImage = filepath.Join(cloudImage, "kernel")
	machine.InitrdImage = filepath.Join(cloudImage, "initramfs-linux.img")

	// can be overridden from the flist itself if exists
	if len(imageInfo.KernelPath) != 0 {
		// try to decompress kernel
		if err := tryDecompressKernel(imageInfo.KernelPath); err != nil {
			return errors.Wrapf(err, "failed to decompress kernel: %s", imageInfo.KernelPath)
		}

		machine.KernelImage = imageInfo.KernelPath
		machine.InitrdImage = imageInfo.InitrdPath
		// we are using kernel from flist, we need to respect
		// user init
		if len(config.Entrypoint) != 0 {
			machine.KernelArgs["init"] = config.Entrypoint
		}
	}

	machine.Boot = pkg.Boot{
		Type: pkg.BootVirtioFS,
		Path: mnt,
	}

	if err := fListStartup(config, filepath.Join(mnt, ".startup.toml")); err != nil {
		return errors.Wrap(err, "failed to apply startup config from flist")
	}

	if err := p.vmMounts(ctx, deployment, config.Mounts, true, machine); err != nil {
		return err
	}
	if config.Corex {
		if err := p.copyFile("/usr/bin/corex", filepath.Join(mnt, "corex"), 0755); err != nil {
			return errors.Wrap(err, "failed to inject corex binary")
		}
		machine.Entrypoint = "/corex --ipv6 -d 7 --interface eth0"
	}

	return nil
}

func (p *Manager) newMyceliumNetworkInterface(ctx context.Context, dl gridtypes.Deployment, wl *gridtypes.WorkloadWithID, config *zos.MyceliumIP) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)
	netID := zos.NetworkID(dl.TwinID, config.Network)

	tapName := wl.ID.Unique("mycelium")
	iface, err := network.SetupMyceliumTap(ctx, tapName, netID, *config)

	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not set up tap device")
	}

	out := pkg.VMIface{
		Tap: iface.Name,
		MAC: iface.HW.String(),
		IPs: []net.IPNet{
			iface.IP,
		},
		Routes: []pkg.Route{
			{
				Net: net.IPNet{
					IP:   net.ParseIP("400::"),
					Mask: net.CIDRMask(7, 128),
				},
				Gateway: iface.Gateway.IP,
			},
		},
		PublicIPv4: false,
		PublicIPv6: false,
	}

	return out, nil
}

func (p *Manager) newYggNetworkInterface(ctx context.Context, wl *gridtypes.WorkloadWithID) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)

	// TODO: if we use `ygg` as a network name. this will conflict
	// if the user has a network that is called `ygg`.
	tapName := wl.ID.Unique("ygg")
	iface, err := network.SetupYggTap(ctx, tapName)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not set up tap device")
	}

	out := pkg.VMIface{
		Tap: iface.Name,
		MAC: iface.HW.String(),
		IPs: []net.IPNet{
			iface.IP,
		},
		Routes: []pkg.Route{
			{
				Net: net.IPNet{
					IP:   net.ParseIP("200::"),
					Mask: net.CIDRMask(7, 128),
				},
				Gateway: iface.Gateway.IP,
			},
		},
		PublicIPv4: false,
		PublicIPv6: false,
	}

	return out, nil
}

func (p *Manager) newPrivNetworkInterface(ctx context.Context, dl gridtypes.Deployment, wl *gridtypes.WorkloadWithID, inf zos.MachineInterface) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)
	netID := zos.NetworkID(dl.TwinID, inf.Network)

	subnet, err := network.GetSubnet(ctx, netID)
	if err != nil {
		return pkg.VMIface{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	inf.IP = inf.IP.To4()
	if inf.IP == nil {
		return pkg.VMIface{}, fmt.Errorf("invalid IPv4 supplied to wg interface")
	}

	if !subnet.Contains(inf.IP) {
		return pkg.VMIface{}, fmt.Errorf("IP %s is not part of local nr subnet %s", inf.IP.String(), subnet.String())
	}

	// always the .1/24 ip is reserved
	if inf.IP[3] == 1 {
		return pkg.VMIface{}, fmt.Errorf("ip %s is reserved", inf.IP.String())
	}

	privNet, err := network.GetNet(ctx, netID)
	if err != nil {
		return pkg.VMIface{}, errors.Wrapf(err, "could not get network range")
	}

	addrCIDR := net.IPNet{
		IP:   inf.IP,
		Mask: subnet.Mask,
	}

	gw4, gw6, err := network.GetDefaultGwIP(ctx, netID)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not get network resource default gateway")
	}

	privIP6, err := network.GetIPv6From4(ctx, netID, inf.IP)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not convert private ipv4 to ipv6")
	}

	tapName := wl.ID.Unique(string(inf.Network))
	iface, err := network.SetupPrivTap(ctx, netID, tapName)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not set up tap device")
	}

	// the mac address uses the global workload id
	// this needs to be the same as how we get it in the actual IP reservation
	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(tapName))

	out := pkg.VMIface{
		Tap: iface,
		MAC: mac.String(),
		IPs: []net.IPNet{
			addrCIDR, privIP6,
		},
		Routes: []pkg.Route{
			{Net: privNet, Gateway: gw4},
			{Net: networkResourceNet, Gateway: gw4},
		},
		IP4DefaultGateway: net.IP(gw4),
		IP6DefaultGateway: gw6,
		PublicIPv4:        false,
		PublicIPv6:        false,
		NetID:             netID,
	}

	return out, nil
}

func (p *Manager) newPubNetworkInterface(ctx context.Context, deployment gridtypes.Deployment, cfg ZMachine) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)
	ipWl, err := deployment.Get(cfg.Network.PublicIP)
	if err != nil {
		return pkg.VMIface{}, err
	}

	tapName := ipWl.ID.Unique("pub")

	config, err := pubip.GetPubIPConfig(ipWl)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not get public ip config")
	}

	pubIface, err := network.SetupPubTap(ctx, tapName)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not set up tap device for public network")
	}

	// the mac address uses the global workload id
	// this needs to be the same as how we get it in the actual IP reservation
	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(tapName))

	// pubic ip config can has
	// - reserved public ipv4
	// - public ipv6
	// - both
	// in all cases we have ipv6 it's handed out via slaac, so we don't need
	// to set the IP on the interface. We need to configure it ONLY for ipv4
	// hence:
	var ips []net.IPNet
	var gw4 net.IP
	var gw6 net.IP

	if !config.IP.Nil() {
		ips = append(ips, config.IP.IPNet)
		gw4 = config.Gateway
	}

	if !config.IPv6.Nil() {
		ips = append(ips, config.IPv6.IPNet)
		gw6, err = network.GetPublicIPV6Gateway(ctx)
		log.Debug().IPAddr("gw", gw6).Msg("found gateway for ipv6")
		if err != nil {
			return pkg.VMIface{}, errors.Wrap(err, "failed to get the default gateway for ipv6")
		}
	}

	return pkg.VMIface{
		Tap:               pubIface,
		MAC:               mac.String(), // mac so we always get the same IPv6 from slaac
		IPs:               ips,
		IP4DefaultGateway: gw4,
		IP6DefaultGateway: gw6,
		PublicIPv4:        config.HasIPv4(),
		PublicIPv6:        config.HasIPv6(),
	}, nil
}

// FListInfo virtual machine details
type FListInfo struct {
	ImagePath  string
	KernelPath string
	InitrdPath string
}

func (t *FListInfo) IsContainer() bool {
	return len(t.ImagePath) == 0
}

func getFlistInfo(flistPath string) (flist FListInfo, err error) {
	files := map[string]*string{
		"/image.raw":       &flist.ImagePath,
		"/boot/vmlinuz":    &flist.KernelPath,
		"/boot/initrd.img": &flist.InitrdPath,
	}

	for rel, ptr := range files {
		path := filepath.Join(flistPath, rel)
		// this path can be a symlink so we need to make sure
		// the symlink is pointing to only files inside the
		// flist.

		// but we need to validate
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return flist, errors.Wrapf(err, "couldn't stat %s", rel)
		}

		if stat.IsDir() {
			return flist, fmt.Errorf("path '%s' cannot be a directory", rel)
		}
		mod := stat.Mode()
		switch mod.Type() {
		case 0:
			// regular file, do nothing
		case os.ModeSymlink:
			// this is a symlink. we
			// need to make sure it's a safe link
			// to a location inside the flist
			link, err := os.Readlink(path)
			if err != nil {
				return flist, errors.Wrapf(err, "failed to read link at '%s", rel)
			}
			// now the link if joined with path, (and cleaned) need to also point
			// to somewhere under flistPath
			abs := filepath.Clean(filepath.Join(flistPath, link))
			if !strings.HasPrefix(abs, flistPath) {
				return flist, fmt.Errorf("path '%s' points to invalid location", rel)
			}
		default:
			return flist, fmt.Errorf("path '%s' is of invalid type: %s", rel, mod.Type().String())
		}

		// set the value
		*ptr = path
	}

	return flist, nil
}

type startup struct {
	Entries map[string]entry `toml:"startup"`
}

type entry struct {
	Name string
	Args args
}

type args struct {
	Name string
	Dir  string
	Args []string
	Env  map[string]string
}

func (e entry) Entrypoint() string {
	if e.Name == "core.system" ||
		e.Name == "core.base" && e.Args.Name != "" {
		var buf strings.Builder

		buf.WriteString(e.Args.Name)
		for _, arg := range e.Args.Args {
			buf.WriteRune(' ')
			arg = strings.Replace(arg, "\"", "\\\"", -1)
			buf.WriteRune('"')
			buf.WriteString(arg)
			buf.WriteRune('"')
		}

		return buf.String()
	}

	return ""
}

func (e entry) WorkingDir() string {
	return e.Args.Dir
}

func (e entry) Envs() map[string]string {
	return e.Args.Env
}

// This code is backward compatible with flist .startup.toml file
// where the flist can define an Entrypoint and some initial environment
// variables. this is used *with* the container configuration like this
// - if no zmachine entry point is defined, use the one from .startup.toml
// - if envs are defined in flist, merge with the env variables from the
func fListStartup(data *zos.ZMachine, path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to load startup file '%s'", path)
	}

	defer f.Close()

	log.Info().Msg("startup file found")
	startup := startup{}
	if _, err := toml.NewDecoder(f).Decode(&startup); err != nil {
		return err
	}

	entry, ok := startup.Entries["entry"]
	if !ok {
		return nil
	}

	data.Env = mergeEnvs(entry.Envs(), data.Env)

	if data.Entrypoint == "" && entry.Entrypoint() != "" {
		data.Entrypoint = entry.Entrypoint()
	}
	return nil
}

// mergeEnvs new into base
func mergeEnvs(base, new map[string]string) map[string]string {
	if len(base) == 0 {
		return new
	}

	for k, v := range new {
		base[k] = v
	}

	return base
}
