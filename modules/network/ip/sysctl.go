package ip

import "io/ioutil"

// EnableIPv6Forwarding enable ipv6 forwarding
func EnableIPv6Forwarding() error {
	return ioutil.WriteFile("/proc/sys/net/ipv6/conf/all/forwarding", []byte("1"), 0644)
}
