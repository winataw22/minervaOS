package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Kubernetes type
type Kubernetes = zos.Kubernetes

// KubernetesResult type
type KubernetesResult = zos.KubernetesResult

// const k3osFlistURL = "https://hub.grid.tf/tf-official-apps/k3os.flist"
const k3osFlistURL = "https://hub.grid.tf/lee/k3os-ch.flist"

func (p *Primitives) kubernetesProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.kubernetesProvisionImpl(ctx, wl)
}

func ensureFList(flister pkg.Flister, url string) (string, error) {
	hash, err := flister.FlistHash(url)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("k8s:%s", hash)

	return flister.NamedMount(name, url, "", pkg.ReadOnlyMountOptions)
}

func (p *Primitives) kubernetesProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result KubernetesResult, err error) {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config Kubernetes

		needsInstall = true
	)

	if vm.Exists(wl.ID.String()) {
		return result, provision.ErrDidNotChange
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	deployment := provision.GetDeployment(ctx)

	netID := zos.NetworkID(deployment.TwinID, string(config.Network))

	// check if the network tap already exists
	// if it does, it's most likely that a vm with the same network id and node id already exists
	// this will cause the reservation to fail
	exists, err := network.TapExists(netID)
	if err != nil {
		return result, errors.Wrap(err, "could not check if tap device exists")
	}

	if exists {
		return result, errors.New("kubernetes vm with same network already exists")
	}

	// check if public ipv4 is supported, should this be requested
	if len(config.PublicIP) > 0 && !network.PublicIPv4Support() {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = wl.ID.String()
	result.IP = config.IP.String()

	cpu, memory, disk, err := vmSize(&config)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(wl.ID.String()); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	imagePath, err := ensureFList(flist, k3osFlistURL)
	if err != nil {
		return result, errors.Wrap(err, "could not mount k3os flist")
	}

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", FilesystemName(wl), "vda")
	if storage.Exists(diskName) {
		needsInstall = false
		info, err := storage.Inspect(diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskPath = info.Path
	} else {
		diskPath, err = storage.Allocate(diskName, int64(disk))
		if err != nil {
			return result, errors.Wrap(err, "failed to reserve filesystem for vm")
		}
	}
	// clean up the disk anyway, even if it has already been installed.
	defer func() {
		if err != nil {
			_ = storage.Deallocate(diskName)
		}
	}()

	var iface string
	iface, err = network.SetupTap(netID)
	if err != nil {
		return result, errors.Wrap(err, "could not set up tap device")
	}

	defer func() {
		if err != nil {
			_ = network.RemoveTap(netID)
		}
	}()

	var pubIface string
	if len(config.PublicIP) > -0 {
		ipWl, err := deployment.Get(config.PublicIP)
		if err != nil {
			return zos.KubernetesResult{}, err
		}
		name := ipWl.ID.String()
		pubIface, err = network.SetupPubTap(name)
		if err != nil {
			return result, errors.Wrap(err, "could not set up tap device for public network")
		}

		defer func() {
			if err != nil {
				_ = network.RemovePubTap(name)
			}
		}()
	}

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, deployment, iface, pubIface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}

	if needsInstall {
		if err = p.kubernetesInstall(ctx, wl.ID.String(), cpu, memory, diskPath, imagePath, netInfo, config); err != nil {
			vm.Delete(wl.ID.String())
			return result, errors.Wrap(err, "failed to install k3s")
		}
	}

	err = p.kubernetesRun(ctx, wl.ID.String(), cpu, memory, diskPath, imagePath, netInfo, config)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		vm.Delete(wl.ID.String())
	}

	return result, err
}

