package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
	"gopkg.in/yaml.v2"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/urfave/cli"
)

func cmdsLive(c *cli.Context) error {
	var (
		userID  = int64(mainui.ThreebotID)
		start   = c.Int("start")
		end     = c.Int("end")
		expired = c.Bool("expired")
		deleted = c.Bool("deleted")
	)

	s := scraper{
		poolSize: 10,
		start:    start,
		end:      end,
		expired:  expired,
		deleted:  deleted,
	}
	cResults := s.Scrap(userID)
	for result := range cResults {
		printResult(result)
	}
	return nil
}

type m map[string]interface{}

const timeLayout = "02-Jan-2006 15:04:05"

func printResult(r workloads.Reservation) {
	expire := r.DataReservation.ExpirationReservation
	output := m{}
	fmt.Printf("ID: %d Expires: %s\n", r.ID, expire.Format(timeLayout))

	resultPerID := make(map[int64][]workloads.Result, len(r.Results))
	for _, r := range r.Results {
		var (
			rid int64
			wid int64
		)
		fmt.Sscanf(r.WorkloadId, "%d-%d", &rid, &wid)
		results := resultPerID[wid]
		results = append(results, r)

		resultPerID[wid] = results
	}

	resultData := func(in json.RawMessage) m {
		var o m
		if err := json.Unmarshal(in, &o); err != nil {
			panic("invalid json")
		}
		return o
	}

	allResults := func(in []workloads.Result) []m {
		var o []m
		for _, i := range in {
			r := resultData(i.DataJson)
			r["node"] = i.NodeId
			r["state"] = i.State.String()
			o = append(o, r)
		}
		return o
	}

	for _, n := range r.DataReservation.Networks {
		d := m{
			"kind":    "network",
			"name":    n.Name,
			"results": allResults(resultPerID[n.WorkloadId]),
		}

		output[fmt.Sprintf("Workload %d", n.WorkloadId)] = d
	}

	for _, c := range r.DataReservation.Containers {
		var ips []string
		for _, ip := range c.NetworkConnection {
			ips = append(ips, ip.Ipaddress.String())
		}

		d := m{
			"kind":    "container",
			"flist":   c.Flist,
			"ip":      ips,
			"results": allResults(resultPerID[c.WorkloadId]),
		}

		output[fmt.Sprintf("Workload %d", c.WorkloadId)] = d
	}

	for _, v := range r.DataReservation.Volumes {
		d := m{
			"kind":    "volume",
			"size":    v.Size,
			"type":    v.Type.String(),
			"results": allResults(resultPerID[v.WorkloadId]),
		}

		output[fmt.Sprintf("Workload %d", v.WorkloadId)] = d

	}

	for _, z := range r.DataReservation.Zdbs {
		d := m{
			"kind":    "zdb",
			"size":    z.Size,
			"mode":    z.Mode.String(),
			"results": allResults(resultPerID[z.WorkloadId]),
		}

		output[fmt.Sprintf("Workload %d", z.WorkloadId)] = d
	}

	for _, k := range r.DataReservation.Kubernetes {
		d := m{
			"kind":    "kubernetes",
			"size":    k.Size,
			"network": k.NetworkId,
			"ip":      k.MasterIps,
			"results": allResults(resultPerID[k.WorkloadId]),
		}

		output[fmt.Sprintf("Workload %d", k.WorkloadId)] = d
	}

	if err := yaml.NewEncoder(os.Stdout).Encode(output); err != nil {
		log.Error().Err(err).Msg("failed to print result")
	}

	fmt.Println("-------------------------")
}

type scraper struct {
	poolSize int
	start    int
	end      int
	expired  bool
	deleted  bool
	wg       sync.WaitGroup
}
type job struct {
	id      int
	user    int64
	expired bool
	deleted bool
}

func (s *scraper) Scrap(user int64) chan workloads.Reservation {

	var (
		cIn  = make(chan job)
		cOut = make(chan workloads.Reservation)
	)

	s.wg.Add(s.poolSize)
	for i := 0; i < s.poolSize; i++ {
		go worker(&s.wg, cIn, cOut)
	}

	go func() {
		defer func() {
			close(cIn)
		}()
		for i := s.start; i < s.end; i++ {
			cIn <- job{
				id:      i,
				user:    user,
				expired: s.expired,
			}
		}
	}()

	go func() {
		s.wg.Wait()
		close(cOut)
	}()

	return cOut
}

func worker(wg *sync.WaitGroup, cIn <-chan job, cOut chan<- workloads.Reservation) {
	defer func() {
		wg.Done()
	}()

	now := time.Now()

	for job := range cIn {
		res, err := getResult(job.id)
		if err != nil {
			continue
		}

		expired := now.After(res.DataReservation.ExpirationReservation.Time)

		if !job.expired && expired {
			continue
		}
		// FIXME: beurk
		if !job.deleted && len(res.Results) > 0 && res.Results[0].State == workloads.ResultStateDeleted {
			continue
		}
		if res.CustomerTid != job.user {
			continue
		}
		cOut <- res
	}
}

func getResult(id int) (res workloads.Reservation, err error) {
	return bcdb.Workloads.Get(schema.ID(id))
}
