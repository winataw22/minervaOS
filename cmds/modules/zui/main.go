package zui

import (
	"sync/atomic"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/urfave/cli/v2"
)

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

//Signaled checks if flag was raised, and lowers the flag
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
	},

	Action: action,
}

func action(ctx *cli.Context) error {
	var (
		msgBrokerCon string = ctx.String("broker")
	)

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus")
	}

	if err := ui.Init(); err != nil {
		return errors.Wrap(err, "failed to initialize term ui")
	}

	defer ui.Close()

	width, _ := ui.TerminalDimensions()

	header := widgets.NewParagraph()
	header.Border = true
	header.SetRect(0, 0, width, 7)

	netgrid := ui.NewGrid()
	netgrid.Title = "Network"
	netgrid.SetRect(0, 7, width, 14)

	provision := ui.NewGrid()
	provision.Title = "Provision"
	provision.SetRect(0, 14, width, 20)
	provision.Border = false

	var flag signalFlag

	if err := headerRenderer(ctx.Context, client, header, &flag); err != nil {
		log.Error().Err(err).Msg("failed to start header renderer")
	}

	if err := netRender(client, netgrid, &flag); err != nil {
		log.Error().Err(err).Msg("failed to start net renderer")
	}

	if err := provisionRender(client, provision, &flag); err != nil {
		log.Error().Err(err).Msg("failed to start net renderer")
	}

	render := func() {
		ui.Render(header, netgrid, provision)
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
