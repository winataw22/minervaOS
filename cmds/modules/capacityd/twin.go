package capacityd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
)

func runMsgBus(ctx context.Context, twin uint32) error {
	// todo: make it argument or parse from broker
	const redis = "/var/run/redis.sock"
	for {
		cmd := exec.CommandContext(ctx,
			"msgbusd",
			"--twin", fmt.Sprint(twin),
			"--redis", redis)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()

		if err == context.Canceled {
			return nil
		} else if err != nil {
			log.Error().Err(err).Msg("msgbusd exited unexpectedly, retrying")
			<-time.After(2 * time.Second)
		}

		// hmm, so msgbusd exited with no error.
		// see if context is canceled
		select {
		case <-ctx.Done():
			return nil
		default:
			//try again then
		}
	}
}
