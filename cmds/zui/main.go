package main

import (
	"flag"
	"sync/atomic"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/version"
)

func trimFloat64(a []float64, size int) []float64 {
	if len(a) > size {
		return a[len(a)-size:]
	}
	return a
}

// Flag is a safe flag
type Flag int32

// Signal raises flag
func (f *Flag) Signal() {
	atomic.SwapInt32((*int32)(f), 1)
}

//Signaled checks if flag was raised, and lowers the flag
func (f *Flag) Signaled() bool {
	return atomic.SwapInt32((*int32)(f), 0) == 1
}

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zbus")
	}

	if err := ui.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize term ui")
	}

	defer ui.Close()

	width, _ := ui.TerminalDimensions()

	header := widgets.NewParagraph()
	header.Border = true
	header.SetRect(0, 0, width, 6)

	netgrid := ui.NewGrid()
	netgrid.Title = "Network"
	netgrid.SetRect(0, 6, width, 12)

	provision := ui.NewGrid()
	provision.Title = "Provision"
	provision.SetRect(0, 12, width, 18)
	provision.Border = false

	var flag Flag

	if err := headerRenderer(client, header, &flag); err != nil {
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
				return
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
