package zui

import (
	"context"
	"fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func memRender(client zbus.Client, grid *ui.Grid, render *signalFlag) error {
	const (
		mega = 1024 * 1024
	)

	percent := widgets.NewGauge()
	percent.Percent = 0
	percent.BarColor = ui.ColorGreen
	percent.Title = "Memory Percent"

	total := widgets.NewParagraph()
	total.Title = "Memory"

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1./2, percent),
			ui.NewCol(1./2, total),
		),
	)

	monitor := stubs.NewSystemMonitorStub(client)
	stream, err := monitor.Memory(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start mem monitor stream")
	}

	go func() {
		for point := range stream {
			percent.Percent = int(point.UsedPercent)
			if point.UsedPercent < 50 {
				percent.BarColor = ui.ColorGreen
			} else if point.UsedPercent >= 50 && point.UsedPercent < 90 {
				percent.BarColor = ui.ColorMagenta
			} else if point.UsedPercent > 90 {
				percent.BarColor = ui.ColorRed
			}

			total.Text = fmt.Sprintf("Total: %d MB, Used: %d MB, Free: %d MB", point.Total/mega, point.Used/mega, point.Free/mega)
			render.Signal()
		}
	}()

	return nil
}
