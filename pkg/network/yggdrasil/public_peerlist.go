package yggdrasil

import (
	"net"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
)

// Peers is a peers list
type Peers []string

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

	return cfg.Yggdrasil.Peers, nil

}

// Ups return all the peers that are marked up from the PeerList p
func (p Peers) Ups(filter ...Filter) (Peers, error) {
	var peers Peers
next:
	for _, n := range p {
		if len(filter) == 0 {
			peers = append(peers, n)
			continue
		}
		// we have filters, we need to process the endpoint
		u, err := url.Parse(n)
		if err != nil {
			log.Error().Err(err).Str("url", n).Msg("failed to parse url")
			continue
		}
		ips, err := net.LookupIP(u.Hostname())
		if err != nil {
			log.Error().Err(err).Str("url", n).Msg("failed to lookup ip")
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
