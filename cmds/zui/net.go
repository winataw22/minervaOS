package main

import (
	"context"
	"fmt"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func addressRender(ctx context.Context, table *widgets.Table, client zbus.Client, render *Flag) error {
	table.Title = "Addresses"
	table.FillRow = true
	table.RowSeparator = false

	table.Rows = [][]string{
		{"ZOS", "Not Configured"},
		{"DMZ", "Not Configured"},
		{"Ygg", "Not Configured"},
		{"Public", "Not Configured"},
	}

	stub := stubs.NewNetworkerStub(client)
	zos, err := stub.ZOSAddresses(ctx)
	if err != nil {
		return err
	}

	dmz, err := stub.DMZAddresses(ctx)
	if err != nil {
		return err
	}

	ygg, err := stub.YggAddresses(ctx)
	if err != nil {
		return err
	}

	pub, err := stub.PublicAddresses(ctx)
	if err != nil {
		return err
	}

	toString := func(al pkg.NetlinkAddresses) string {
		var buf strings.Builder
		for _, a := range al {
			if buf.Len() > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(a.String())
		}

		return buf.String()
	}

	go func() {
		for {
			table.ColumnWidths = []int{6, table.Size().X - 9}
			select {
			case a := <-zos:
				table.Rows[0][1] = toString(a)
			case a := <-dmz:
				table.Rows[1][1] = toString(a)
			case a := <-ygg:
				table.Rows[2][1] = toString(a)
			case a := <-pub:
				table.Rows[3][1] = toString(a)
			}

			render.Signal()
		}
	}()

	return nil
}

func netRender(client zbus.Client, grid *ui.Grid, render *Flag) error {
	addresses := widgets.NewTable()
	statistics := widgets.NewTable()

	statistics.Title = "Traffic"
	statistics.RowSeparator = false
	statistics.Rows = [][]string{
		{"NIC", "SENT", "RECV"},
	}

	grid.Set(
		ui.NewRow(1./6,
			ui.NewCol(1, addresses),
		),
		ui.NewRow(5.0/6,
			ui.NewCol(1, statistics),
		),
	)

	ctx := context.Background()

	if err := addressRender(ctx, addresses, client, render); err != nil {
		return err
	}

	monitor := stubs.NewSystemMonitorStub(client)
	stats, err := monitor.Nics(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	go func() {
		for s := range stats {
			// keep the header
			rows := statistics.Rows[:1]
			for _, nic := range s {
				if nic.Name == "lo" {
					continue
				}
				rows = append(rows,
					[]string{
						nic.Name,
						fmt.Sprintf("%d KB", nic.RateOut/1024),
						fmt.Sprintf("%d KB", nic.RateIn/1024),
					},
				)
			}

			statistics.Rows = rows
			render.Signal()
		}
	}()

	return nil
}
