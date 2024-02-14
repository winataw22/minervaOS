package networkhealth

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/perf"
)

const defaultRequestTimeout = 5 * time.Second

type NetworkHealthTask struct{}

type ServiceStatus struct {
	Url         string `json:"url"`
	IsReachable bool   `json:"is_reachable"`
}

var _ perf.Task = (*NetworkHealthTask)(nil)

func NewTask() *NetworkHealthTask {
	return &NetworkHealthTask{}
}

func (t *NetworkHealthTask) ID() string {
	return "network-health"
}

func (t *NetworkHealthTask) Description() string {
	return "Network health check runs periodically to check the connection to most of grid services."
}

func (t *NetworkHealthTask) Cron() string {
	return "0 */15 * * * *"
}

// Jitter returns the max number of seconds the job can sleep before actual execution.
func (t *NetworkHealthTask) Jitter() uint32 {
	return 5 * 60
}

func (t *NetworkHealthTask) Run(ctx context.Context) (interface{}, error) {
	env := environment.MustGet()
	servicesUrl := []string{
		env.ActivationURL, env.GraphQL, env.FlistURL,
	}
	servicesUrl = append(append(servicesUrl, env.SubstrateURL...), env.RelayURL...)

	reports := []ServiceStatus{}

	var wg sync.WaitGroup
	var mut sync.Mutex
	for _, serviceUrl := range servicesUrl {
		wg.Add(1)
		go func(serviceUrl string) {
			defer wg.Done()
			report := getNetworkReport(ctx, serviceUrl)

			mut.Lock()
			defer mut.Unlock()

			reports = append(reports, report)
		}(serviceUrl)
	}
	wg.Wait()

	return reports, nil
}

func getNetworkReport(ctx context.Context, serviceUrl string) ServiceStatus {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	report := ServiceStatus{
		Url:         serviceUrl,
		IsReachable: true,
	}

	address := parseUrl(serviceUrl)
	err := isReachable(ctx, address)
	if err != nil {
		report.IsReachable = false
	}

	return report
}

func parseUrl(serviceUrl string) string {
	u, err := url.Parse(serviceUrl)
	if err != nil {
		return ""
	}

	port := ":80"
	if u.Scheme == "https" || u.Scheme == "wss" {
		port = ":443"
	}

	if u.Port() == "" {
		u.Host += port
	}

	return u.Host
}

func isReachable(ctx context.Context, address string) error {
	d := net.Dialer{Timeout: defaultRequestTimeout}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	return nil
}
