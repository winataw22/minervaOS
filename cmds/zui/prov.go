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
	prov.Title = "System Load"
	prov.RowSeparator = false

	prov.Rows = [][]string{
		{"CPU Usage", "", "Memory Usage", ""},
		{"Containers", "", "Volumes", ""},
		{"Networks", "", "VMs", ""},
		{"ZDB NS", "", "Debug", ""},
	}

	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1, prov),
		),
	)

	ctx := context.Background()

	monitor := stubs.NewProvisionStub(client)
	counters, err := monitor.Counters(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	go func() {
		for counter := range counters {
			rows := prov.Rows
			rows[1][1] = fmt.Sprint(counter.Container)
			rows[1][3] = fmt.Sprint(counter.Volume)
			rows[2][1] = fmt.Sprint(counter.Network)
			rows[2][3] = fmt.Sprint(counter.VM)
			rows[3][1] = fmt.Sprint(counter.ZDB)
			rows[3][3] = fmt.Sprint(counter.Debug)

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

	memoryMonitor := stubs.NewSystemMonitorStub(client)
	memStream, err := memoryMonitor.Memory(context.Background())
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
