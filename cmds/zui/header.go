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
		farm = "not attached to a farm"
	} else {
		farm = fmt.Sprintf("%d", farmID)
	}

	format := fmt.Sprintf("Zero OS [%s] Version: %%s NodeID: %s FarmID: %s", env.RunningMode.String(), nodeID.Identity(), farm)

	h.Text = "Zero OS"

	host := stubs.NewVersionMonitorStub(c)
	ctx := context.Background()
	ch, err := host.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start update stream for version")
	}

	go func() {
		for version := range ch {
			v := fmt.Sprintf(format, version.String())
			if h.Text != v {
				h.Text = v
				r.Signal()
			}
		}
	}()

	return nil
}
