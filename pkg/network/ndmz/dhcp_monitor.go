package ndmz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/vishvananda/netlink"
)

// DHCPMon monitor a network interface status and force
// renew of DHCP lease if needed
type DHCPMon struct {
	z         *zinit.Client
	service   string
	iface     string
	namespace string
}

// NewDHCPMon create a new DHCPMon object managing interface iface
// namespace is then network namespace name to use. it can be empty.
func NewDHCPMon(iface, namespace string, z *zinit.Client) *DHCPMon {
	return &DHCPMon{
		z:         z,
		service:   fmt.Sprintf("dhcp-%s", iface),
		iface:     iface,
		namespace: namespace,
	}
}

// Start creates a zinit service for a DHCP client and start monitoring it
// this method is blocking, start is in a goroutine if needed.
// cancel the context to start it.
func (d *DHCPMon) Start(ctx context.Context) error {

	if err := d.startZinit(); err != nil {
		return err
	}
	defer func() {
		if err := d.stopZinit(); err != nil {
			log.Error().Err(err).Msgf("error stopping zinit service %s", d.service)
		}
	}()

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-t.C:
			has, err := hasDefaultRoute(d.iface, d.namespace)
			if err != nil {
				log.Error().Str("iface", d.iface).Err(err).Msg("error checking default gateway")
				continue
			}

			if !has {
				log.Info().Msg("ndmz default route missing, waking up udhcpc")
				if err := d.wakeUp(); err != nil {
					log.Error().Err(err).Msg("error while sending signal to service ")
				}
			}
		}
	}
}

// wakeUp sends a signal to the udhcpc daemon to force a release of the DHCP lease
func (d *DHCPMon) wakeUp() error {
	err := d.z.Kill(d.service, zinit.SIGUSR1)
	if err != nil {
		log.Error().Err(err).Msg("error while sending signal to service ")
	}
	return err
}

// hasDefaultRoute checks if the network interface iface has a default route configured
// if netNS is not empty, switch to the network namespace named netNS before checking the routes
func hasDefaultRoute(iface, netNS string) (bool, error) {
	var hasDefault bool
	do := func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return err
		}
		hasDefault, _, err = ifaceutil.HasDefaultGW(link, netlink.FAMILY_V4)
		return err
	}

	var oerr error
	if netNS != "" {
		n, err := namespace.GetByName(netNS)
		if err != nil {
			return false, err
		}
		oerr = n.Do(do)
	} else {
		oerr = do(nil)
	}
	return hasDefault, oerr
}

func (d *DHCPMon) startZinit() error {
	status, err := d.z.Status(d.service)
	if err != nil && err != zinit.ErrUnknownService {
		log.Error().Err(err).Msgf("error checking zinit service %s status", d.service)
		return err
	}

	if status.State.Exited() {
		log.Info().Msgf("zinit service %s already exists but is stopped, starting it", d.service)
		return d.z.Start(d.service)
	}

	log.Info().Msgf("create and start %s zinit service", d.service)
	exec := fmt.Sprintf("/sbin/udhcpc -v -f -i %s -t 20 -T 1 -s /usr/share/udhcp/simple.script", d.iface)

	if d.namespace != "" {
		exec = fmt.Sprintf("ip netns exec %s %s", strings.TrimSpace(d.namespace), exec)
	}

	err = zinit.AddService(d.service, zinit.InitService{
		Exec:    exec,
		Oneshot: false,
		After:   []string{},
	})

	if err != nil {
		log.Error().Err(err).Msg("fail to create dhcp-zos zinit service")
		return err
	}

	if err := d.z.Monitor(d.service); err != nil {
		log.Error().Err(err).Msg("fail to start monitoring dhcp-zos zinit service")
		return err
	}

	return err
}

// Stop stops a zinit background process
func (d *DHCPMon) stopZinit() error {
	err := d.z.StopWait(time.Second*10, d.service)
	if err != nil {
		return errors.Wrapf(err, "failed to stop zinit service %s", d.service)
	}
	return d.z.Forget(d.service)
}
