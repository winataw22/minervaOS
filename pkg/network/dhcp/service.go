package dhcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	olderExecPath = "/sbin/udhcpc"
	execPath      = "/usr/sbin/dhcpcd"
)

type ClientService struct {
	Name      string
	Iface     string
	Namespace string

	z *zinit.Client
}

func NewService(iface string, namespace string, z *zinit.Client) ClientService {
	return ClientService{
		Name:      fmt.Sprintf("dhcp-%s", iface),
		Iface:     iface,
		Namespace: namespace,
		z:         z,
	}
}

func (s ClientService) getNamespacedExec(exec string) string {
	return fmt.Sprintf("ip netns exec %s %s", strings.TrimSpace(s.Namespace), exec)
}

func (s ClientService) IsUsingOlderClient() bool {
	var filter zinit.Filter
	if s.Namespace != "" {
		regex := fmt.Sprintf(`^%s`, s.getNamespacedExec(olderExecPath))
		filter = zinit.WithExecRegex(regex)
	} else {
		filter = zinit.WithExec("udhcpc")
	}

	olderService, _ := s.z.Matches(zinit.WithName(s.Name), filter)
	return len(olderService) > 0
}

func (s ClientService) Create() error {
	log.Info().Msgf("create %s zinit service", s.Name)

	exec := fmt.Sprintf("%s %s -B", execPath, s.Iface)
	if s.Namespace != "" {
		exec = s.getNamespacedExec(exec)
	}

	err := zinit.AddService(s.Name, zinit.InitService{
		Exec:    exec,
		Oneshot: false,
		After:   []string{},
	})

	if err != nil {
		log.Error().Err(err).Msgf("fail to create %s zinit service", s.Name)
		return err
	}

	if err := s.z.Monitor(s.Name); err != nil && err != zinit.ErrAlreadyMonitored {
		log.Error().Err(err).Msgf("fail to start monitoring %s zinit service", s.Name)
		return err
	}

	return err
}

func (s ClientService) Destroy() error {
	if err := s.z.Destroy(20*time.Second, s.Name); err != nil {
		log.Error().Err(err).Msgf("fail to terminate %s zinit service", s.Name)
		return err
	}

	return nil
}

func (s ClientService) DestroyOlderService() error {
	if s.IsUsingOlderClient() {
		if err := s.Destroy(); err != nil {
			log.Error().Err(err).Msgf("fail to terminate older %s (udhcpc) service", s.Name)
			return err
		}
	}

	return nil
}

func (s ClientService) Start() error {
	return s.z.Start(s.Name)
}

func (s ClientService) Stop() error {
	return s.z.Stop(s.Name)
}
