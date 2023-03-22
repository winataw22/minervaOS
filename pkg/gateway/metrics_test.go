package gateway

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testBody = `
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2.198962176e+09
# HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE process_virtual_memory_max_bytes gauge
process_virtual_memory_max_bytes -1
# HELP traefik_config_last_reload_failure Last config reload failure
# TYPE traefik_config_last_reload_failure gauge
traefik_config_last_reload_failure 0
# HELP traefik_config_last_reload_success Last config reload success
# TYPE traefik_config_last_reload_success gauge
traefik_config_last_reload_success 1.631101061e+09
# HELP traefik_config_reloads_failure_total Config failure reloads
# TYPE traefik_config_reloads_failure_total counter
traefik_config_reloads_failure_total 0
# HELP traefik_config_reloads_total Config reloads
# TYPE traefik_config_reloads_total counter
traefik_config_reloads_total 1
# HELP traefik_entrypoint_open_connections How many open connections exist on an entrypoint, partitioned by method and protocol.
# TYPE traefik_entrypoint_open_connections gauge
traefik_entrypoint_open_connections{entrypoint="metrics",method="GET",protocol="http"} 1
traefik_entrypoint_open_connections{entrypoint="metrics",method="POST",protocol="http"} 0
# HELP traefik_entrypoint_request_duration_seconds How long it took to process the request on an entrypoint, partitioned by status code, protocol, and method.
# TYPE traefik_entrypoint_request_duration_seconds histogram
traefik_entrypoint_request_duration_seconds_bucket{code="502",entrypoint="metrics",method="POST",protocol="http",le="0.1"} -1
traefik_entrypoint_request_duration_seconds_bucket{code="502",entrypoint="metrics",method="POST",protocol="http",le="0.3"} 1
traefik_entrypoint_request_duration_seconds_bucket{code="502",entrypoint="metrics",method="POST",protocol="http",le="1.2"} 1
traefik_entrypoint_request_duration_seconds_bucket{code="502",entrypoint="metrics",method="POST",protocol="http",le="5"} 1
traefik_entrypoint_request_duration_seconds_bucket{code="502",entrypoint="metrics",method="POST",protocol="http",le="+Inf"} 1
traefik_entrypoint_request_duration_seconds_sum{code="502",entrypoint="metrics",method="POST",protocol="http"} 0.00067077
traefik_entrypoint_request_duration_seconds_count{code="502",entrypoint="metrics",method="POST",protocol="http"} 1
# HELP traefik_entrypoint_requests_total How many HTTP requests processed on an entrypoint, partitioned by status code, protocol, and method.
# TYPE traefik_entrypoint_requests_total counter
traefik_entrypoint_requests_total{code="502",entrypoint="metrics",method="POST",protocol="http"} 1
# HELP traefik_service_open_connections How many open connections exist on a service, partitioned by method and protocol.
# TYPE traefik_service_open_connections gauge
traefik_service_open_connections{method="POST",protocol="http",service="foo@file"} 0
# HELP traefik_service_request_duration_seconds How long it took to process the request on a service, partitioned by status code, protocol, and method.
# TYPE traefik_service_request_duration_seconds histogram
traefik_service_request_duration_seconds_bucket{code="502",method="POST",protocol="http",service="foo@file",le="0.1"} 1
traefik_service_request_duration_seconds_bucket{code="502",method="POST",protocol="http",service="foo@file",le="0.3"} 1
traefik_service_request_duration_seconds_bucket{code="502",method="POST",protocol="http",service="foo@file",le="1.2"} 1
traefik_service_request_duration_seconds_bucket{code="502",method="POST",protocol="http",service="foo@file",le="5"} 1
traefik_service_request_duration_seconds_bucket{code="502",method="POST",protocol="http",service="foo@file",le="+Inf"} 1
traefik_service_request_duration_seconds_sum{code="502",method="POST",protocol="http",service="foo@file"} 0.000484654
traefik_service_request_duration_seconds_count{code="502",method="POST",protocol="http",service="foo@file"} 1
# HELP traefik_service_requests_total How many HTTP requests processed on a service, partitioned by status code, protocol, and method.
# TYPE traefik_service_requests_total counter
traefik_service_requests_total{code="502",method="POST",protocol="http",service="foo@file"} 1`

	bigNumber = `
traefik_service_bytes_sent_total{service="50-715-gw@file"} 8.95470206e+08
`
	testValues = `
# TYPE traefik_service_requests_bytes_total counter
traefik_service_requests_bytes_total{code="200",method="GET",protocol="http",service="10-123-gateway@file"} 100
traefik_service_requests_bytes_total{code="404",method="GET",protocol="http",service="10-123-gateway@file"} 120
traefik_service_requests_bytes_total{code="502",method="GET",protocol="http",service="10-123-gateway@file"} 10
traefik_service_responses_bytes_total{code="200",method="GET",protocol="http",service="10-123-gateway@file"} 1307
traefik_service_responses_bytes_total{code="404",method="GET",protocol="http",service="10-123-gateway@file"} 469
traefik_service_responses_bytes_total{code="502",method="GET",protocol="http",service="10-123-gateway@file"} 22
	`
)

func TestParseMetrics(t *testing.T) {

	buf := bytes.NewBuffer([]byte(testBody))

	results, err := parseMetrics(buf)
	require.NoError(t, err)

	// single value, no tags
	m, ok := results["traefik_config_reloads_total"]
	require.True(t, ok)

	require.Equal(t, "traefik_config_reloads_total", m.key)
	require.Len(t, m.values, 1)
	require.EqualValues(t, m.values[0].value, 1)

	// multiple values

	m, ok = results["traefik_entrypoint_request_duration_seconds_bucket"]
	require.True(t, ok)

	require.Equal(t, "traefik_entrypoint_request_duration_seconds_bucket", m.key)
	require.Len(t, m.values, 5)
	fst := m.values[0]

	require.EqualValues(t, fst.value, -1)
	require.Len(t, fst.labels, 5)

	fmt.Println(fst.labels)
	require.Equal(t, "502", fst.labels["code"])
	require.Equal(t, "0.1", fst.labels["le"])
}

func TestParseBigMetrics(t *testing.T) {

	buf := bytes.NewBuffer([]byte(bigNumber))

	results, err := parseMetrics(buf)
	require.NoError(t, err)

	// single value, no tags
	m, ok := results["traefik_service_bytes_sent_total"]
	require.True(t, ok)

	require.Equal(t, "traefik_service_bytes_sent_total", m.key)
	require.Len(t, m.values, 1)
	require.EqualValues(t, float64(8.95470206e+08), m.values[0].value)
}

func TestMetrics(t *testing.T) {

	buf := bytes.NewBuffer([]byte(testValues))

	results, err := parseMetrics(buf)
	require.NoError(t, err)

	request := results[metricRequest]
	responses := results[metricResponse]

	servicesRequests := request.group("service", func(s string) string {
		return strings.TrimSuffix(s, "@file")
	})

	servicesResponses := responses.group("service", func(s string) string {
		return strings.TrimSuffix(s, "@file")
	})

	require.Len(t, servicesRequests, 1)
	require.Len(t, servicesResponses, 1)

	require.EqualValues(t, 230, servicesRequests["10-123-gateway"])
	require.EqualValues(t, 1798, servicesResponses["10-123-gateway"])
}
