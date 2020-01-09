package namespace

import (
	"context"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// Monitor the dmz namespace for updates
func Monitor(ctx context.Context, name string) (chan netlink.AddrUpdate, error) {
	ns, err := netns.GetFromName(name)
	if err != nil {
		return nil, err
	}

	ch := make(chan netlink.AddrUpdate)
	if err := netlink.AddrSubscribeAt(ns, ch, ctx.Done()); err != nil {
		close(ch)
		ns.Close()
		return nil, err
	}

	return ch, nil
}
