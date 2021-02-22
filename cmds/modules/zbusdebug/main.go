package zbusdebug

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/urfave/cli/v2"
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

	//Module entry point
	Module cli.Command = cli.Command{
		Name:  "zbusdebug",
		Usage: "show status summery for running zbus modules",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "broker",
				Value: "unix:///var/run/redis.sock",
				Usage: "connection string to the message `BROKER`",
			},
			&cli.StringFlag{
				Name:  "module",
				Usage: "debug specific `MODULE`",
			},
		},
		Action: action,
	}
)

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		module       string = cli.String("module")
	)

	cl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to initialize zbus client")
	}

	var debug []string
	if module != "" {
		_, ok := PossibleModules[module]
		if !ok {
			return fmt.Errorf("unknown module")
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
			if len(debug) == 1 {
				return err
			}
		}
	}

	return nil
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
