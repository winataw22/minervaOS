package zui

import (
	"context"
	"fmt"
	"sort"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func diskRender(client zbus.Client, grid *ui.Grid, render *signalFlag) error {
	const (
		mega = 1024 * 1024
	)

	pools := widgets.NewTable()
	pools.Title = "Storage Pools"
	pools.RowSeparator = false
	pools.TextAlignment = ui.AlignCenter
	pools.Rows = [][]string{
		{"POOL", "TOTAL", "USED"},
	}

	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1, pools),
		),
	)

	ctx := context.Background()

	monitor := stubs.NewStorageModuleStub(client)
	stats, err := monitor.Monitor(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	var keys []string

	go func() {
		for s := range stats {
			if len(keys) != len(s) {
				for key := range s {
					keys = append(keys, key)
				}
				sort.Strings(keys)
			}

			rows := pools.Rows[:1]

			for _, key := range keys {
				pool := s[key]
				rows = append(rows,
					[]string{
						key,
						fmt.Sprintf("%d MB", pool.Total/mega),
						fmt.Sprintf("%0.00f%%", 100.0*(float64(pool.Used)/float64(pool.Total))),
					},
				)
			}

			pools.Rows = rows
			render.Signal()
		}
	}()

	return nil
}
