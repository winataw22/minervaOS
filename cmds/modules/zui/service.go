package zui

import (
	"context"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/threefoldtech/zbus"

	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	activeStatus     = "Active"
	InProgressStatus = "In progress"
	FailedStatus     = "Failed"
	InactiveStatus   = "Inactive"

	networkdService     = "Networkd"
	registrationService = "Registerar"
	statisticsService   = "Statistics"
	containerdService   = "Containerd"
	storagedService     = "Storaged"
	nodedService        = "Noded"
	PowerdService       = "Powerd"
)

// serviceRender monitors the state of some services
func serviceRender(ctx context.Context, client zbus.Client, grid *ui.Grid, render *signalFlag) (chan bool, error) {
	services := widgets.NewTable()

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1, services),
		),
	)

	services.Title = "Services Status"
	services.RowSeparator = false

	services.Rows = [][]string{
		{"", "Status"},
		{networkdService, InProgressStatus},
		{registrationService, InactiveStatus},
		{statisticsService, InactiveStatus},
		{containerdService, InactiveStatus},
		{storagedService, InactiveStatus},
		{nodedService, InactiveStatus},
		{PowerdService, InactiveStatus},
	}

	render.Signal()

	done := make(chan bool)

	go func() {
		type serviceStatus struct {
			service string
			status  bool
		}
		servicesStatus := make(chan serviceStatus)

		go func() {
			status := getNetworkStatus(ctx, client)
			servicesStatus <- serviceStatus{service: networkdService, status: status}
		}()

		go func() {
			getStatisticsStatus(ctx, client)
			servicesStatus <- serviceStatus{service: statisticsService, status: true}
		}()

		go func() {
			getContainerdStatus(ctx, client)
			servicesStatus <- serviceStatus{service: containerdService, status: true}
		}()
		go func() {
			getStoragedStatus(ctx, client)
			servicesStatus <- serviceStatus{service: storagedService, status: true}
		}()
		go func() {
			getNodedStatus(ctx, client)
			servicesStatus <- serviceStatus{service: nodedService, status: true}
		}()
		go func() {
			getPowerdStatus(ctx, client)
			servicesStatus <- serviceStatus{service: PowerdService, status: true}
		}()

		for i := 0; i < 6; i++ {
			services.Rows[2][1] = getRegistrarStatus(ctx, client)

			service := <-servicesStatus
			status := red(FailedStatus)
			if service.status {
				status = green(activeStatus)
			}

			switch service.service {
			case networkdService:
				services.Rows[1][1] = status
			case statisticsService:
				services.Rows[3][1] = status
			case containerdService:
				services.Rows[4][1] = status
			case storagedService:
				services.Rows[5][1] = status
			case nodedService:
				services.Rows[6][1] = status
			case PowerdService:
				services.Rows[7][1] = status
			}

			render.Signal()
		}

		done <- true
	}()

	return done, nil
}

func getRegistrarStatus(ctx context.Context, client zbus.Client) string {
	register := stubs.NewRegistrarStub(client)
	if _, err := register.NodeID(ctx); err != nil {
		if isInProgressError(err) {
			return InProgressStatus
		}
		return red(FailedStatus)
	}
	return green(activeStatus)
}

func getNetworkStatus(ctx context.Context, client zbus.Client) bool {
	network := stubs.NewNetworkerStub(client)

	if err := network.Ready(ctx); err != nil {
		return false
	}
	return true
}

func getStatisticsStatus(ctx context.Context, client zbus.Client) {
	statistics := stubs.NewStatisticsStub(client)
	statistics.Total(ctx)
}

func getContainerdStatus(ctx context.Context, client zbus.Client) {
	statistics := stubs.NewContainerModuleStub(client)
	statistics.ListNS(ctx)
}

func getStoragedStatus(ctx context.Context, client zbus.Client) {
	storaged := stubs.NewStorageModuleStub(client)
	storaged.Devices(ctx)
}

func getNodedStatus(ctx context.Context, client zbus.Client) {
	noded := stubs.NewSystemMonitorStub(client)
	noded.NodeID(ctx)
}

func getPowerdStatus(ctx context.Context, client zbus.Client) {
	powerd := stubs.NewIdentityManagerStub(client)
	powerd.NodeID(ctx)
}
