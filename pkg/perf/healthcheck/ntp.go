package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const acceptableSkew = 10 * time.Minute

func RunNTPCheck(ctx context.Context) {
	operation := func() error {
		return ntpCheck(ctx)
	}
	go func() {
		for {
			exp := backoff.NewExponentialBackOff()
			retryNotify := func(err error, d time.Duration) {
				log.Error().Err(err).Msg("failed to run ntp check")
			}

			if err := backoff.RetryNotify(operation, backoff.WithContext(exp, ctx), retryNotify); err != nil {
				log.Error().Err(err).Send()
				continue
			}

			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error().Err(ctx.Err()).Msg("cli context done with error")
				}
				return
			case <-time.After(time.Minute):
			}
		}
	}()
}

func ntpCheck(ctx context.Context) error {
	z := zinit.Default()
	zcl, err := perf.TryGetZbusClient(ctx)
	if err != nil {
		return fmt.Errorf("ntpCheck expects zbus client in the context and found none %w", err)
	}
	utcTime, err := getCurrentUTCTime(zcl)
	if err != nil {
		return err
	}

	if math.Abs(float64(time.Since(utcTime))) > float64(acceptableSkew) {
		if err := z.Kill("ntp", zinit.SIGTERM); err != nil {
			return errors.Wrapf(err, "failed to restart ntpd")
		}
	}

	return nil
}

func getCurrentUTCTime(zcl zbus.Client) (time.Time, error) {

	// TimeServer represents a time server with its name and fetching function
	type TimeServer struct {
		Name string
		Func func() (time.Time, error)
	}

	// List of time servers, and here not in the global vars, so we can inject zcl to pass to getTimeChainWithZCL
	var timeServers = []TimeServer{
		{
			Name: "tfchain",
			Func: func() (time.Time, error) {
				return getTimeChainWithZCL(zcl)
			},
		},
		{
			Name: "worldtimeapi",
			Func: getWorldTimeAPI,
		},
		{
			Name: "worldclockapi",
			Func: getWorldClockAPI,
		},
		{
			Name: "timeapi.io",
			Func: getTimeAPI,
		},
	}
	for _, server := range timeServers {
		log.Info().Msg(fmt.Sprint("running NTP check against ", server.Name))
		utcTime, err := server.Func()
		if err == nil {
			log.Info().Msg(fmt.Sprint("utc time from ", server.Name, ": ", utcTime))
			return utcTime, nil
		}
		log.Error().Err(err).Str("server", server.Name).Msg("failed to get time from server")
	}
	return time.Time{}, errors.New("failed to get time from all servers")
}

func getWorldTimeAPI() (time.Time, error) {
	timeRes, err := http.Get("https://worldtimeapi.org/api/timezone/UTC")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date from worldtimeapi")
	}
	defer timeRes.Body.Close()

	var utcTime struct {
		DateTime time.Time `json:"datetime"`
	}
	if err := json.NewDecoder(timeRes.Body).Decode(&utcTime); err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response from worldtimeapi")
	}

	return utcTime.DateTime, nil
}

func getWorldClockAPI() (time.Time, error) {
	timeRes, err := http.Get("http://worldclockapi.com/api/json/utc/now")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date from worldclockapi")
	}
	defer timeRes.Body.Close()

	var utcTime struct {
		CurrentDateTime string `json:"currentDateTime"` // Changed to string, needs manual parsing
	}
	if err := json.NewDecoder(timeRes.Body).Decode(&utcTime); err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response from worldclockapi")
	}

	// Parse the time manually, handling the "Z"
	return time.Parse("2006-01-02T15:04Z", utcTime.CurrentDateTime)
}

func getTimeAPI() (time.Time, error) {
	timeRes, err := http.Get("https://timeapi.io/api/Time/current/zone?timeZone=UTC")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date from timeapi.io")
	}
	defer timeRes.Body.Close()

	var utcTime struct {
		DateTime string `json:"dateTime"` // Changed to string, needs manual parsing
	}
	if err := json.NewDecoder(timeRes.Body).Decode(&utcTime); err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response from timeapi.io")
	}

	// Parse the time manually, handling the fractional seconds
	return time.Parse("2006-01-02T15:04:05.999999", utcTime.DateTime)
}

func getTimeChainWithZCL(zcl zbus.Client) (time.Time, error) {
	gw := stubs.NewSubstrateGatewayStub(zcl)
	return gw.GetTime(context.Background())
}
