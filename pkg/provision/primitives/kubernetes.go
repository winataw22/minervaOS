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

	PlainClusterSecret string `json:"-"`
}

const k3osFlistURL = "https://hub.grid.tf/tf-official-apps/k3os.flist"

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
			_ = vm.Delete(reservation.ID)
			_ = network.RemoveTap(netID)
		}
	}()

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, reservation.User, iface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}

	if needsInstall {
		if err = p.kubernetesInstall(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config); err != nil {
			return result, errors.Wrap(err, "failed to install k3s")
		}
	}

	err = p.kubernetesRun(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config)
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

	if err := storage.Deallocate(fmt.Sprintf("%s-%s", reservation.ID, "vda")); err != nil {
		return errors.Wrap(err, "could not remove vDisk")
	}

	// Unmount flist, skip error if any.
	if err := flist.NamedUmount(reservation.ID); err != nil {
		log.Err(err).Msg("could not unmount flist")
	}

	return nil
}

func (p *Provisioner) buildNetworkInfo(ctx context.Context, userID string, iface string, cfg Kubernetes) (pkg.VMNetworkInfo, error) {
	network := stubs.NewNetworkerStub(p.zbus)

	netID := provision.NetworkID(userID, string(cfg.NetworkID))
	subnet, err := network.GetSubnet(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	if !subnet.Contains(cfg.IP) {
		return pkg.VMNetworkInfo{}, fmt.Errorf("IP %s is not part of local nr subnet %s", cfg.IP.String(), subnet.String())
	}

	addrCIDR := net.IPNet{
		IP:   cfg.IP,
		Mask: subnet.Mask,
	}

	gw, err := network.GetDefaultGwIP(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource default gateway")
	}

	networkInfo := pkg.VMNetworkInfo{
		Tap:         iface,
		MAC:         "", // rely on static IP configuration so we don't care here
		AddressCIDR: addrCIDR,
		GatewayIP:   net.IP(gw),
		Nameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("8.8.4.4")},
	}

	return networkInfo, nil
}

// returns the vCpu's, memory, disksize for a vm size
// memory and disk size is expressed in MiB
func vmSize(size uint8) (uint8, uint64, uint64, error) {
	switch size {
	case 1:
		// 1 vCpu, 2 GiB RAM, 50 GB disk
		return 1, 2 * 1024, 50 * 1024, nil
	case 2:
		// 2 vCpu, 4 GiB RAM, 100 GB disk
		return 2, 4 * 1024, 100 * 1024, nil
	}

	return 0, 0, 0, fmt.Errorf("unsupported vm size %d, only size 1 and 2 are supported", size)
}
