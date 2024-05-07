package healthcheck

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const acceptableSkew = 10 * time.Minute

func RunNTPCheck(ctx context.Context) {
	go func() {
		for {
			if err := ntpCheck(); err != nil {
				log.Error().Err(err).Msg("failed to run ntp check")
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

func ntpCheck() error {
	z := zinit.Default()

	utcTime, err := getCurrentUTCTime()
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

func getCurrentUTCTime() (time.Time, error) {
	timeRes, err := http.Get("https://worldtimeapi.org/api/timezone/UTC")
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to get date")
	}

	var utcTime struct {
		DateTime time.Time `json:"datetime"`
	}
	err = json.NewDecoder(timeRes.Body).Decode(&utcTime)
	timeRes.Body.Close()
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "failed to decode date response")
	}

	return utcTime.DateTime, nil
}
