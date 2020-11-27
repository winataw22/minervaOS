package main

import (
	"context"
	"crypto/ed25519"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/vishvananda/netlink"

	"github.com/threefoldtech/zos/pkg/network/latency"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/version"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

func main() {
	app.Initialize()

	var (
		root   string
		broker string
		ver    bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/networkd", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	waitYggdrasilBin()

	if err := bootstrap.DefaultBridgeValid(); err != nil {
		log.Fatal().Err(err).Msg("invalid setup")
	}

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zbus broker")
	}

	directory, err := explorerClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to BCDB")
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	// already sends all the interfaces detail we find
	// this won't contains the ndmz IP yet, but this is OK.
	ifaces, err := getLocalInterfaces()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read local network interfaces")
	}
	if err := publishIfaces(ifaces, nodeID, directory); err != nil {
		log.Fatal().Err(err).Msg("failed to publish network interfaces to BCDB")
	}

	exitIface, err := getPubIface(directory, nodeID.Identity())
	pubConfMaster := ""
	if err == nil {
		pubConfMaster = exitIface.Master
	}
	hasPubConf := err == nil

	ndmzNs, err := buildNDMZ(nodeID.Identity(), pubConfMaster)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create ndmz")
	}

	if err := ndmzNs.Create(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to create ndmz")
	}

	if hasPubConf {
		if err := configurePubIface(ndmzNs, exitIface, nodeID); err != nil {
			log.Error().Err(err).Msg("failed to configure public interface")
			os.Exit(1)
		}
	}

	ygg, err := startYggdrasil(ctx, identity.PrivateKey(), ndmzNs)
	if err != nil {
		log.Fatal().Err(err).Msgf("fail to start yggdrasil")
	}

	gw, err := ygg.Gateway()
	if err != nil {
		log.Fatal().Err(err).Msgf("fail read yggdrasil subnet")
	}

	if err := ndmzNs.SetIP6PublicIface(gw); err != nil {
		log.Fatal().Err(err).Msgf("fail to configure yggdrasil subnet gateway IP")
	}

	// send another detail of network interfaces now that ndmz is created
	ndmzIfaces, err := getNdmzInterfaces()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read ndmz network interfaces")
	}
	ifaces = append(ifaces, ndmzIfaces...)

	if err := publishIfaces(ifaces, nodeID, directory); err != nil {
		log.Fatal().Err(err).Msg("failed to publish ndmz network interfaces to BCDB")
	}

	// watch modification of the address on the nic so we can update the explorer
	// with eventual new values
	go startAddrWatch(ctx, nodeID, directory, ifaces)

	log.Info().Msg("start zbus server")
	if err := os.MkdirAll(root, 0750); err != nil {
		log.Fatal().Err(err).Msgf("fail to create module root")
	}

	networker, err := network.NewNetworker(identity, directory, root, ndmzNs, ygg)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating network manager")
	}

	if err := startServer(ctx, broker, networker); err != nil {
		log.Fatal().Err(err).Msg("unexpected error")
	}
}

func startServer(ctx context.Context, broker string, networker pkg.Networker) error {

	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Error().Err(err).Msgf("fail to connect to message broker server")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, networker)

	log.Info().
		Str("broker", broker).
		Uint("worker nr", 1).
		Msg("starting networkd module")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

func waitYggdrasilBin() {
	log.Info().Msg("wait for yggdrasil binary to be available")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 //forever
	_ = backoff.RetryNotify(func() error {
		_, err := exec.LookPath("yggdrasil")
		return err
	}, bo, func(err error, d time.Duration) {
		log.Warn().Err(err).Msgf("yggdrasil binary not found, retying in %s", d.String())
	})
}

func fetchPeerList() yggdrasil.PeerList {
	// Try to fetch public peer
	// If we failed to do so, use the fallback hardcoded peer list
	var pl yggdrasil.PeerList

	// Do not retry more than 4 times
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 4)

	fetchPeerList := func() error {
		p, err := yggdrasil.FetchPeerList()
		if err != nil {
			log.Debug().Err(err).Msg("failed to fetch yggdrasil peers")
			return err
		}
		pl = p
		return nil
	}

	err := backoff.Retry(fetchPeerList, bo)
	if err != nil {
		log.Error().Err(err).Msg("failed to read yggdrasil public peer list online, using fallback")
		pl = yggdrasil.PeerListFallback
	}

	return pl
}

