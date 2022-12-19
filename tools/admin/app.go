package main

import (
	"fmt"
	"os"

	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

var (
	substrateManagers = map[rmb.Network]substrate.Manager{
		rmb.DevelopmentNetwork: substrate.NewManager("wss://tfchain.dev.grid.tf/"),
		rmb.QANetwork:          substrate.NewManager("wss://tfchain.qa.grid.tf/"),
		rmb.TestingNetwork:     substrate.NewManager("wss://tfchain.test.grid.tf/"),
		rmb.ProductionNetwork: substrate.NewManager("wss://tfchain.grid.tf/",
			"wss://02.tfchain.grid.tf/",
			"wss://03.tfchain.grid.tf/",
			"wss://04.tfchain.grid.tf/",
		),
	}
)

func setup(ctx *cli.Context) (identity substrate.Identity, mgr substrate.Manager, err error) {
	mnemonics := ctx.String("mnemonics")
	network := ctx.String("network")
	typ := ctx.String("key-type")

	mgr, ok := substrateManagers[rmb.Network(network)]
	if !ok {
		return nil, nil, fmt.Errorf("unknown network '%s'", network)
	}

	if typ == "sr25519" {
		identity, err = substrate.NewIdentityFromSr25519Phrase(mnemonics)
	} else if typ == "ed25519" {
		identity, err = substrate.NewIdentityFromEd25519Phrase(mnemonics)
	} else {
		err = fmt.Errorf("invalid key type")
	}

	return
}

func getClient(ctx *cli.Context, sub *substrate.Substrate, id substrate.Identity) (rmb.Client, error) {
	twin, err := sub.GetTwinByPubKey(id.PublicKey())
	if err != nil {
		return nil, err
	}

	return rmb.NewProxyClient(twin, id, rmb.Network(ctx.String("network")), nil)
}

func networkShowPublicConfig(ctx *cli.Context) error {
	identity, mgr, err := setup(ctx)
	if err != nil {
		return err
	}

	sub, err := mgr.Substrate()
	if err != nil {
		return err
	}
	defer sub.Close()

	cl, err := getClient(ctx, sub, identity)
	if err != nil {
		return err
	}

	node, err := sub.GetNode(uint32(ctx.Uint64("node")))
	if err != nil {
		return err
	}

	nodeClient := client.NewNodeClient(uint32(node.TwinID), cl)

	cfg, err := nodeClient.NetworkGetPublicConfig(ctx.Context)
	if err != nil {
		return err
	}

	fmt.Println("---")
	enc := yaml.NewEncoder(os.Stdout)
	return enc.Encode(cfg)
}

func networkShowPublicExit(ctx *cli.Context) error {
	identity, mgr, err := setup(ctx)
	if err != nil {
		return err
	}

	sub, err := mgr.Substrate()
	if err != nil {
		return err
	}
	defer sub.Close()

	cl, err := getClient(ctx, sub, identity)
	if err != nil {
		return err
	}

	node, err := sub.GetNode(uint32(ctx.Uint64("node")))
	if err != nil {
		return err
	}

	nodeClient := client.NewNodeClient(uint32(node.TwinID), cl)

	cfg, err := nodeClient.NetworkGetPublicExitDevice(ctx.Context)
	if err != nil {
		return err
	}

	fmt.Println("---")
	if cfg.IsSingle {
		fmt.Println("single setup")
	} else if cfg.IsDual {
		fmt.Printf("dual setup (%s)\n", cfg.AsDualInterface)
	} else {
		return fmt.Errorf("node setup is unknown")
	}

	return nil
}

func networkListPublicExit(ctx *cli.Context) error {
	identity, mgr, err := setup(ctx)
	if err != nil {
		return err
	}

	sub, err := mgr.Substrate()
	if err != nil {
		return err
	}
	defer sub.Close()

	cl, err := getClient(ctx, sub, identity)
	if err != nil {
		return err
	}

	node, err := sub.GetNode(uint32(ctx.Uint64("node")))
	if err != nil {
		return err
	}

	nodeClient := client.NewNodeClient(uint32(node.TwinID), cl)

	infs, err := nodeClient.NetworkListAllInterfaces(ctx.Context)
	if err != nil {
		return err
	}

	fmt.Println("---")
	for name, inf := range infs {
		fmt.Printf("- %s: \"%s\"\n", name, inf.Mac)
	}

	return nil
}

func networkSetPublicExit(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("missing required nic name")
	}

	nic := ctx.Args().First()

	identity, mgr, err := setup(ctx)
	if err != nil {
		return err
	}

	sub, err := mgr.Substrate()
	if err != nil {
		return err
	}
	defer sub.Close()

	cl, err := getClient(ctx, sub, identity)
	if err != nil {
		return err
	}

	node, err := sub.GetNode(uint32(ctx.Uint64("node")))
	if err != nil {
		return err
	}

	nodeClient := client.NewNodeClient(uint32(node.TwinID), cl)

	return nodeClient.NetworkSetPublicExitDevice(ctx.Context, nic)
}
