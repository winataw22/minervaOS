package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
)

const defaultRequestTimeout = 5 * time.Second

func networkCheck(ctx context.Context) []error {
	env := environment.MustGet()
	servicesUrl := []string{env.FlistURL}

	servicesUrl = append(append(servicesUrl, env.SubstrateURL...), env.RelayURL...)
	servicesUrl = append(append(servicesUrl, env.ActivationURL...), env.GraphQL...)

	var errors []error

	var wg sync.WaitGroup
	var mut sync.Mutex
	for _, serviceUrl := range servicesUrl {
		wg.Add(1)
		go func(serviceUrl string) {
			defer wg.Done()

			err := checkService(ctx, serviceUrl)
			if err != nil {
				mut.Lock()
				defer mut.Unlock()

				errors = append(errors, err)
			}
		}(serviceUrl)
	}
	wg.Wait()

	if len(errors) == 0 {
		if err := app.DeleteFlag(app.NotReachable); err != nil {
			log.Error().Err(err).Msg("failed to delete readonly flag")
		}
	}

	return errors
}

func checkService(ctx context.Context, serviceUrl string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	address := parseUrl(serviceUrl)
	err := isReachable(ctx, address)
	if err != nil {
		if err := app.SetFlag(app.NotReachable); err != nil {
			log.Error().Err(err).Msg("failed to set not reachable flag")
		}
		return fmt.Errorf("%s is not reachable: %w", serviceUrl, err)
	}

	return nil
}

func parseUrl(serviceUrl string) string {
	u, err := url.Parse(serviceUrl)
	if err != nil {
		return ""
	}

	port := ":80"
	if u.Scheme == "https" || u.Scheme == "wss" {
		port = ":443"
	}

	if u.Port() == "" {
		u.Host += port
	}

	return u.Host
}

func isReachable(ctx context.Context, address string) error {
	d := net.Dialer{Timeout: defaultRequestTimeout}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	return nil
}
