package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	generated "github.com/threefoldtech/zos/pkg/gedis/types/provision"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func cmdsLive(c *cli.Context) error {
	var (
		userID  = c.Int64("id")
		start   = c.Int("start")
		end     = c.Int("end")
		expired = c.Bool("expired")
		deleted = c.Bool("deleted")
	)

	// keypair, err := identity.LoadKeyPair(seedPath)
	// if err != nil {
	// 	return errors.Wrapf(err, "could not find seed file at %s", seedPath)
	// }

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

const timeLayout = "02-Jan-2006 15:04:05"

func printResult(r generated.TfgridReservation1) {
	expire := r.DataReservation.ExpirationReservation
	fmt.Printf("ID:%6d expired at:%20s", r.ID, expire.Format(timeLayout))

	if len(r.Results) <= 0 {
		fmt.Printf("state: not deployed yet\n")
		return
	}

	resultPerID := make(map[int64]generated.TfgridReservationResult1, len(r.Results))
	for _, r := range r.Results {
		var (
			rid int64
			wid int64
		)
		fmt.Sscanf(r.WorkloadID, "%d-%d", &rid, &wid)
		resultPerID[wid] = r
	}

	for _, n := range r.DataReservation.Networks {
		fmt.Printf("\tnetwork ID: %s\n", n.Name)
	}
	for _, c := range r.DataReservation.Containers {
		result := resultPerID[c.WorkloadID]
		if result.State == generated.TfgridReservationResult1StateError {
			fmt.Printf("\terror: %s\n", result.Message)
			continue
		}

		data := provision.Container{}
		if err := json.Unmarshal(result.DataJSON, &data); err != nil {
			panic(err)
		}
		fmt.Printf("\tflist: %s", data.FList)
		for _, ip := range data.Network.IPs {
			fmt.Printf("\tIP: %s", ip)
		}
		fmt.Printf("\n")
	}
	for _, v := range r.DataReservation.Volumes {
		result := resultPerID[v.WorkloadID]
		if result.State == generated.TfgridReservationResult1StateError {
			fmt.Printf("\terror: %s\n", result.Message)
			continue
		}

		data := provision.VolumeResult{}
		if err := json.Unmarshal(result.DataJSON, &data); err != nil {
			panic(err)
		}
		fmt.Printf("\tVolume ID: %s Size: %d Type: %s\n", data.ID, v.Size, v.Type)
	}
	for _, z := range r.DataReservation.Zdbs {
		result := resultPerID[z.WorkloadID]
		if result.State == generated.TfgridReservationResult1StateError {
			fmt.Printf("\terror: %s\n", result.Message)
			continue
		}

		data := provision.ZDBResult{}
		if err := json.Unmarshal(result.DataJSON, &data); err != nil {
			panic(err)
		}
		fmt.Printf("\tAddr %s:%d Namespace %s\n", data.IP, data.Port, data.Namespace)
	}
	for _, k := range r.DataReservation.Kubernetes {
		result := resultPerID[k.WorkloadID]
		if result.State == generated.TfgridReservationResult1StateError {
			fmt.Printf("\terror: %s\n", result.Message)
			continue
		}

		data := provision.Kubernetes{}
		if err := json.Unmarshal(result.DataJSON, &data); err != nil {
			panic(err)
		}

		fmt.Printf("\tip: %v", data.IP)
		if data.MasterIPs == nil || len(data.MasterIPs) == 0 {
			fmt.Print(" master\n")
		} else {
			fmt.Printf("\n")
		}
	}
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

func (s *scraper) Scrap(user int64) chan generated.TfgridReservation1 {

	var (
		cIn  = make(chan job)
		cOut = make(chan generated.TfgridReservation1)
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

func worker(wg *sync.WaitGroup, cIn <-chan job, cOut chan<- generated.TfgridReservation1) {
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
		if !job.deleted && len(res.Results) > 0 && res.Results[0].State == generated.TfgridReservationResult1StateDeleted {
			continue
		}
		if res.CustomerTid != job.user {
			continue
		}
		cOut <- res
	}
}

func getResult(id int) (res generated.TfgridReservation1, err error) {
	url := fmt.Sprintf("https://explorer.devnet.grid.tf/reservations/%d", id)
	resp, err := http.Get(url)
	if err != nil {
		return res, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return res, os.ErrNotExist
	}

	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("wrong status code %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return res, err
	}

	return res, nil
}
