package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/vishvananda/netlink"
)

const carrierFile = "/sys/class/net/%s/carrier"

func filterDevices(links []netlink.Link) []*netlink.Device {
	devices := []*netlink.Device{}

	for _, link := range links {
		if link.Type() == "device" {
			device, ok := link.(*netlink.Device)
			if ok {
				devices = append(devices, device)
			}
		}
	}
	return devices
}

func isPlugged(inf string) bool {
	data, err := ioutil.ReadFile(fmt.Sprintf(carrierFile, inf))
	if err != nil {
		return false
	}
	data = bytes.TrimSpace(data)
	return string(data) != "1"
}

func isVirtEth(inf string) bool {
	path := fmt.Sprintf("/sys/class/net/%s/device", inf)
	dest, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return strings.Contains(filepath.Base(dest), "virtio")
}
