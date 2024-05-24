package zosapi

import (
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func (g *ZosAPI) SetupRoutes(router *peer.Router) {

	root := router.SubRoute("zos")
	root.Use(g.log)
	system := root.SubRoute("system")
	system.WithHandler("version", g.systemVersionHandler)
	system.WithHandler("dmi", g.systemDMIHandler)
	system.WithHandler("hypervisor", g.systemHypervisorHandler)
	system.WithHandler("diagnostics", g.systemDiagnosticsHandler)

	perf := root.SubRoute("perf")
	perf.WithHandler("get", g.perfGetHandler)
	perf.WithHandler("get_all", g.perfGetAllHandler)

	gpu := root.SubRoute("gpu")
	gpu.WithHandler("list", g.gpuListHandler)

	storage := root.SubRoute("storage")
	storage.WithHandler("pools", g.storagePoolsHandler)

	network := root.SubRoute("network")
	network.WithHandler("list_wg_ports", g.networkListWGPortsHandler)
	network.WithHandler("public_config_get", g.networkPublicConfigGetHandler)
	network.WithHandler("interfaces", g.networkInterfacesHandler)
	network.WithHandler("has_ipv6", g.networkHasIPv6Handler)
	network.WithHandler("list_public_ips", g.networkListPublicIPsHandler)
	network.WithHandler("list_private_ips", g.networkListPrivateIPsHandler)

	statistics := root.SubRoute("statistics")
	statistics.WithHandler("get", g.statisticsGetHandler)

	deployment := root.SubRoute("deployment")
	deployment.WithHandler("deploy", g.deploymentDeployHandler)
	deployment.WithHandler("update", g.deploymentUpdateHandler)
	deployment.WithHandler("delete", g.deploymentDeleteHandler)
	deployment.WithHandler("get", g.deploymentGetHandler)
	deployment.WithHandler("list", g.deploymentListHandler)
	deployment.WithHandler("changes", g.deploymentChangesHandler)

	admin := root.SubRoute("admin")
	admin.Use(g.authorized)
	admin.WithHandler("interfaces", g.adminInterfacesHandler)
	admin.WithHandler("set_public_nic", g.adminSetPublicNICHandler)
	admin.WithHandler("get_public_nic", g.adminGetPublicNICHandler)
}
