package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/app"
)

type Urls struct {
	Substrate string
	Graphql   string
}

type Network string

const (
	MainNetwork Network = "main"
	TestNetwork Network = "test"
	DevNetwork  Network = "dev"
)

var (
	networks = map[Network]Urls{
		MainNetwork: {
			Substrate: "wss://tfchain.grid.tf",
			Graphql:   "https://graphql.grid.tf/graphql",
		},
		TestNetwork: {
			Substrate: "wss://tfchain.test.grid.tf",
			Graphql:   "https://graphql.test.grid.tf/graphql",
		},
		DevNetwork: {
			Substrate: "wss://tfchain.dev.grid.tf",
			Graphql:   "https://graphql.dev.grid.tf/graphql",
		},
	}
)

type options struct {
	mnemonics string
	network   string
	farm      uint64
}

type Result struct {
	Nodes []struct {
		NodeID uint64
		FarmID uint64
	}
}

func run(opt options) error {
	identity, err := substrate.NewIdentityFromSr25519Phrase(opt.mnemonics)
	if err != nil {
		return errors.Wrap(err, "failed to create identity from mnemonics")
	}

	urls, ok := networks[Network(opt.network)]
	if !ok {
		return fmt.Errorf("unknown network '%s'", opt.network)
	}

	const (
		limit = 100
	)

	cl, err := substrate.NewManager(urls.Substrate).Substrate()
	if err != nil {
		return err
	}

	defer cl.Close()

	nodes, err := cl.GetNodes(uint32(opt.farm))
	if err != nil {
		return fmt.Errorf("failed to get farm nodes: %w", err)
	}

	for _, node := range nodes {
		hash, err := cl.SetNodePowerTarget(identity, node, true)
		if err != nil {
			log.Error().Err(err).Uint32("node", node).Msg("failed to set node target to up")
			continue
		}
		log.Info().Uint32("node", node).Str("block", hash.Hex()).Msg("node target was set to up")
	}

	return nil
}

func main() {
	app.Initialize()
	var opt options
	var debug bool

	flag.StringVar(&opt.network, "network", "main", "network (main, test, dev)")
	flag.StringVar(&opt.mnemonics, "mnemonics", "", "mnemonics for the farmer")
	flag.Uint64Var(&opt.farm, "farm", 0, "farm id")
	flag.BoolVar(&debug, "debug", false, "show debugging logs")
	flag.Parse()

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if opt.farm == 0 {
		fmt.Fprintf(os.Stderr, "farm id is required")
	}

	if len(opt.mnemonics) == 0 {
		fmt.Fprintln(os.Stderr, "mnemonics is required")
		os.Exit(1)
	}

	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

}
