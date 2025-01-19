package gateway

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/namespace"
)

const (
	publicNS   = "public"
	metricsURL = "http://127.0.0.1:8082/metrics"

	metricRequest  = "traefik_service_requests_bytes_total"
	metricResponse = "traefik_service_responses_bytes_total"
)

var (
	ErrMetricsNotAvailable = errors.New("metrics not available")

	metricM = regexp.MustCompile(`^(\w+)({[^}]+})? ([0-9\.e\+-]+)`)
	tagsM   = regexp.MustCompile(`([^,={]+)="([^"]+)"`)
)

type value struct {
	value  float64
	labels map[string]string
}

type metric struct {
	key    string
	values []value
}

// group group values by label, if multiple values in this
// metric has the same value for the label, values are added.
// applies the mapping function on the value of the label
func (m *metric) group(l string, mapping func(string) string) map[string]float64 {
	result := make(map[string]float64)
	if mapping == nil {
		mapping = func(s string) string { return s }
	}
	for _, v := range m.values {
		result[mapping(v.labels[l])] += v.value
	}
	return result
}

func tags(s string) map[string]string {
	matches := tagsM.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, m := range matches {
		result[m[1]] = m[2]
	}
	return result
}

func parseMetrics(in io.Reader) (map[string]*metric, error) {
	scanner := bufio.NewScanner(in)
	results := make(map[string]*metric)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := metricM.FindStringSubmatch(line)

		if len(parts) == 0 {
			// no match
			continue
		}
		key := parts[1]
		labels := tags(parts[2])
		valueStr := parts[3]

		valueF, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse value '%s' for metric '%s'", valueStr, key)
		}

		v := value{
			value:  valueF,
			labels: labels,
		}

		m, ok := results[key]
		if !ok {
			m = &metric{key: key}
			results[key] = m
		}

		m.values = append(m.values, v)
	}

	return results, nil
}

func metrics(rawUrl string) (map[string]*metric, error) {
	// so. why this is done like this you may ask?
	// and we don't have a straight answer to you. but here is the deal
	// traefik is running (always) inside the public namespace
	// this is why we call `metrics` method from inside that namespace
	// we expected this to work since the go routine now is locked to
	// the OS thread that is running inside this namespace.
	// seems that is wrong, using the http client (default or custom)
	// will always use the host namespace, so we assume there is a go routine
	// spawned in the client somewhere which is not fixed to the same os thread.
	//
	// we tried to disable keep-alive so we create new connection always.
	// and other tricks as well. but nothing worked.
	//
	// the only way was to create a tcp connection ourselves and then
	// use this int he http client.
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}

	con, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, errors.Wrap(ErrMetricsNotAvailable, err.Error())
	}

	defer con.Close()

	cl := retryablehttp.NewClient()
	cl.HTTPClient.Transport = &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return con, nil
		},
	}
	cl.RetryMax = 5

	response, err := cl.Get(rawUrl)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics")
	}

	defer func() {
		_, _ = io.ReadAll(response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode != 200 {
		return nil, errors.Wrapf(err, "got wrong status code for metrics: %s", response.Status)
	}

	return parseMetrics(response.Body)
}

func (g *gatewayModule) Metrics() (result pkg.GatewayMetrics, err error) {
	// metric is only available if traefik is running. we can instead of doing
	// all the checks, we can try to directly get the metrics and see if we
	// can get it. we need to do these operations anyway.
	pubNS, err := namespace.GetByName(publicNS)
	if err != nil {
		// gateway is not enabled
		return result, nil
	}

	defer pubNS.Close()
	var values map[string]*metric
	err = pubNS.Do(func(_ ns.NetNS) error {
		log.Debug().Str("namespace", publicNS).Str("url", metricsURL).Msg("requesting metrics from traefik")
		values, err = metrics(metricsURL)

		return err
	})

	if errors.Is(err, ErrMetricsNotAvailable) {
		// traefik is not running because there
		// are no gateway configured
		return result, nil
	} else if err != nil {
		return result, err
	}

	mapping := func(s string) string {
		return strings.TrimSuffix(s, "@file")
	}
	if m, ok := values[metricRequest]; ok {
		// sent metrics.
		result.Request = m.group("service", mapping)
	}

	if m, ok := values[metricResponse]; ok {
		result.Response = m.group("service", mapping)
	}

	return
}
