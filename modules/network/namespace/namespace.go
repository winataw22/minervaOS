package namespace

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/containernetworking/plugins/pkg/ns"

	"github.com/rs/zerolog/log"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	netNSPath = "/var/run/netns"
)

// Create creates a new named network namespace and bind mount
// its file descriptor to /var/run/netns/{name}
func Create(name string) (ns.NetNS, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if Exists(name) {
		nsPath := filepath.Join(netNSPath, name)
		return ns.GetNS(nsPath)
	}

	origin, err := netns.Get()
	if err != nil {
		return nil, err
	}
	defer func() {
		netns.Set(origin)
	}()

	// create a network namespace
	nsHandle, err := netns.New()
	if err != nil {
		return nil, err
	}
	defer nsHandle.Close()

	nsPath, err := mountBindNetNS(name)
	if err != nil {
		return nil, err
	}

	return ns.GetNS(nsPath)
}

// Delete deletes a network namespace
func Delete(name string) error {
	path := filepath.Join(netNSPath, name)
	ns, err := netns.GetFromPath(path)
	if err != nil {
		return err
	}

	if err := ns.Close(); err != nil {
		return err
	}

	if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	return nil
}

// Exists checks if a network namespace exists or not
func Exists(name string) bool {
	_, err := netns.GetFromName(name)
	return err == nil
}

// GetByName return a namespace by its name
func GetByName(name string) (ns.NetNS, error) {
	nsPath := filepath.Join(netNSPath, name)
	return ns.GetNS(nsPath)
}

func mountBindNetNS(name string) (string, error) {
	log.Info().Msg("create netnsPath")
	if err := os.MkdirAll(netNSPath, 0660); err != nil {
		return "", err
	}

	nsPath := filepath.Join(netNSPath, name)
	log.Info().Msg("create file")
	if err := touch(nsPath); err != nil {
		return "", err
	}

	src := "/proc/self/ns/net"
	log.Info().
		Str("src", src).
		Str("dest", nsPath).
		Msg("bind mount")

	if err := syscall.Mount(src, nsPath, "bind", syscall.MS_BIND, ""); err != nil {
		os.Remove(nsPath)
		return "", err
	}
	return nsPath, nil
}

func touch(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0660)
	if err != nil {
		return err
	}
	return f.Close()
}

// SetLink move a link to the named network namespace
func SetLink(link netlink.Link, name string) error {
	log.Info().Msg("get ns")
	ns, err := netns.GetFromName(name)
	if err != nil {
		return err
	}
	defer ns.Close()

	log.Info().Msg("linkSetNsFd")
	return netlink.LinkSetNsFd(link, int(ns))
}

// RouteAdd adds a route into a named network namespace
func RouteAdd(name string, route *netlink.Route) error {
	ns, err := netns.GetFromName(name)
	if err != nil {
		return err
	}
	defer ns.Close()

	h, err := netlink.NewHandleAt(ns)
	if err != nil {
		return err
	}
	return h.RouteAdd(route)
}
