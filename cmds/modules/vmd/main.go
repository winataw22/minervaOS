package vmd

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/vm"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const module = "vmd"

// Module entry point
var Module cli.Command = cli.Command{
	Name:  module,
	Usage: "handles virtual machines creation",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/vmd",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.UintFlag{
			Name:  "workers",
			Usage: "number of workers `N`",
			Value: 1,
		},
	},
	Action: action,
}

// copy files from src dir to dst dir works at one level only for
// our specific use case
func copyDepth1(src, dst string) error {

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && src == path {
			return nil
		} else if info.IsDir() {
			return filepath.SkipDir
		}

		srcF, err := os.Open(path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, filepath.Base(path))
		dstF, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(dstF, srcF)
		return err
	})
}

func action(cli *cli.Context) error {
	var (
		moduleRoot   string = cli.String("root")
		msgBrokerCon string = cli.String("broker")
		workerNr     uint   = cli.Uint("workers")
	)

	if err := os.MkdirAll(moduleRoot, 0755); err != nil {
		log.Fatal().Err(err).Str("root", moduleRoot).Msg("Failed to create module root")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to message broker")
	}
	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	config, err := cache.VolatileDir(module, 50*1024*1024) //50mb volatile directory
	if err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "failed to create vmd volatile storage")
	}

	mod, err := vm.NewVMModule(client, moduleRoot, config)
	if err != nil {
		return errors.Wrap(err, "failed to create a new instance of manager")
	}

	/*
		* So, this is a not very clean upgrade process to fix a bug related
		* to restart a zos node.
		* What happens is on a fresh start, VMD will start monitoring the config
		* files (from a previous boot) hence it will immediately start to boot
		* the vms while there might be a lot of dependencies that are not ready yet
		* (for example networks or tap devices). So the right way to do this is
		* to wait for provisiond to re-comission the vm hence the order is granteed
		*
		* So this mean the VMD configs should actually be volatile so they are gone
		* if the machine is rebooted. The problem is how to know if vmd is starting
		* because of an update (hence vms must make sure current running vms should
		 * stay running). Or it's a fresh update hence config files should be ignored
		 * (note this is because configs used to be stored on persisted storage)
	*/
	oldConfig := filepath.Join(moduleRoot, vm.ConfigDir)
	if vms, err := mod.List(); err != nil {
		return errors.Wrap(err, "failed to list running vms")
	} else if len(vms) > 0 {
		log.Info().Msg("vmd is updating")
		// if there are running VMs then we assume this is an "update"
		// in that case we need to move the files from the deprecated (persisted)
		// moduleRoot directory to the volatile directory.
		err := copyDepth1(oldConfig, config)
		if os.IsNotExist(err) {
			log.Info().Msg("no config migration needed")
		} else if err != nil {
			log.Error().Err(err).Msg("failed to migrate config files to volatile root")
		} else {
			log.Info().Msg("config migration is done")
		}
	}

	// in all cases the oldCOnfig should be gone
	if err := os.RemoveAll(oldConfig); err != nil {
		log.Error().Err(err).Msg("failed to clean up deprecated module root")
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, mod)

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	mod.Monitor(ctx)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting vmd module")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}
