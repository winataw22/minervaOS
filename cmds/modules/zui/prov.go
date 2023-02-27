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

const (
	gig = 1024 * 1024 * 1024.0
)

func provisionRender(client zbus.Client, grid *ui.Grid, render *signalFlag) error {
	prov := widgets.NewTable()
	prov.Title = "System Used Capacity"
	prov.RowSeparator = false

	prov.Rows = [][]string{
		{"CPU Usage", "", "Memory Usage", ""},
		{"CRU Reserved", "", "MRU Reserved", ""},
		{"SSD Reserved", "", "HDD Reserved", ""},
		{"IPv4 Reserved", ""},
	}

	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1, prov),
		),
	)

	ctx := context.Background()

	monitor := stubs.NewStatisticsStub(client)
	counters, err := monitor.ReservedStream(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	go func() {
		for counter := range counters {
			rows := prov.Rows
			rows[1][1] = fmt.Sprint(counter.CRU)
			rows[1][3] = fmt.Sprintf("%0.00f GB", float64(counter.MRU)/gig)
			rows[2][1] = fmt.Sprintf("%0.00f GB", float64(counter.SRU)/gig)
			rows[2][3] = fmt.Sprintf("%0.00f GB", float64(counter.HRU)/gig)
			rows[3][1] = fmt.Sprint(counter.IPV4U)

			render.Signal()
		}
	}()

	sysMonitor := stubs.NewSystemMonitorStub(client)
	stream, err := sysMonitor.CPU(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start cpu monitor stream")
	}

	go func() {
		for point := range stream {
			prov.Mutex.Lock()
			prov.Rows[0][1] = fmt.Sprintf("%0.00f%%", point.Percent)
			render.Signal()
			prov.Mutex.Unlock()
		}
	}()

	memStream, err := sysMonitor.Memory(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start mem monitor stream")
	}

	go func() {
		for point := range memStream {
			prov.Rows[0][3] = fmt.Sprintf("%0.00f%%", point.UsedPercent)
			render.Signal()
		}
	}()

	return nil
}
