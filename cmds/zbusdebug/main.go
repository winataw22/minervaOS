package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/version"
	"gopkg.in/yaml.v2"

	"github.com/rs/zerolog/log"
)

var (
	// PossibleModules is a list of all know zos modules. the modules must match
	// the module name declared by the server. Hence, we collect them here for
	// validation
	PossibleModules = map[string]struct{}{
		"storage":   struct{}{},
		"monitor":   struct{}{},
		"identityd": struct{}{},
		"vmd":       struct{}{},
		"flist":     struct{}{},
		"network":   struct{}{},
		"container": struct{}{},
		"provision": struct{}{},
	}
)

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		module       string
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&module, "module", "", "debug specific module")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize zbus client")
	}

	var debug []string
	if module != "" {
		_, ok := PossibleModules[module]
		if !ok {
			log.Fatal().Msg("unknown module")
		}

		debug = append(debug, module)
	} else {
		for module := range PossibleModules {
			debug = append(debug, module)
		}
	}
	parent := context.Background()
	for _, module := range debug {
		if err := printModuleStatus(parent, cl, module); err != nil {
			log.Error().Str("module", module).Err(err).Msg("failed to get status for module")
		}
	}

}

func printModuleStatus(ctx context.Context, cl zbus.Client, module string) error {
	fmt.Println("## Status for ", module)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	status, err := cl.Status(ctx, module)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(os.Stdout)
	defer enc.Close()

	enc.Encode(status)
	fmt.Println()
	return nil
}
