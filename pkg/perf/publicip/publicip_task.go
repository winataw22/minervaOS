package publicip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/perf/graphql"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/vishvananda/netlink"
)

const (
	ValidState   = "valid"
	InvalidState = "invalid"
	SkippedState = "skipped"

	IPsNotMatching      = "public ip does not match farm ip"
	PublicIPDataInvalid = "public ip or gateway data are not valid"
	IPIsUsed            = "ip is already assigned to a contract"
	FetchRealIPFailed   = "failed to get real public IP to the node"
)

var errPublicIPLookup = errors.New("failed to reach public ip service")
var errSkippedValidating = errors.New("skipped, there is a node with less ID available")

const testMacvlan = "pub"
const testNamespace = "pubtestns"

type publicIPValidationTask struct{}

type IPReport struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

var _ perf.Task = (*publicIPValidationTask)(nil)

func NewTask() perf.Task {
	return &publicIPValidationTask{}
}

func (p *publicIPValidationTask) ID() string {
	return "public-ip-validation"
}

func (p *publicIPValidationTask) Cron() string {
	return "0 0 */6 * * *"
}

func (p *publicIPValidationTask) Description() string {
	return "Runs on the least NodeID node in a farm to validate all its IPs."
}

func (p *publicIPValidationTask) Jitter() uint32 {
	return 10 * 60
}

