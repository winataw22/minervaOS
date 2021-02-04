/*
Package options abstract setting common networking sys flags on the selected namespaces
*/
package options

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
)

// SetIPv6Forwarding enables or disables forwarding for ipv6
func SetIPv6Forwarding(f bool) error {
	_, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", flag(f))
	return err
}

//RouterAdvertisements flag
type RouterAdvertisements int

const (
	//RAOff Do not accept Router Advertisements
	RAOff RouterAdvertisements = iota
	//RAAcceptIfForwardingIsDisabled Accept Router Advertisements if forwarding is disabled
	RAAcceptIfForwardingIsDisabled
	//RAAcceptIfForwardingIsEnabled Overrule forwarding behaviour. Accept Router Advertisements even if forwarding is enabled.
	RAAcceptIfForwardingIsEnabled
)

// SetIPv6AcceptRA enables or disables forwarding for ipv6
func SetIPv6AcceptRA(f RouterAdvertisements) error {
	_, err := sysctl.Sysctl("net.ipv6.conf.all.accept_ra", fmt.Sprint(int(f)))
	return err
}

// SetIPv6LearnDefaultRouteInRA Learn default router in Router Advertisement.
func SetIPv6LearnDefaultRouteInRA(f bool) error {
	_, err := sysctl.Sysctl("net.ipv6.conf.all.accept_ra_defrtr", flag(f))
	return err
}
