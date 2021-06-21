package primitives

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Kubernetes type
type Kubernetes = zos.Kubernetes

// KubernetesResult type
type KubernetesResult = zos.KubernetesResult

func (p *Primitives) buildNetworkInfo(ctx context.Context, deployment gridtypes.Deployment, iface string, pubIface string, cfg ZMachine) (pkg.VMNetworkInfo, error) {
	network := stubs.NewNetworkerStub(p.zbus)
	netConfig := cfg.Network.Interfaces[0]
	netID := zos.NetworkID(deployment.TwinID, netConfig.Network)
	subnet, err := network.GetSubnet(ctx, netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	if !subnet.Contains(netConfig.IP) {
		return pkg.VMNetworkInfo{}, fmt.Errorf("IP %s is not part of local nr subnet %s", netConfig.IP.String(), subnet.String())
	}

	privNet, err := network.GetNet(ctx, netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network range")
	}

	addrCIDR := net.IPNet{
		IP:   netConfig.IP,
		Mask: subnet.Mask,
	}

	gw4, gw6, err := network.GetDefaultGwIP(ctx, netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not get network resource default gateway")
	}

	privIP6, err := network.GetIPv6From4(ctx, netID, netConfig.IP)
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

	pubIP := cfg.Network.PublicIP
	if len(pubIP) > 0 {
		// A public ip is set, load the reservation, extract the ip and make a config
		// for it
		ipWl, err := deployment.Get(pubIP)
		if err != nil {
			return pkg.VMNetworkInfo{}, err
		}

		pubIP, pubGw, err := p.getPubIPConfig(ipWl)
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
func (p *Primitives) getPubIPConfig(wl *gridtypes.WorkloadWithID) (ip net.IPNet, gw net.IP, err error) {

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

func getFlistInfo(imagePath string) (FListInfo, error) {
	// entities, err := ioutil.ReadDir(imagePath)
	// if err != nil {
	// 	return FListInfo{}, err
	// }
	// out, err := exec.Command("mountpoint", imagePath).CombinedOutput()
	// if err != nil {
	// 	return FListInfo{}, err
	// }
	// log.Debug().Msgf("mountpoint: %s", string(out))
	// log.Debug().Str("mnt", imagePath).Msg("listing files in")
	// for _, ent := range entities {
	// 	log.Debug().Str("file", ent.Name()).Msg("file found")
	// }

	kernel := filepath.Join(imagePath, "kernel")
	log.Debug().Str("file", kernel).Msg("checking kernel")
	if _, err := os.Stat(kernel); os.IsNotExist(err) {
		return FListInfo{Container: true}, nil
	} else if err != nil {
		return FListInfo{}, errors.Wrap(err, "couldn't stat /kernel")
	}

	initrd := filepath.Join(imagePath, "initrd")
	log.Debug().Str("file", initrd).Msg("checking initrd")
	if _, err := os.Stat(initrd); os.IsNotExist(err) {
		initrd = "" // optional
	} else if err != nil {
		return FListInfo{}, errors.Wrap(err, "couldn't state /initrd")
	}

	image := imagePath + "/image.raw"
	log.Debug().Str("file", image).Msg("checking image")
	if _, err := os.Stat(image); err != nil {
		return FListInfo{}, errors.Wrap(err, "couldn't stat /image.raw")
	}

	return FListInfo{Initrd: initrd, Kernel: kernel, ImagePath: image}, nil
}
