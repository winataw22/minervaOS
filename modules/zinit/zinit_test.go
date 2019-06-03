// Package zinit exposes function to interat with zinit service life cyle management
package zinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseList(t *testing.T) {
	s := `
ntp: Running
telnetd: Running
network-dhcp: Success
haveged: Success
debug-tty: Running
routing: Success
udevd: Running
dhcp_zos: Running
udev-trigger: Success
sshd-setup: Success
local-modprobe: Success
networkd: Success
sshd: Running`
	services, err := parseList(s)
	require.NoError(t, err)

	assert.Equal(t, map[string]ServiceState{
		"ntp":            ServiceStatusRunning,
		"telnetd":        ServiceStatusRunning,
		"network-dhcp":   ServiceStatusSuccess,
		"haveged":        ServiceStatusSuccess,
		"debug-tty":      ServiceStatusRunning,
		"routing":        ServiceStatusSuccess,
		"udevd":          ServiceStatusRunning,
		"dhcp_zos":       ServiceStatusRunning,
		"udev-trigger":   ServiceStatusSuccess,
		"sshd-setup":     ServiceStatusSuccess,
		"local-modprobe": ServiceStatusSuccess,
		"networkd":       ServiceStatusSuccess,
		"sshd":           ServiceStatusRunning,
	}, services)
}

func TestParseStatus(t *testing.T) {
	s := `
name: ntp
pid: 223
state: Running
target: Up
after:
  - network-dhcp: Success`
	status, err := parseStatus(s)
	require.NoError(t, err)

	assert.Equal(t, ServiceStatus{
		Name:   "ntp",
		Pid:    223,
		State:  ServiceStatusRunning,
		Target: ServiceTargetUp,
	}, status)
}
