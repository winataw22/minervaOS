package main

import (
	"context"
	_ "fmt"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	_ "github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func addressRender(ctx context.Context, table *widgets.Table, client zbus.Client, render *Flag) error {
	table.Title = "Network"
	table.FillRow = true
	table.RowSeparator = false

	table.Rows = [][]string{
		{"ZOS", "Not configured"},
		{"DMZ", "Not configured"},
		{"YGG", "Not configured"},
		{"PUB", "Not configured"},
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

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1, addresses),
		),
	)
	ctx := context.Background()

	if err := addressRender(ctx, addresses, client, render); err != nil {
		return err
	}

	return nil
}
