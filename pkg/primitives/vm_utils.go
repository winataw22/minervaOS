package primitives

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	networkResourceNet = net.IPNet{
		IP:   net.ParseIP("100.64.0.0"),
		Mask: net.IPv4Mask(0xff, 0xff, 0, 0),
	}
)

func (p *Primitives) newYggNetworkInterface(ctx context.Context, wl *gridtypes.WorkloadWithID) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)

	//TODO: if we use `ygg` as a network name. this will conflict
	//if the user has a network that is called `ygg`.
	tapName := tapNameFromName(wl.ID, "ygg")
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
		Public: false,
	}

	return out, nil
}

func (p *Primitives) newPrivNetworkInterface(ctx context.Context, dl gridtypes.Deployment, wl *gridtypes.WorkloadWithID, inf zos.MachineInterface) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)
	netID := zos.NetworkID(dl.TwinID, inf.Network)

	subnet, err := network.GetSubnet(ctx, netID)
	if err != nil {
		return pkg.VMIface{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	if !subnet.Contains(inf.IP) {
		return pkg.VMIface{}, fmt.Errorf("IP %s is not part of local nr subnet %s", inf.IP.String(), subnet.String())
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

	tapName := tapNameFromName(wl.ID, string(inf.Network))
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
		Public:            false,
	}

	return out, nil
}

func (p *Primitives) newPubNetworkInterface(ctx context.Context, deployment gridtypes.Deployment, cfg ZMachine) (pkg.VMIface, error) {
	network := stubs.NewNetworkerStub(p.zbus)
	ipWl, err := deployment.Get(cfg.Network.PublicIP)
	if err != nil {
		return pkg.VMIface{}, err
	}

	tapName := tapNameFromName(ipWl.ID, "pub")

	config, err := p.getPubIPConfig(ipWl)
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
	var gw net.IP
	if !config.IP.Nil() {
		ips = []net.IPNet{
			config.IP.IPNet,
		}
		gw = config.Gateway
	}
	return pkg.VMIface{
		Tap:               pubIface,
		MAC:               mac.String(), // mac so we always get the same IPv6 from slaac
		IPs:               ips,
		IP4DefaultGateway: gw,
		// for now we get ipv6 from slaac, so leave ipv6 stuffs this empty
		Public: true,
	}, nil
}

// Get the public ip, and the gateway from the reservation ID
func (p *Primitives) getPubIPConfig(wl *gridtypes.WorkloadWithID) (result zos.PublicIPResult, err error) {
	if wl.Type != zos.PublicIPv4Type && wl.Type != zos.PublicIPType {
		return result, fmt.Errorf("workload for public IP is of wrong type")
	}

	if wl.Result.State != gridtypes.StateOk {
		return result, fmt.Errorf("public ip workload is not okay")
	}

	if err := wl.Result.Unmarshal(&result); err != nil {
		return result, errors.Wrap(err, "failed to load ip result")
	}

	return result, nil
}

func getFlistInfo(imagePath string) (FListInfo, error) {
	image := imagePath + "/image.raw"
	log.Debug().Str("file", image).Msg("checking image")
	_, err := os.Stat(image)

	if os.IsNotExist(err) {
		return FListInfo{}, nil
	} else if err != nil {
		return FListInfo{}, errors.Wrap(err, "couldn't stat /image.raw")
	}

	return FListInfo{ImagePath: image}, nil
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
	if _, err := toml.DecodeReader(f, &startup); err != nil {
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