func (p *Primitives) kubernetesInstall(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	cmdline := fmt.Sprintf("console=ttyS0 reboot=k panic=1 k3os.mode=install k3os.install.silent k3os.debug k3os.install.device=/dev/vda k3os.token=%s k3os.k3s_args=\"--flannel-iface=eth0\"", cfg.ClusterSecret)
	// if there is no server url configured, the node is set up as a master, therefore
	// this will cause nodes with an empty master list to be implicitly treated as
	// a master node
	for _, ip := range cfg.MasterIPs {
		var ipstring string
		if ip.To4() != nil {
			ipstring = ip.String()
		} else if ip.To16() != nil {
			ipstring = fmt.Sprintf("[%s]", ip.String())
		} else {
			return errors.New("invalid master IP")
		}
		cmdline = fmt.Sprintf("%s k3os.server_url=https://%s:6443", cmdline, ipstring)
	}
	for _, key := range cfg.SSHKeys {
		cmdline = fmt.Sprintf("%s ssh_authorized_keys=\"%s\"", cmdline, key)
	}
	if cfg.DatastoreEndpoint != "" {
		cmdline = fmt.Sprintf("%s k3os.k3s_args=\"--datastore-endpoint=%s\"", cmdline, cfg.DatastoreEndpoint)
	}
	if cfg.DisableDefaultIngress {
		cmdline = fmt.Sprintf("%s k3os.k3s_args=\"--disable=traefik\"", cmdline)
	}

	disks := make([]pkg.VMDisk, 2)
	// install disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}
	// install ISO
	disks[1] = pkg.VMDisk{Path: imagePath + "/k3os-amd64.iso", ReadOnly: true, Root: false}

	installVM := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  cmdline,
		Disks:       disks,
		NoKeepAlive: true, //machine will not restarted automatically when it exists
	}

	if err := vm.Run(installVM); err != nil {
		return errors.Wrap(err, "could not run vm")
	}

	deadline, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()
	for {
		if !vm.Exists(name) {
			// install is done
			break
		}
		select {
		case <-time.After(time.Second * 3):
			// retry after 3 secs
		case <-deadline.Done():
			// If install takes longer than 5 minutes, we consider it a failure.
			// In that case, we attempt a delete first. This will kill the vm process
			// if it is still going. The actual resources (disk, taps, ...) should
			// be handled by the caller.
			logs, err := vm.Logs(name)
			if err != nil {
				log.Error().Err(err).Msg("failed to get machine logs")
			} else {
				log.Warn().Str("vm", name).Str("type", "machine-logs").Msg(logs)
			}

			if err := vm.Delete(name); err != nil {
				log.Warn().Err(err).Msg("could not delete vm who's install deadline expired")
			}
			return errors.New("failed to install vm in 5 minutes")
		}
	}

	// Delete the VM, the disk will be installed now
	return vm.Delete(name)
}

func (p *Primitives) kubernetesRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	disks := make([]pkg.VMDisk, 1)
	// installed disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}

	kubevm := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  "console=ttyS0 reboot=k panic=1",
		Disks:       disks,
	}

	return vm.Run(kubevm)
}

func (p *Primitives) kubernetesDecomission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		cfg Kubernetes
	)

	if err := json.Unmarshal(wl.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(wl.ID.String()); err == nil {
		if err := vm.Delete(wl.ID.String()); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", wl.ID)
		}
	}

	deployment := provision.GetDeployment(ctx)

	netID := zos.NetworkID(deployment.TwinID, string(cfg.Network))
	if err := network.RemoveTap(netID); err != nil {
		return errors.Wrap(err, "could not clean up tap device")
	}

	if len(cfg.PublicIP) > 0 {
		if err := network.RemovePubTap(cfg.PublicIP); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	if err := storage.Deallocate(fmt.Sprintf("%s-%s", wl.ID, "vda")); err != nil {
		return errors.Wrap(err, "could not remove vDisk")
	}

	return nil
}

func (p *Primitives) buildNetworkInfo(ctx context.Context, deployment gridtypes.Deployment, iface string, pubIface string, cfg Kubernetes) (pkg.VMNetworkInfo, error) {
	network := stubs.NewNetworkerStub(p.zbus)

	netID := zos.NetworkID(deployment.TwinID, string(cfg.Network))
	subnet, err := network.GetSubnet(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	if !subnet.Contains(cfg.IP) {
		return pkg.VMNetworkInfo{}, fmt.Errorf("IP %s is not part of local nr subnet %s", cfg.IP.String(), subnet.String())
	}

	privNet, err := network.GetNet(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network range")
	}

	addrCIDR := net.IPNet{
		IP:   cfg.IP,
		Mask: subnet.Mask,
	}

	gw4, gw6, err := network.GetDefaultGwIP(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not get network resource default gateway")
	}

	privIP6, err := network.GetIPv6From4(netID, cfg.IP)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not convert private ipv4 to ipv6")
	}

	networkInfo := pkg.VMNetworkInfo{
		Ifaces: []pkg.VMIface{{
			Tap:            iface,
			MAC:            "", // rely on static IP configuration so we don't care here
			IP4AddressCIDR: addrCIDR,
			IP4GatewayIP:   net.IP(gw4),
			IP4Net:         privNet,
			IP6AddressCIDR: privIP6,
			IP6GatewayIP:   gw6,
			Public:         false,
		}},
		Nameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1"), net.ParseIP("2001:4860:4860::8888")},
	}

	if len(cfg.PublicIP) > 0 {
		// A public ip is set, load the reservation, extract the ip and make a config
		// for it
		ipWl, err := deployment.Get(cfg.PublicIP)
		if err != nil {
			return pkg.VMNetworkInfo{}, err
		}

		pubIP, pubGw, err := p.getPubIPConfig(ipWl, cfg.PublicIP)
		if err != nil {
			return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not get public ip config")
		}

		// the mac address uses the global workload id
		// this needs to be the same as how we get it in the actual IP reservation
		mac := ifaceutil.HardwareAddrFromInputBytes([]byte(ipWl.ID.String()))

		iface := pkg.VMIface{
			Tap:            pubIface,
			MAC:            mac.String(), // mac so we always get the same IPv6 from slaac
			IP4AddressCIDR: pubIP,
			IP4GatewayIP:   pubGw,
			// for now we get ipv6 from slaac, so leave ipv6 stuffs this empty
			//
			Public: true,
		}

		networkInfo.Ifaces = append(networkInfo.Ifaces, iface)
	}

	return networkInfo, nil
}

