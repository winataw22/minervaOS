package main

import (
	"context"
	"fmt"

	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// func headerRenderer(c zbus.Client, h *widgets.Paragraph, r *Flag) error {
func headerRenderer(c zbus.Client, h *widgets.Paragraph, r *Flag) error {
	env, err := environment.Get()
	if err != nil {
		return err
	}

	identity := stubs.NewIdentityManagerStub(c)
	nodeID := identity.NodeID()
	var farm string
	farmID, err := identity.FarmID()
	if err != nil {
		farm = "not set"
	} else {
		farm = fmt.Sprintf("%d", farmID)
	}

	h.Text = "\n    Fetching realtime node information... please wait."

	var s string
	s = "          Welcome to [Zero-OS](fg:yellow), [ThreeFold](fg:blue) Autonomous Operating System\n" +
		"\n" +
		" This is node [%s](fg:green) (farmer [%s](fg:green))\n" +
		" running Zero-OS version [%s](fg:blue) (mode [%s](fg:cyan))"

	host := stubs.NewVersionMonitorStub(c)
	ctx := context.Background()
	ch, err := host.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start update stream for version")
	}

	go func() {
		for version := range ch {
			h.Text = fmt.Sprintf(s, nodeID, farm, version.String(), env.RunningMode.String())
			r.Signal()
		}
	}()

	return nil
}
