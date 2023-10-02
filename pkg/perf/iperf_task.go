package perf

import (
	"context"
	"net"
	"os/exec"

	goIperf "github.com/BGrewell/go-iperf"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/iperf"
)

// IperfTest for iperf tcp/udp tests
type IperfTest struct {
	taskID   string
	schedule string
}

// IperfResult for iperf test results
type IperfResult struct {
	UploadSpeed   float64                      `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64                      `json:"download_speed"` // in bit/sec
	NodeID        uint32                       `json:"node_id"`
	NodeIpv4      string                       `json:"node_ip"`
	TestType      string                       `json:"test_type"`
	Error         string                       `json:"error"`
	CpuReport     goIperf.CpuUtilizationReport `json:"cpu_report"`
}

// NewIperfTest creates a new iperf test
func NewIperfTest() IperfTest {
	return IperfTest{taskID: "iperf", schedule: "0 0 */6 * * *"}
}

// ID returns the ID of the tcp task
func (t *IperfTest) ID() string {
	return t.taskID
}

// Cron returns the schedule for the tcp task
func (t *IperfTest) Cron() string {
	return t.schedule
}

// Run runs the tcp test and returns the result
func (t *IperfTest) Run(ctx context.Context) (interface{}, error) {
	env := environment.MustGet()
	g := NewGraphQl(env.GraphQL)

	// get nodes
	freeFarmNodes, err := g.ListPublicNodes(ctx, 0, 1, 0, true, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list freefarm nodes from graphql")
	}

	nodes, err := g.ListPublicNodes(ctx, 12, 0, 1, true, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list random nodes from graphql")
	}

	nodes = append(nodes, freeFarmNodes...)

	_, err = exec.LookPath("iperf")
	if err != nil {
		return nil, err
	}

	var results []IperfResult
	for _, node := range nodes {
		clientIP, _, err := net.ParseCIDR(node.PublicConfig.Ipv4)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse ipv4 address")
			continue
		}

		clientIPv6, _, err := net.ParseCIDR(node.PublicConfig.Ipv6)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse ipv6 address")
			continue
		}

		// TCP
		res := t.runIperfTest(ctx, clientIP.String(), true)
		res.NodeID = node.NodeID
		results = append(results, res)

		res = t.runIperfTest(ctx, clientIPv6.String(), true)
		res.NodeID = node.NodeID
		results = append(results, res)

		// UDP
		res = t.runIperfTest(ctx, clientIP.String(), false)
		res.NodeID = node.NodeID
		results = append(results, res)

		res = t.runIperfTest(ctx, clientIPv6.String(), false)
		res.NodeID = node.NodeID
		results = append(results, res)
	}

	return results, nil
}

func (t *IperfTest) runIperfTest(ctx context.Context, clientIP string, tcp bool) IperfResult {
	iperfClient := goIperf.NewClient(clientIP)
	iperfClient.SetBandwidth("1M")
	iperfClient.SetPort(iperf.IperfPort)
	iperfClient.SetInterval(20)
	iperfClient.SetJSON(true)

	if !tcp {
		iperfClient.SetLength("16B")
		iperfClient.SetProto(goIperf.PROTO_UDP)
	}

	err := iperfClient.Start()
	if err != nil {
		log.Error().Err(err).Msgf("failed to start iperf client with ip '%s'", clientIP)
	}

	<-iperfClient.Done

	iperfResult := IperfResult{
		UploadSpeed:   iperfClient.Report().End.SumSent.BitsPerSecond,
		DownloadSpeed: iperfClient.Report().End.SumReceived.BitsPerSecond,
		CpuReport:     iperfClient.Report().End.CpuReport,
		NodeIpv4:      clientIP,
		TestType:      string(iperfClient.Proto()),
		Error:         iperfClient.Report().Error,
	}

	if !tcp && len(iperfClient.Report().End.Streams) > 0 {
		iperfResult.DownloadSpeed = iperfClient.Report().End.Streams[0].Udp.BitsPerSecond
	}

	return iperfResult
}