func (p *publicIPValidationTask) Run(ctx context.Context) (interface{}, error) {
	netNS, err := namespace.GetByName(testNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", testNamespace, err)
	}
	cl := perf.GetZbusClient(ctx)
	apiGateway := stubs.NewAPIGatewayStub(cl)
	farmID := environment.MustGet().FarmID

	shouldRun, err := isLeastValidNode(ctx, uint32(farmID), apiGateway)
	if err != nil {
		return nil, fmt.Errorf("failed to check if the node should run public IP verification: %w", err)
	}
	if !shouldRun {
		log.Warn().Msg(errSkippedValidating.Error())
		return errSkippedValidating, nil
	}

	farm, err := apiGateway.GetFarm(ctx, uint32(farmID))
	if err != nil {
		return nil, fmt.Errorf("failed to get farm with id %d: %w", farmID, err)
	}
	var report map[string]IPReport
	err = netNS.Do(func(_ ns.NetNS) error {
		report, err = p.validateIPs(farm.PublicIPs)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to run public IP validation: %w", err)
	}
	return report, nil
}

func (p *publicIPValidationTask) validateIPs(publicIPs []substrate.PublicIP) (map[string]IPReport, error) {
	report := make(map[string]IPReport)
	mv, err := macvlan.GetByName(testMacvlan)
	if err != nil {
		return nil, fmt.Errorf("failed to get macvlan %s in namespace %s: %w", testMacvlan, testNamespace, err)
	}
	// to delete any leftover IPs or routes
	err = deleteAllIPsAndRoutes(mv)
	if err != nil {
		log.Err(err).Send()
	}

	for _, publicIP := range publicIPs {
		report[publicIP.IP] = IPReport{
			State: ValidState,
		}
		if publicIP.ContractID != 0 {
			report[publicIP.IP] = IPReport{
				State:  SkippedState,
				Reason: IPIsUsed,
			}
			continue
		}

		ip, ipNet, routes, err := getIPWithRoute(publicIP)
		if err != nil {
			report[publicIP.IP] = IPReport{
				State:  InvalidState,
				Reason: PublicIPDataInvalid,
			}
			log.Err(err).Send()
			continue
		}
		err = macvlan.Install(mv, nil, ipNet, routes, nil)
		if err != nil {
			report[publicIP.IP] = IPReport{
				State:  InvalidState,
				Reason: PublicIPDataInvalid,
			}
			log.Err(err).Msgf("failed to install macvlan %s with ip %s to namespace %s", testMacvlan, ipNet, testNamespace)
			continue
		}

		realIP, err := getRealPublicIP()
		if errors.Is(err, errPublicIPLookup) {
			report[publicIP.IP] = IPReport{
				State:  InvalidState,
				Reason: PublicIPDataInvalid,
			}
		} else if err != nil {
			report[publicIP.IP] = IPReport{
				State:  SkippedState,
				Reason: FetchRealIPFailed,
			}
		} else if !ip.Equal(realIP) {
			report[publicIP.IP] = IPReport{
				State:  InvalidState,
				Reason: IPsNotMatching,
			}
		}

		err = deleteAllIPsAndRoutes(mv)
		if err != nil {
			log.Err(err).Send()
		}
	}
	err = netlink.LinkSetDown(mv)
	if err != nil {
		return nil, fmt.Errorf("failed to set link down: %w", err)
	}

	return report, nil
}

func isLeastValidNode(ctx context.Context, farmID uint32, apiGateway *stubs.APIGatewayStub) (bool, error) {
	env := environment.MustGet()
	gql := graphql.NewGraphQl(env.GraphQL)

	nodes, err := gql.GetUpNodes(ctx, 0, farmID, 0, false, false)
	if err != nil {
		return false, fmt.Errorf("failed to get farm %d nodes: %w", farmID, err)
	}
	cl := perf.GetZbusClient(ctx)
	registrar := stubs.NewRegistrarStub(cl)
	var nodeID uint32
	err = backoff.Retry(func() error {
		nodeID, err = registrar.NodeID(ctx)
		if err != nil {
			log.Err(err).Msg("failed to get node id")
			return err
		}
		return nil
	}, backoff.NewConstantBackOff(10*time.Second))

	if err != nil {
		return false, fmt.Errorf("failed to get node id: %w", err)
	}

	for _, node := range nodes {
		if node.NodeID >= uint32(nodeID) {
			continue
		}
		n, err := apiGateway.GetNode(ctx, node.NodeID)
		if err != nil {
			return false, fmt.Errorf("failed to get node %d: %w", node.NodeID, err)
		}
		ip, err := getValidNodeIP(n)
		if err != nil {
			return false, err
		}
		// stop at three and quiet output
		err = exec.CommandContext(ctx, "ping", "-c", "3", "-q", ip).Run()
		if err != nil {
			log.Err(err).Msgf("failed to ping node %d", node.NodeID)
			continue
		}
		return false, nil
	}
	return true, nil
}

func getValidNodeIP(node substrate.Node) (string, error) {
	for _, inf := range node.Interfaces {
		if inf.Name != "zos" {
			continue
		}
		if len(inf.IPs) == 0 {
			return "", fmt.Errorf("no private IP available on node %d", node.ID)
		}
		return inf.IPs[0], nil
	}
	return "", fmt.Errorf("failed to get private IP for node %d", node.ID)
}

func getIPWithRoute(publicIP substrate.PublicIP) (net.IP, []*net.IPNet, []*netlink.Route, error) {
	ip, ipNet, err := net.ParseCIDR(publicIP.IP)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse IP %s: %w", publicIP.IP, err)
	}
	ipNet.IP = ip
	gateway := net.ParseIP(publicIP.Gateway)
	if gateway == nil {
		return nil, nil, nil, fmt.Errorf("failed to parse gateway %s: %w", publicIP.Gateway, err)
	}
	route := netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		Gw: gateway,
	}
	return ip, []*net.IPNet{ipNet}, []*netlink.Route{&route}, nil
}

func getRealPublicIP() (net.IP, error) {
	// for testing now, should change to cloudflare
	con, err := net.DialTimeout("tcp", "api.ipify.org:443", 10*time.Second)
	if err != nil {
		return nil, errors.Join(err, errPublicIPLookup)
	}

	defer con.Close()

	cl := http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return con, nil
			},
		},
	}
	response, err := cl.Get("https://api.ipify.org/")
	if err != nil {
		return nil, errors.Join(err, errPublicIPLookup)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("request to get public IP failed with status code %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(string(body)), nil
}

func deleteAllIPsAndRoutes(macvlan netlink.Link) error {
	addresses, err := netlink.AddrList(macvlan, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to list addresses in macvlan %s: %w", testMacvlan, err)
	}
	for _, addr := range addresses {
		err = netlink.AddrDel(macvlan, &addr)
		if err != nil {
			log.Err(err).Msgf("failed to delete address %s", addr)
		}
	}
	routes, err := netlink.RouteList(macvlan, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to list routes in macvlan %s: %w", testMacvlan, err)
	}
	for _, route := range routes {
		err = netlink.RouteDel(&route)
		if err != nil {
			log.Err(err).Msgf("failed to delete route %s", route)
		}
	}
	return err
}
