package iperf

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/iperf"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/perf/graphql"
)

// IperfTest for iperf tcp/udp tests
type IperfTest struct {
	taskID      string
	schedule    string
	description string
}

// IperfResult for iperf test results
type IperfResult struct {
	UploadSpeed   float64               `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64               `json:"download_speed"` // in bit/sec
	NodeID        uint32                `json:"node_id"`
	NodeIpv4      string                `json:"node_ip"`
	TestType      string                `json:"test_type"`
	Error         string                `json:"error"`
	CpuReport     CPUUtilizationPercent `json:"cpu_report"`
}

// NewTask creates a new iperf test
func NewTask() perf.Task {
	// because go-iperf left tmp directories with perf binary in it each time
	// the task had run
	matches, _ := filepath.Glob("/tmp/goiperf*")
	for _, match := range matches {
		os.RemoveAll(match)
	}
	return &IperfTest{
		taskID:      "iperf",
		schedule:    "0 0 */6 * * *",
		description: "Test public nodes network performance with both UDP and TCP over IPv4 and IPv6",
	}
}

// ID returns the ID of the tcp task
func (t *IperfTest) ID() string {
	return t.taskID
}

// Cron returns the schedule for the tcp task
func (t *IperfTest) Cron() string {
	return t.schedule
}

// Description returns the task description
func (t *IperfTest) Description() string {
	return t.description
}

// Run runs the tcp test and returns the result
func (t *IperfTest) Run(ctx context.Context) (interface{}, error) {
	env := environment.MustGet()
	g := graphql.NewGraphQl(env.GraphQL)

	// get public up nodes
	freeFarmNodes, err := g.GetUpNodes(ctx, 0, 1, 0, true, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list freefarm nodes from graphql")
	}

	nodes, err := g.GetUpNodes(ctx, 12, 0, 1, true, true)
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
	opts := make([]string, 0)
	opts = append(opts,
		"--client", clientIP,
		"--bandwidth", "1M",
		"--port", fmt.Sprint(iperf.IperfPort),
		"--interval", "20",
		"--json",
	)

	if !tcp {
		opts = append(opts, "--length", "16B", "--udp")
	}
	output, err := exec.CommandContext(ctx, "iperf", opts...).CombinedOutput()
	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		log.Err(err).Msg("failed to run iperf")
		return IperfResult{}
	}

	var report iperfCommandOutput
	if err := json.Unmarshal(output, &report); err != nil {
		log.Err(err).Msg("failed to parse iperf output")
		return IperfResult{}
	}

	proto := "tcp"
	if !tcp {
		proto = "udp"
	}
	iperfResult := IperfResult{
		UploadSpeed:   report.End.SumSent.BitsPerSecond,
		DownloadSpeed: report.End.SumReceived.BitsPerSecond,
		CpuReport:     report.End.CPUUtilizationPercent,
		NodeIpv4:      clientIP,
		TestType:      proto,
		Error:         report.Error,
	}

	if !tcp && len(report.End.Streams) > 0 {
		iperfResult.DownloadSpeed = report.End.Streams[0].UDP.BitsPerSecond
	}

	return iperfResult
}
