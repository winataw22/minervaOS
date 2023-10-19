package options

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/rs/zerolog/log"
)

type sysOption struct {
	key string
	val string
}

func (s *sysOption) apply(inf string) error {
	key := fmt.Sprintf(s.key, inf)
	log.Debug().Str("key", key).Str("value", s.val).Msg("sysctl")
	_, err := sysctl.Sysctl(key, s.val)
	return err
}

// IPv6Disable disabled Ipv6 on interface
func IPv6Disable(f bool) Option {
	return &sysOption{
		key: "net/ipv6/conf/%s/disable_ipv6",
		val: flag(f),
	}
}

// ProxyArp sets proxy arp on interface
func ProxyArp(f bool) Option {
	return &sysOption{
		key: "net/ipv4/conf/%s/proxy_arp",
		val: flag(f),
	}
}

// AcceptRA enables or disables forwarding for ipv6
func AcceptRA(f RouterAdvertisements) Option {
	return &sysOption{
		key: "net/ipv6/conf/%s/accept_ra",
		val: fmt.Sprintf("%d", f),
	}
}

// LearnDefaultRouteInRA Learn default router in Router Advertisement.
func LearnDefaultRouteInRA(f bool) Option {
	return &sysOption{
		key: "net/ipv6/conf/%s/accept_ra_defrtr",
		val: flag(f),
	}
}
