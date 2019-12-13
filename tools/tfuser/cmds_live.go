package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func cmdsLive(c *cli.Context) error {
	var (
		seedPath = c.String("seed")
		start    = c.Int("start")
		end      = c.Int("end")
	)

	keypair, err := identity.LoadKeyPair(seedPath)
	if err != nil {
		return errors.Wrapf(err, "could not find seed file at %s", seedPath)
	}

	s := scraper{
		poolSize: 10,
		start:    start,
		end:      end,
	}

	cResults := s.Scrap(keypair.Identity())
	for result := range cResults {
		printResult(result)
	}
	return nil
}

const timeLayout = "02-Jan-2006 15:04:05"

func printResult(r res) {
	expire := r.Created.Add(r.Duration)
	fmt.Printf("ID:%6s Type:%10s expired at:%20s", r.ID, r.Type, expire.Format(timeLayout))
	if r.Result == nil {
		fmt.Printf("state: not deployed yet\n")
		return
	}
	fmt.Printf("state: %6s", r.Result.State)
	if r.Result.State == "error" {
		fmt.Printf("\t%s\n", r.Result.Error)
		return
	}

	switch r.Type {
	case provision.VolumeReservation:
		rData := provision.VolumeResult{}
		data := provision.Volume{}
		if err := json.Unmarshal(r.Data, &data); err != nil {
			panic(err)
		}
		if err := json.Unmarshal(r.Result.Data, &rData); err != nil {
			panic(err)
		}
		fmt.Printf("\tVolume ID: %s Size: %d Type: %s\n", rData.ID, data.Size, data.Type)
	case provision.ZDBReservation:
		data := provision.ZDBResult{}
		if err := json.Unmarshal(r.Result.Data, &data); err != nil {
			panic(err)
		}
		fmt.Printf("\tAddr %s:%d Namespace %s\n", data.IP, data.Port, data.Namespace)

	case provision.ContainerReservation:
		data := provision.Container{}
		if err := json.Unmarshal(r.Data, &data); err != nil {
			panic(err)
		}
		fmt.Printf("\tflist: %s", data.FList)
		for _, ip := range data.Network.IPs {
			fmt.Printf("\tIP: %s", ip)
		}
		fmt.Printf("\n")
	case provision.NetworkReservation:
		data := pkg.Network{}
		if err := json.Unmarshal(r.Data, &data); err != nil {
			panic(err)
		}

		fmt.Printf("\tnetwork ID: %s\n", data.NetID)
	}
}

type scraper struct {
	poolSize int
	start    int
	end      int
	wg       sync.WaitGroup
}
type job struct {
	id   int
	user string
}
type res struct {
	provision.Reservation
	Result *provision.Result `json:"result"`
}

func (s *scraper) Scrap(user string) chan res {

	var (
		cIn  = make(chan job)
		cOut = make(chan res)
	)

	s.wg.Add(s.poolSize)
	for i := 0; i < s.poolSize; i++ {
		go worker(&s.wg, cIn, cOut)
	}

	go func() {
		defer func() {
			close(cIn)
		}()
		for i := s.start; i < 500; i++ {
			cIn <- job{
				id:   i,
				user: user,
			}
		}
	}()

	go func() {
		s.wg.Wait()
		close(cOut)
	}()

	return cOut
}

func worker(wg *sync.WaitGroup, cIn <-chan job, cOut chan<- res) {
	defer func() {
		wg.Done()
	}()

	for job := range cIn {
		res, err := getResult(job.id)
		if err != nil {
			continue
		}
		if res.Expired() == true {
			continue
		}
		if res.User != job.user {
			continue
		}
		cOut <- res
	}
}

func getResult(id int) (res, error) {
	url := fmt.Sprintf("https://explorer.devnet.grid.tf/reservations/%d-1", id)
	resp, err := http.Get(url)
	if err != nil {
		return res{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return res{}, os.ErrNotExist
	}

	if resp.StatusCode != http.StatusOK {
		return res{}, fmt.Errorf("wrong status code %s", resp.Status)
	}

	b := res{}
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return res{}, err
	}

	return b, nil
}
