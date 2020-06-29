package latency

import (
	"bytes"
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// Sorter lets you get the latency from a list of endpoint
// and return an ascending sorted list of endpoint
type Sorter struct {
	endpoints []string
	worker    int
	filters   []IPFilter
}

// Result is the struct return by the LatencySorter
type Result struct {
	Endpoint string
	Latency  time.Duration
}

// IPFilter is function used by Sorted to filters out IP address during the latency test
type IPFilter func(net.IP) bool

// IPV4Only is an IPFilter function that filters out non IPv4 address
func IPV4Only(ip net.IP) bool {
	return ip.To4() != nil
}

// ExcludePrefix is a IPFilter function that filters IPs that start with prefix
func ExcludePrefix(prefix []byte) IPFilter {
	return func(ip net.IP) bool {
		return !bytes.HasPrefix(ip, prefix)
	}
}

// NewSorter create a new LatencySorter that will sort endpoints by latency
// you can controle the concurrency by tuning the worker value
func NewSorter(endpoints []string, worker int, filters ...IPFilter) *Sorter {
	return &Sorter{
		endpoints: endpoints,
		worker:    worker,
		filters:   filters,
	}
}

// Run concurrently checks the latency of all the endpoint contained in l
func (l *Sorter) Run(ctx context.Context) []Result {

	worker := func(in <-chan string, out chan<- Result) {
		for endpoint := range in {
			var (
				r    = Result{Endpoint: endpoint}
				addr string
				err  error
			)

			addr = cleanupEndpoint(endpoint)
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				continue
			}

			skip := false
			for _, filter := range l.filters {
				ip := net.ParseIP(host)
				skip = !filter(ip)
				if skip {
					break
				}
			}
			if skip {
				continue
			}

			t, err := Latency(addr)
			if err == nil {
				r.Latency = t
			}
			out <- r
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(l.worker)
	in := make(chan string)
	out := make(chan Result)

	go func() {
		wg.Wait()
		close(out)
	}()

	for i := 0; i < l.worker; i++ {
		go func() {
			defer wg.Done()
			worker(in, out)
		}()
	}

	go func() {
		for _, endpoint := range l.endpoints {
			in <- endpoint
		}
		close(in)
	}()

	results := make([]Result, 0, len(l.endpoints))
	for result := range out {
		if result.Latency == 0 {
			continue
		}
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Latency < results[j].Latency
	})

	return results
}

// Latency does a TCP dial to host and return the amount of time it took to get a response or an error if it fails to connect
func Latency(host string) (time.Duration, error) {
	start := time.Now()
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return 0, err
	}
	duration := time.Since(start)
	conn.Close()

	// divide by 3/2 to account for the TCP 3 way handhake
	return duration / (3 / 2), nil
}

func cleanupEndpoint(endpoint string) string {
	if strings.HasPrefix(endpoint, "tcp://") || strings.HasPrefix(endpoint, "tls://") {
		return endpoint[6:]
	}
	return endpoint
}
