package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// KubernetesResult result returned by k3s reservation
type KubernetesResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// Kubernetes reservation data
type Kubernetes struct {
	// Size of the vm, this defines the amount of vCpu, memory, and the disk size
	// Docs: docs/kubernetes/sizes.md
	Size uint8 `json:"size"`

	// NetworkID of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	NetworkID pkg.NetID `json:"network_id"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`

	// ClusterSecret is the hex encoded encrypted cluster secret.
	ClusterSecret string `json:"cluster_secret"`
	// MasterIPs define the URL's for the kubernetes master nodes. If this
	// list is empty, this node is considered to be a master node.
	MasterIPs []net.IP `json:"master_ips"`
	// SSHKeys is a list of ssh keys to add to the VM. Keys can be either
	// a full ssh key, or in the form of `github:${username}`. In case of
	// the later, the VM will retrieve the github keys for this username
	// when it boots.
	SSHKeys []string `json:"ssh_keys"`
	// PublicIP points to a reservation for a public ip
	PublicIP schema.ID `json:"public_ip"`

	PlainClusterSecret string `json:"-"`
}

// const k3osFlistURL = "https://hub.grid.tf/tf-official-apps/k3os.flist"
const k3osFlistURL = "https://hub.grid.tf/lee/k3os.flist"

func (p *Provisioner) kubernetesProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.kubernetesProvisionImpl(ctx, reservation)
}

func (p *Provisioner) kubernetesProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result KubernetesResult, err error) {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config Kubernetes

		needsInstall = true
	)

	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	netID := provision.NetworkID(reservation.User, string(config.NetworkID))

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
	if config.PublicIP != 0 && !network.PublicIPv4Support() {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = reservation.ID
	result.IP = config.IP.String()

	config.PlainClusterSecret, err = decryptSecret(config.ClusterSecret, reservation.User, reservation.Version, p.zbus)
	if err != nil {
		return result, errors.Wrap(err, "failed to decrypt namespace password")
	}

	cpu, memory, disk, err := vmSize(config.Size)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	var imagePath string
	imagePath, err = flist.NamedMount(provision.FilesystemName(*reservation), k3osFlistURL, "", pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrap(err, "could not mount k3os flist")
	}
	// In case of future errors in the provisioning make sure we clean up
	defer func() {
		if err != nil {
			_ = flist.Umount(imagePath)
		}
	}()

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", provision.FilesystemName(*reservation), "vda")
	if storage.Exists(diskName) {
		needsInstall = false
		info, err := storage.Inspect(diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskName = info.Path
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
	if config.PublicIP != 0 {
		pubIface, err = network.SetupPubTap(netID)
		if err != nil {
			return result, errors.Wrap(err, "could not set up tap device for public network")
		}

		defer func() {
			if err != nil {
				_ = network.RemovePubTap(netID)
			}
		}()
	}

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, reservation.Version, reservation.User, iface, pubIface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}

	if needsInstall {
		if err = p.kubernetesInstall(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config); err != nil {
			return result, errors.Wrap(err, "failed to install k3s")
		}
	}

	err = p.kubernetesRun(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		vm.Delete(reservation.ID)
	}

	return result, err
}

func (p *Provisioner) kubernetesInstall(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	cmdline := fmt.Sprintf("console=ttyS0 reboot=k panic=1 k3os.mode=install k3os.install.silent k3os.install.device=/dev/vda k3os.token=%s", cfg.PlainClusterSecret)
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

func (p *Provisioner) kubernetesRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	disks := make([]pkg.VMDisk, 1)
	// installed disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}
	cmdline := fmt.Sprintf("console=ttyS0 reboot=k panic=1 k3os.token=%s", cfg.PlainClusterSecret)

	kubevm := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  cmdline,
		Disks:       disks,
	}

	return vm.Run(kubevm)
}

func (p *Provisioner) kubernetesDecomission(ctx context.Context, reservation *provision.Reservation) error {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		cfg Kubernetes
	)

	if err := json.Unmarshal(reservation.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(reservation.ID); err == nil {
		if err := vm.Delete(reservation.ID); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", reservation.ID)
		}
	}

	netID := provision.NetworkID(reservation.User, string(cfg.NetworkID))
	if err := network.RemoveTap(netID); err != nil {
		return errors.Wrap(err, "could not clean up tap device")
	}

	if cfg.PublicIP != 0 {
		if err := network.RemovePubTap(netID); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	if err := storage.Deallocate(fmt.Sprintf("%s-%s", reservation.ID, "vda")); err != nil {
		return errors.Wrap(err, "could not remove vDisk")
	}

	// Unmount flist, skip error if any.
	if err := flist.NamedUmount(reservation.ID); err != nil {
		log.Err(err).Msg("could not unmount flist")
	}

	return nil
}

func (p *Provisioner) buildNetworkInfo(ctx context.Context, rversion int, userID string, iface string, pubIface string, cfg Kubernetes) (pkg.VMNetworkInfo, error) {
	network := stubs.NewNetworkerStub(p.zbus)

	netID := provision.NetworkID(userID, string(cfg.NetworkID))
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

	// from this reservation version on we deploy new VM's with the custom boot script for IP
	if rversion >= 2 {
		networkInfo.NewStyle = true
	}

	if cfg.PublicIP != 0 {
		// A public ip is set, load the reservation, extract the ip and make a config
		// for it
		// TODO: proper firewalling on the host

		pubIP, pubGw, err := p.getPubIPConfig(cfg.PublicIP)
		if err != nil {
			return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not get public ip config")
		}

		iface := pkg.VMIface{
			Tap:            pubIface,
			MAC:            "", // static ip is set so not important
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
func (p *Provisioner) getPubIPConfig(rid schema.ID) (net.IPNet, net.IP, error) {
	// TODO: check if there is a better way to do this
	explorerClient, err := app.ExplorerClient()
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not create explorer client")
	}

	// explorerClient.Workloads.Get(...) is currently broken
	workloadDefinition, err := explorerClient.Workloads.NodeWorkloadGet(fmt.Sprintf("%d-1", rid))
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not load public ip reservation")
	}
	// load IP
	ip, ok := workloadDefinition.(*workloads.PublicIP)
	if !ok {
		return net.IPNet{}, nil, errors.Wrap(err, "could not decode ip reservation")
	}
	identity := stubs.NewIdentityManagerStub(p.zbus)
	self := identity.NodeID().Identity()
	selfDescription, err := explorerClient.Directory.NodeGet(self, false)
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not get our own node description")
	}
	farm, err := explorerClient.Directory.FarmGet(schema.ID(selfDescription.FarmId))
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not get our own farm")
	}

	var pubGw schema.IP
	for _, ips := range farm.IPAddresses {
		if ips.ReservationID == rid {
			pubGw = ips.Gateway
			break
		}
	}
	if pubGw.IP == nil {
		return net.IPNet{}, nil, errors.New("unable to identify public ip gateway")
	}

	return ip.IPaddress.IPNet, pubGw.IP, nil
}

// returns the vCpu's, memory, disksize for a vm size
// memory and disk size is expressed in MiB
func vmSize(size uint8) (cpu uint8, memory uint64, storage uint64, err error) {

	switch size {
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
	}

	return 0, 0, 0, fmt.Errorf("unsupported vm size %d, only size 1 and 2 are supported", size)
}