// Get the public ip, and the gateway from the reservation ID
func (p *Primitives) getPubIPConfig(wl *gridtypes.WorkloadWithID, name string) (ip net.IPNet, gw net.IP, err error) {

	//CRITICAL: TODO
	// in this function we need to return the IP from the IP workload
	// but we also need to get the Gateway IP from the farmer some how
	// we used to get this from the explorer, but now we need another
	// way to do this. for now the only option is to get it from the
	// reservation itself. hence we added the gatway fields to ip data
	if wl.Type != zos.PublicIPType {
		return ip, gw, fmt.Errorf("workload for public IP is of wrong type")
	}

	if wl.Result.State != gridtypes.StateOk {
		return ip, gw, fmt.Errorf("public ip workload is not okay")
	}
	ipData, err := wl.WorkloadData()
	if err != nil {
		return
	}
	data, ok := ipData.(*zos.PublicIP)
	if !ok {
		return ip, gw, fmt.Errorf("invalid ip data in deployment got '%T'", ipData)
	}

	return data.IP.IPNet, data.Gateway, nil
}

// returns the vCpu's, memory, disksize for a vm size
// memory and disk size is expressed in MiB
func vmSize(vm *Kubernetes) (cpu uint8, memory uint64, storage uint64, err error) {
	switch vm.Size {
	case 0:
		return 0, 0, 0, fmt.Errorf("not supported size")
	case 1:
		// 1 vCpu, 2 GiB RAM, 50 GB disk
		return 1, 2 * 1024, 50 * 1024, nil
	case 2:
		// 2 vCpu, 4 GiB RAM, 100 GB disk
		return 2, 4 * 1024, 100 * 1024, nil
	case 3:
		// 2 vCpu, 8 GiB RAM, 25 GB disk
		return 2, 8 * 1024, 25 * 1024, nil
	case 4:
		// 2 vCpu, 8 GiB RAM, 50 GB disk
		return 2, 8 * 1024, 50 * 1024, nil
	case 5:
		// 2 vCpu, 8 GiB RAM, 200 GB disk
		return 2, 8 * 1024, 200 * 1024, nil
	case 6:
		// 4 vCpu, 16 GiB RAM, 50 GB disk
		return 4, 16 * 1024, 50 * 1024, nil
	case 7:
		// 4 vCpu, 16 GiB RAM, 100 GB disk
		return 4, 16 * 1024, 100 * 1024, nil
	case 8:
		// 4 vCpu, 16 GiB RAM, 400 GB disk
		return 4, 16 * 1024, 400 * 1024, nil
	case 9:
		// 8 vCpu, 32 GiB RAM, 100 GB disk
		return 8, 32 * 1024, 100 * 1024, nil
	case 10:
		// 8 vCpu, 32 GiB RAM, 200 GB disk
		return 8, 32 * 1024, 200 * 1024, nil
	case 11:
		// 8 vCpu, 32 GiB RAM, 800 GB disk
		return 8, 32 * 1024, 800 * 1024, nil
	case 12:
		// 1 vCpu, 64 GiB RAM, 200 GB disk
		return 1, 64 * 1024, 200 * 1024, nil
	case 13:
		// 1 mvCpu, 64 GiB RAM, 400 GB disk
		return 1, 64 * 1024, 400 * 1024, nil
	case 14:
		//1 vCpu, 64 GiB RAM, 800 GB disk
		return 1, 64 * 1024, 800 * 1024, nil
	case 15:
		//1 vCpu, 2 GiB RAM, 25 GB disk
		return 1, 2 * 1024, 25 * 1024, nil
	case 16:
		//2 vCpu, 4 GiB RAM, 50 GB disk
		return 2, 4 * 1024, 50 * 1024, nil
	case 17:
		//4 vCpu, 8 GiB RAM, 50 GB disk
		return 4, 8 * 1024, 50 * 1024, nil
	case 18:
		//1 vCpu, 1 GiB RAM, 25 GB disk
		return 1, 1 * 1024, 25 * 1024, nil
	}

	return 0, 0, 0, fmt.Errorf("unsupported vm size %d, only size 1 to 18 are supported", vm.Size)
}
