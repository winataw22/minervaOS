package zui

import (
	"context"
	"sync/atomic"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/zui"
	"github.com/urfave/cli/v2"
)

const module string = "zui"

func trimFloat64(a []float64, size int) []float64 {
	if len(a) > size {
		return a[len(a)-size:]
	}
	return a
}

// signalFlag is a safe flag
type signalFlag int32

// Signal raises flag
func (f *signalFlag) Signal() {
	atomic.SwapInt32((*int32)(f), 1)
}

// Signaled checks if flag was raised, and lowers the flag
func (f *signalFlag) Signaled() bool {
	return atomic.SwapInt32((*int32)(f), 0) == 1
}

// Module is the app entry point
var Module cli.Command = cli.Command{
	Name:  "zui",
	Usage: "starts zos UI",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message broker",
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

func action(ctx *cli.Context) error {
	var (
		msgBrokerCon string = ctx.String("broker")
		workerNr     uint   = ctx.Uint("workers")
	)

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	if err := ui.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize term ui")
	}

	defer ui.Close()

	width, _ := ui.TerminalDimensions()

	header := widgets.NewParagraph()
	header.Border = true
	header.SetRect(0, 0, width, 8)

	netgrid := ui.NewGrid()
	netgrid.Title = "Network"
	netgrid.SetRect(0, 8, width, 14)

	resources := ui.NewGrid()
	resources.Title = "Provision"
	resources.SetRect(0, 14, width, 22)
	resources.Border = false

	errorsParagraph := widgets.NewParagraph()
	errorsParagraph.Title = "Errors"
	errorsParagraph.SetRect(0, 22, width, 26)
	errorsParagraph.Border = true
	errorsParagraph.WrapText = true

	var flag signalFlag

	if err := headerRenderer(ctx.Context, client, header, &flag); err != nil {
		log.Error().Err(err).Msg("failed to start header renderer")
	}

	if err := netRender(client, netgrid, &flag); err != nil {
		log.Error().Err(err).Msg("failed to start net renderer")
	}

	if err := resourcesRender(client, resources, &flag); err != nil {
		log.Error().Err(err).Msg("failed to start resources renderer")
	}

	mod := zui.New(ctx.Context, errorsParagraph, &flag)

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, mod)

	go func() {
		if err := server.Run(ctx.Context); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("unexpected error")
		}

	}()

	render := func() {
		ui.Render(header, netgrid, resources, errorsParagraph)
	}

	render()

	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return nil
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				header.SetRect(0, 0, payload.Width, 3)
				errorsParagraph.SetRect(0, 22, payload.Width, 26)
				// grid.SetRect(0, 3, payload.Width, payload.Height)
				ui.Clear()
				render()
			}
		case <-time.After(1 * time.Second):
			// check if any of the widgets asks for a render
			if flag.Signaled() {
				render()
			}
		}
	}
}
