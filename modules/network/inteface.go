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

func interfaces() ([]netlink.Link, error) {
	return netlink.LinkList()
}

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

func filterBridge(links []netlink.Link) []*netlink.Bridge {
	bridges := []*netlink.Bridge{}

	for _, link := range links {
		if link.Type() == "bridge" {
			bridge, ok := link.(*netlink.Bridge)
			if ok {
				bridges = append(bridges, bridge)
			}
		}
	}
	return bridges
}

func isPlugged(inf string) bool {
	data, err := ioutil.ReadFile(fmt.Sprintf(carrierFile, inf))
	if err != nil {
		return false
	}
	data = bytes.TrimSpace(data)
	if string(data) != "1" {
		return false
	}

	return true
}

func isVirtEth(inf string) bool {
	path := fmt.Sprintf("/sys/class/net/%s/device", inf)
	dest, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return strings.Contains(filepath.Base(dest), "virtio")
}
