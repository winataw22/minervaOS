package namespace

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	netNSPath = "/var/run/netns"
)

func CreateNetNS(name string) (netns.NsHandle, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origin, err := netns.Get()
	if err != nil {
		return 0, err
	}
	defer func() {
		netns.Set(origin)
	}()

	// create a network namespace
	ns, err := netns.New()
	if err != nil {
		return 0, err
	}
	defer ns.Close()

	// // set its ID
	// // In an attempt to avoid namespace id collisions, set this to something
	// // insanely high. When the kernel assigns IDs, it does so starting from 0
	// // So, just use our pid shifted up 16 bits
	// wantID := os.Getpid() << 16

	// h, err := netlink.NewHandle()
	// if err != nil {
	// 	return 0, err
	// }

	// err = h.SetNetNsIdByFd(int(ns), wantID)
	// if err != nil {
	// 	return 0, err
	// }

	if err := mountBindNetNS(name); err != nil {
		return 0, err
	}

	return ns, nil
}

func DeleteNetNS(name string) error {
	path := filepath.Join(netNSPath, name)
	ns, err := netns.GetFromPath(path)
	if err != nil {
		return err
	}

	if err := ns.Close(); err != nil {
		return err
	}

	if err := syscall.Unmount(path, syscall.MNT_FORCE); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	return nil
}

func mountBindNetNS(name string) error {
	log.Info().Msg("create netnsPath")
	if err := os.MkdirAll(netNSPath, 0660); err != nil {
		return err
	}

	nsPath := filepath.Join(netNSPath, name)
	log.Info().Msg("create file")
	if err := touch(nsPath); err != nil {
		return err
	}

	src := "/proc/self/ns/net"
	log.Info().
		Str("src", src).
		Str("dest", nsPath).
		Msg("bind mount")

	if err := syscall.Mount(src, nsPath, "bind", syscall.MS_BIND, ""); err != nil {
		os.Remove(nsPath)
		return err
	}
	return nil
}

func touch(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0660)
	if err != nil {
		return err
	}
	return f.Close()
}

func SetLinkNS(link netlink.Link, name string) error {
	log.Info().Msg("get ns")
	ns, err := netns.GetFromName(name)
	if err != nil {
		return err
	}
	defer ns.Close()

	log.Info().Msg("linkSetNsFd")
	return netlink.LinkSetNsFd(link, int(ns))
}

// RouteAddNS adds a route into a named network namespace
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

// NSContext allows the caller to define a portion of the code
// where the network namespace is switched
type NSContext struct {
	origins netns.NsHandle
	working netns.NsHandle
}

func (c *NSContext) Enter(nsName string) error {
	log.Info().Str("name", nsName).Msg("enter network namespace")
	// Lock thread to prevent switching of namespaces
	runtime.LockOSThread()

	var err error
	// save handle to host namespace
	c.origins, err = netns.Get()
	if err != nil {
		return err
	}

	// get handle to target namespace
	c.working, err = netns.GetFromName(nsName)
	if err != nil {
		return err
	}

	// set working namespace to target namespace
	return netns.Set(c.working)
}

func (c *NSContext) Exit() error {
	log.Info().Msg("exit network namespace")
	// always unlock thread
	defer runtime.UnlockOSThread()

	// Switch back to the original namespace
	if err := netns.Set(c.origins); err != nil {
		return err
	}
	// close working namespace
	if err := c.working.Close(); err != nil {
		return err
	}
	// close origin namespace
	// if err := c.origin.Close(); err != nil {
	// 	return err
	// }
	return nil
}
