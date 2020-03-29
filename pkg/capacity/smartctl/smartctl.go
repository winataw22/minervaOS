package smartctl

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var reScan = regexp.MustCompile(`(?m)^([^\s]+)\s+-d\s+([^\s]+)\s+#`)
var reHeader = regexp.MustCompile(`(?m)([^\[]+)\[([^\[]+)\]`)
var reInfo = regexp.MustCompile(`(?m)([^:]+):\s+(.+)`)

// ErrEmpty is return when smatctl doesn't find any device
var ErrEmpty = errors.New("smartctl returned an empty response")

// Device represents a device as returned by "smartctl --scan"
type Device struct {
	Type string
	Path string
}

// ListDevices return a list of device present on the local system
func ListDevices() ([]Device, error) {
	cmd := exec.Command("smartctl", "--scan")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, ErrEmpty
	}

	return parseScan(output)
}

// Info contains information about a device as returned by "smartctl -i {path} -d {type}"
type Info struct {
	Tool        string
	Environment string
	Information map[string]string
}

// DeviceInfo info return the information from a specific device
func DeviceInfo(d Device) (Info, error) {
	cmd := exec.Command("smartctl", "-i", d.Path, "-d", d.Type)
	output, err := cmd.Output()
	if err != nil {
		return Info{}, err
	}

	return parseInfo(output)
}

func parseScan(b []byte) ([]Device, error) {
	trimed := strings.TrimSpace(string(b))
	lines := strings.Split(trimed, "\n")
	devices := make([]Device, len(lines))

	for i, line := range lines {
		match := reScan.FindStringSubmatch(line)
		if len(match) != 3 {
			return nil, fmt.Errorf("failed to parse ourput of 'smartctl --scan'")
		}
		devices[i] = Device{
			Type: match[2],
			Path: match[1],
		}
	}

	return devices, nil
}

func parseInfo(b []byte) (Info, error) {
	info := Info{}
	trimed := strings.TrimSpace(string(b))
	lines := strings.Split(trimed, "\n")
	if len(lines) == 0 {
		return info, fmt.Errorf("fail to parse smarctl device information")
	}

	header := reHeader.FindStringSubmatch(lines[0])
	info.Tool = strings.TrimSpace(header[1])
	info.Environment = strings.TrimSpace(header[2])
	info.Information = map[string]string{}

	for _, line := range lines[1:] {
		match := reInfo.FindStringSubmatch(line)
		if len(match) < 3 {
			continue
		}

		k := strings.TrimSpace(match[1])
		v := strings.TrimSpace(match[2])

		if k == "Local Time is" {
			// not needed and we don't want to have time based value
			continue
		}
		info.Information[k] = v
	}

	return info, nil
}
