package zui

import (
	"context"
	"fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	gig     = 1024 * 1024 * 1024.0
	mb      = 1024 * 1024.0
	loading = "Loading..."
)

func resourcesRender(client zbus.Client, grid *ui.Grid, render *signalFlag) error {
	prov := widgets.NewTable()
	usage := widgets.NewTable()
	prov.FillRow = true

	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(.6, prov),
			ui.NewCol(.4, usage),
		),
	)

	if err := provRender(client, render, prov); err != nil {
		return errors.Wrap(err, "failed to render system provisioned resources")
	}

	if err := usageRender(client, render, usage); err != nil {
		return errors.Wrap(err, "failed to render system resources usage")
	}

	return nil
}

func provRender(client zbus.Client, render *signalFlag, prov *widgets.Table) error {
	prov.Title = "System Resources"
	prov.RowSeparator = false

	prov.Rows = [][]string{
		{"", "Total", "Reserved"},
		{"CRU", loading, loading},
		{"Memory", loading, loading},
		{"SSD", loading, loading},
		{"HDD", loading, loading},
		{"IPv4", loading, loading},
	}

	monitor := stubs.NewStatisticsStub(client)

	total := monitor.Total(context.Background())
	assignTotalResources(prov, total)
	render.Signal()

	reserved, err := monitor.ReservedStream(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	go func() {
		for counter := range reserved {
			rows := prov.Rows
			rows[1][2] = fmt.Sprint(counter.CRU)
			rows[2][2] = fmt.Sprintf("%0.00f GB", float64(counter.MRU)/gig)
			rows[3][2] = fmt.Sprintf("%0.00f GB", float64(counter.SRU)/gig)
			rows[4][2] = fmt.Sprintf("%0.00f GB", float64(counter.HRU)/gig)
			rows[5][2] = fmt.Sprint(counter.IPV4U)

			render.Signal()
		}
	}()

	return nil
}

func usageRender(client zbus.Client, render *signalFlag, usage *widgets.Table) error {
	usage.Title = "Usage"
	usage.RowSeparator = false
	usage.FillRow = true

	usage.Rows = [][]string{
		{"CPU", loading},
		{"Memory", loading},
	}

	sysMonitor := stubs.NewSystemMonitorStub(client)
	cpuStream, err := sysMonitor.CPU(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start cpu monitor stream")
	}

	go func() {
		for point := range cpuStream {
			usage.Rows[0][1] = fmt.Sprintf("%0.00f%%", point.Percent)
			render.Signal()
		}
	}()

	memStream, err := sysMonitor.Memory(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start mem monitor stream")
	}

	go func() {
		for point := range memStream {
			usage.Rows[1][1] = fmt.Sprintf("%0.00f MB", float64(point.Used/mb))
			render.Signal()
		}
	}()

	return nil
}

func assignTotalResources(prov *widgets.Table, total gridtypes.Capacity) {
	rows := prov.Rows
	rows[1][1] = fmt.Sprint(total.CRU)
	rows[2][1] = fmt.Sprintf("%0.00f GB", float64(total.MRU)/gig)
	rows[3][1] = fmt.Sprintf("%0.00f GB", float64(total.SRU)/gig)
	rows[4][1] = fmt.Sprintf("%0.00f GB", float64(total.HRU)/gig)
	rows[5][1] = fmt.Sprint(total.IPV4U)
}
