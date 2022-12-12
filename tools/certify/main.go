package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
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
	dry       bool
	network   string
}

type Result struct {
	Nodes []struct {
		NodeID uint64
		FarmID uint64
	}
}

func run(opt options) error {
	sudo, err := substrate.NewIdentityFromSr25519Phrase(opt.mnemonics)
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

	grqph := graphql.NewClient(urls.Graphql)
	req := graphql.NewRequest(`
	query MyQuery ($limit: Int!, $offset: Int!){
		nodes(where: {certification_eq: Diy, secure_eq: true}, limit: $limit, offset: $offset, orderBy: nodeID_ASC) {
		  nodeID
		  farmID
		}
	  }
	`)

	possible := 0
	certified := 0

	i := 0
	for {
		req.Var("limit", limit)
		req.Var("offset", i*limit)
		i += 1
		var result Result
		if err := grqph.Run(context.TODO(), req, &result); err != nil {
			return err
		}
		if len(result.Nodes) == 0 {
			break
		}

		for _, node := range result.Nodes {
			log := log.With().
				Uint32("node-id", uint32(node.NodeID)).
				Uint32("farm-id", uint32(node.FarmID)).
				Logger()

			log.Info().Msg("possible node to certify")
			possible++
			if opt.dry {
				continue
			}

			if err := cl.SetNodeCertificate(sudo, uint32(node.NodeID), substrate.NodeCertification{IsCertified: true}); err != nil {
				log.Error().Err(err).Msg("failed to mark node as certified")
				continue
			}

			certified++
		}
	}

	log.Info().Int("count", possible).Msg("found nodes that can be certified")
	log.Info().Int("count", certified).Msg("nodes that has been certified by this run")
	return nil
}

func main() {
	app.Initialize()
	var opt options
	var debug bool

	flag.StringVar(&opt.network, "network", "main", "network (main, test, dev)")
	flag.BoolVar(&opt.dry, "dry-run", false, "print the list of the nodes to be migrated")
	flag.StringVar(&opt.mnemonics, "mnemonics", "", "mnemonics for the sudo key")
	flag.BoolVar(&debug, "debug", false, "show debugging logs")
	flag.Parse()

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
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
