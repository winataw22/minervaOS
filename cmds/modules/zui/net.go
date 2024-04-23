package zui

import (
	"context"
	"fmt"
	"net"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	_ "github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func addressRender(ctx context.Context, table *widgets.Table, client zbus.Client, render *signalFlag) error {
	table.Title = "Network"
	table.FillRow = true
	table.RowSeparator = false

	table.Rows = [][]string{
		{"ZOS", "Not configured"},
		{"DMZ", "Not configured"},
		{"YGG", "Not configured"},
		{"PUB", "Not configured"},
		{"DUL", "Not configured"},
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
			if a.IP == nil || len(a.IP) == 0 {
				continue
			}

			if buf.Len() > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(a.String())
		}

		return buf.String()
	}

	a, err := stub.Interfaces(ctx, types.DefaultBridge, "")
	if err != nil {
		return err
	}

	table.Rows[0][1] = toString(a[types.DefaultBridge].IPs)

	go func() {
		for {
			render.Signal()

			table.ColumnWidths = []int{6, table.Size().X - 9}
			select {
			case a := <-zos:
				table.Rows[0][1] = toString(a)
			case a := <-dmz:
				table.Rows[1][1] = toString(a)
			case a := <-ygg:
				table.Rows[2][1] = toString(a)
			case a := <-pub:
				str := "no public config"
				if a.HasPublicConfig {
					str = toString([]net.IPNet{a.IPv4.IPNet, a.IPv6.IPNet})
				}
				table.Rows[3][1] = str
			}

			exit, err := stub.GetPublicExitDevice(ctx)
			dual := exit.String()
			if err != nil {
				dual = fmt.Sprintf("error: %s", err)
			}

			table.Rows[4][1] = dual
		}
	}()

	return nil
}

func netRender(client zbus.Client, grid *ui.Grid, render *signalFlag) error {
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
