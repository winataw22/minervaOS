package yggdrasil

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
)

//PeerListFallback is an hardcoded list of public yggdrasil node
// it is used to have some available peer to connect to when we failed to read the online public peer info
var PeerListFallback = Peers{
	{
		Endpoint: "tls://45.147.198.155:6010",
		Up:       true,
	},
	{
		Endpoint: "tcp://51.15.204.214:12345",
		Up:       true,
	},
	{
		Endpoint: "tls://51.255.223.60:54232",
		Up:       true,
	},
}

// NodeInfo is the know information about an yggdrasil public node
type NodeInfo struct {
	Endpoint   string `json:"-"`
	Up         bool   `json:"up"`
	ProtoMinor int    `json:"proto_minor"`
}

// Peers is a peers list
type Peers []NodeInfo

type Filter func(ip net.IP) bool

// Ranges is a list of net.IPNet
type Ranges []net.IPNet

// Exclude ranges, return IPs that are NOT in the given ranges
func Exclude(ranges Ranges) Filter {
	return func(ip net.IP) bool {
		for _, n := range ranges {
			if n.Contains(ip) {
				return false
			}
		}
		return true
	}
}

// Include ranges, return IPs that are IN one of the given ranges
func Include(ranges Ranges) Filter {
	return func(ip net.IP) bool {
		for _, n := range ranges {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}
}

// IPV4Only is an IPFilter function that filters out non IPv4 address
func IPV4Only(ip net.IP) bool {
	return ip.To4() != nil
}

func fetchZosYggList() (Peers, error) {
	cfg, err := environment.GetConfig()
	if err != nil {
		return nil, err
	}

	var peers Peers
	for _, peer := range cfg.Yggdrasil.Peers {
		peers = append(peers, NodeInfo{
			Endpoint:   peer,
			Up:         true,
			ProtoMinor: 4,
		})
	}

	return peers, nil
}

func fetchPubYggList() Peers {
	// Try to fetch public peer
	// If we failed to do so, use the fallback hardcoded peer list
	var pl Peers

	// Do not retry more than 4 times
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 4)

	fetchPeerList := func() error {
		p, err := FetchPubYggPeerList()
		if err != nil {
			log.Debug().Err(err).Msg("failed to fetch yggdrasil peers")
			return err
		}
		pl = p
		return nil
	}

	err := backoff.Retry(fetchPeerList, bo)
	if err != nil {
		log.Error().Err(err).Msg("failed to read yggdrasil public peer list online, using fallback")
		pl = PeerListFallback
	}

	return pl
}

// FetchPubYggPeerList download the list of public yggdrasil peer from https://publicpeers.neilalexander.dev/publicnodes.json
func FetchPubYggPeerList() (Peers, error) {
	//pl := PeerList{}
	pl := map[string]map[string]NodeInfo{}

	resp, err := http.Get("https://publicpeers.neilalexander.dev/publicnodes.json")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&pl); err != nil {
		return nil, err
	}

	var peers Peers
	for _, nodes := range pl {
		for endpoint, node := range nodes {
			if node.ProtoMinor != 4 {
				continue
			}

			node.Endpoint = endpoint
			peers = append(peers, node)
		}
	}

	return peers, nil
}

// Ups return all the peers that are marked up from the PeerList p
func (p Peers) Ups(filter ...Filter) (Peers, error) {
	var peers Peers
next:
	for _, n := range p {
		if !n.Up {
			continue
		}
		if len(filter) == 0 {
			peers = append(peers, n)
			continue
		}
		// we have filters, we need to process the endpoint
		u, err := url.Parse(n.Endpoint)
		if err != nil {
			log.Error().Err(err).Str("url", n.Endpoint).Msg("failed to parse url")
			continue
		}
		ips, err := net.LookupIP(u.Hostname())
		if err != nil {
			log.Error().Err(err).Str("url", n.Endpoint).Msg("failed to lookup ip")
			continue
		}

		for _, ip := range ips {
			for _, f := range filter {
				if !f(ip) {
					continue next
				}
			}
		}

		peers = append(peers, n)
	}

	return peers, nil
}
