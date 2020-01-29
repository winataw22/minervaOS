package main

import (
	"context"
	"fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func provisionRender(client zbus.Client, grid *ui.Grid, render *Flag) error {

	prov := widgets.NewTable()
	prov.Title = "Workloads"
	prov.RowSeparator = false

	prov.Rows = [][]string{
		{"Containers", ""},
		{"Volumes", ""},
		{"Networks", ""},
		{"VMs", ""},
		{"ZDB Namespaces", ""},
		{"Debug", ""},
	}

	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1, prov),
		),
	)

	ctx := context.Background()

	monitor := stubs.NewProvisionMonitorStub(client)
	counters, err := monitor.Counters(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	go func() {
		for counter := range counters {
			rows := prov.Rows
			rows[0][1] = fmt.Sprint(counter.Container)
			rows[1][1] = fmt.Sprint(counter.Volume)
			rows[2][1] = fmt.Sprint(counter.Network)
			rows[3][1] = fmt.Sprint(counter.VM)
			rows[4][1] = fmt.Sprint(counter.ZDB)
			rows[5][1] = fmt.Sprint(counter.Debug)

			render.Signal()
		}
	}()

	return nil
}
