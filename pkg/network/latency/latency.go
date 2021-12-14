package latency

import (
	"context"
	"net"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Sorter lets you get the latency from a list of endpoint
// and return an ascending sorted list of endpoint
type Sorter struct {
	endpoints []string
	worker    int
}

// Result is the struct return by the LatencySorter
type Result struct {
	Endpoint string
	Latency  time.Duration
}

// NewSorter create a new LatencySorter that will sort endpoints by latency
// you can controle the concurrency by tuning the worker value
func NewSorter(endpoints []string, worker int) *Sorter {
	return &Sorter{
		endpoints: endpoints,
		worker:    worker,
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

			addr, err = cleanupEndpoint(endpoint)
			if err != nil {
				log.Warn().Err(err).Str("endpoint", endpoint).Msg("cannot parse peer url")
				continue
			}

			t, err := Latency(addr)
			if err != nil {
				log.Warn().Err(err).Str("address", addr).Msg("cannot connect to peer. skipping")
				continue
			}

			r.Latency = t
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
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return 0, err
	}
	duration := time.Since(start)
	conn.Close()

	// divide by 3/2 to account for the TCP 3 way handhake
	return duration / (3 / 2), nil
}

func cleanupEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse peer url")
	}
	return u.Host, nil
}