func startYggdrasil(ctx context.Context, privateKey ed25519.PrivateKey, dmz ndmz.DMZ) (*yggdrasil.Server, error) {
	pl := fetchPeerList()
	peersUp := pl.Ups()
	endpoints := make([]string, len(peersUp))
	for i, p := range peersUp {
		endpoints[i] = p.Endpoint
	}

	// filter out the possible yggdrasil public node
	var filter latency.IPFilter
	_, ipv4Only := dmz.(*ndmz.Hidden)
	if ipv4Only {
		// if we are a hidden node,only keep ipv4 public nodes
		filter = latency.IPV4Only
	} else {
		// if we are a dual stack node, filter out all the nodes from the same
		// segment so we do not just connect locally
		npub6IP, err := getDMZNPub6Addr()
		if err != nil {
			return nil, err
		}
		filter = latency.ExcludePrefix(npub6IP[:8])
	}

	ls := latency.NewSorter(endpoints, 5, filter)
	results := ls.Run(ctx)
	if len(results) == 0 {
		return nil, fmt.Errorf("cannot find public yggdrasil peer to connect to")
	}

	// select the best 3 public peers
	peers := make([]string, 3)
	for i := 0; i < 3; i++ {
		if len(results) > i {
			peers[i] = results[i].Endpoint
			log.Info().Str("endpoint", results[i].Endpoint).Msg("yggdrasill public peer selected")
		}
	}

	z, err := zinit.New("")
	if err != nil {
		return nil, err
	}

	cfg := yggdrasil.GenerateConfig(privateKey)
	cfg.Peers = peers

	server := yggdrasil.NewServer(z, &cfg)

	go func() {
		select {
		case <-ctx.Done():
			if err := server.Stop(); err != nil {
				log.Error().Err(err).Msg("error while stopping yggdrasil")
			}
			if err := z.Close(); err != nil {
				log.Error().Err(err).Msg("error while closing zinit client")
			}
			log.Info().Msg("yggdrasil stopped")
		}
	}()

	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}

func startAddrWatch(ctx context.Context, nodeID pkg.Identifier, cl client.Directory, ifaces []types.IfaceInfo) {

	ifaceNames := make([]string, len(ifaces))
	for i, iface := range ifaces {
		ifaceNames[i] = iface.Name
	}
	log.Info().Msgf("watched interfaces %v", ifaceNames)

	f := func() error {
		wl := NewWatchedLinks(ifaceNames, nodeID, cl)
		if err := wl.Forever(ctx); err != nil {
			log.Error().Err(err).Msg("error in address watcher")
			return err
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute
	bo.MaxElapsedTime = 0 // retry forever
	backoff.Retry(f, bo)
}

// instantiate the proper client based on the running mode
func explorerClient() (client.Directory, error) {
	client, err := app.ExplorerClient()
	if err != nil {
		return nil, err
	}

	return client.Directory, nil
}

func buildNDMZ(nodeID string, pubMaster string) (ndmz.DMZ, error) {
	master := pubMaster

	var err error
	if master == "" {
		notify := func(err error, d time.Duration) {
			log.Warn().Err(err).Msgf("did not find a valid IPV6 master address for ndmz, retry in %s", d.String())
		}

		findMaster := func() error {
			var err error
			master, err = ndmz.FindIPv6Master()
			return err
		}

		bo := backoff.NewExponentialBackOff()
		// wait for 2 minute for public ipv6
		bo.MaxElapsedTime = time.Minute * 2
		bo.MaxInterval = time.Second * 10
		err = backoff.RetryNotify(findMaster, bo, notify)
	}

	// if ipv6 found, use dual stack ndmz
	if err == nil && master != "" {
		log.Info().Str("ndmz_npub6_master", master).Msg("network mode dualstack")
		return ndmz.NewDualStack(nodeID, master), nil
	}

	// else use ipv4 only mode
	log.Info().Msg("network mode hidden ipv4 only")
	return ndmz.NewHidden(nodeID), nil
}

func getDMZNPub6Addr() (net.IP, error) {
	netns, err := namespace.GetByName(ndmz.NetNSNDMZ)
	if err != nil {
		return nil, err
	}
	defer netns.Close()

	var ip net.IP
	getIP := func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(ndmz.DMZPub6)
		if err != nil {
			return err
		}
		addrs, err := netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			if addr.IP.IsGlobalUnicast() {
				ip = addr.IP
				return nil
			}
		}

		return nil
	}

	if err = netns.Do(getIP); err != nil {
		return nil, err
	}

	if ip == nil {
		return nil, fmt.Errorf("didn't not find the IP of npub6 interface")
	}

	return ip, nil
}
