// Package zinit exposes function to interat with zinit service life cyle management
package zinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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
log: Ring
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

func TestParseService(t *testing.T) {
	b := []byte(`
exec: /bin/true
test: test -e /bin/true
oneshot: false
log: ring
after:
 - one
 - two	
`)
	var s InitService
	err := yaml.Unmarshal(b, &s)
	require.NoError(t, err)

	assert.Equal(t, InitService{
		Exec:    "/bin/true",
		Test:    "test -e /bin/true",
		Oneshot: false,
		Log:     RingLogType,
		After:   []string{"one", "two"},
	}, s)
}
