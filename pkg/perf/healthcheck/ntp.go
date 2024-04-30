package healthcheck

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const acceptableSkew = 10 * time.Minute

func ntpCheck(ctx context.Context) []error {
	var errs []error
	z := zinit.Default()

	utcTime, err := getCurrentUTCTime()
	if err != nil {
		errs = append(errs, err)
	}

	if math.Abs(float64(time.Since(utcTime))) > float64(acceptableSkew) {
		if err := z.Kill("ntp", zinit.SIGTERM); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to restart ntpd"))
		}
	}

	return errs
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
