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

func cpuRender(client zbus.Client, grid *ui.Grid, render *Flag) error {
	const (
		rows = 8
		cols = 20
	)

	view := widgets.NewTable()
	view.Title = "CPU Percentage Per Core"
	view.FillRow = true
	view.TextStyle.Modifier = ui.ModifierBold
	view.RowSeparator = false
	view.TextAlignment = ui.AlignCenter

	view.Rows = func() [][]string {
		rows := make([][]string, rows)
		for i := 0; i < len(rows); i++ {
			rows[i] = make([]string, cols)
		}

		return rows
	}()

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1, view),
		),
	)

	monitor := stubs.NewSystemMonitorStub(client)
	stream, err := monitor.CPU(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start cpu monitor stream")
	}

	go func() {
		for point := range stream {
			view.Mutex.Lock()
			for i, cpu := range point {
				r := i / rows
				c := i % rows
				if r >= rows || c >= cols {
					// no enough space to show this CPU
					continue
				}

				view.Rows[r][c] = fmt.Sprintf("%0.00f%%", cpu.Percent)
			}
			view.Mutex.Unlock()

			render.Signal()
		}
	}()

	return nil
}
