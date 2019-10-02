package upgrade

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/version"
)

func revisionOf(bin string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get '%s' version string", bin)
	}

	_, revision, err := version.Parse(strings.TrimSpace(string(output)))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse current revision")
	}

	return revision, nil
}

// currentRevision tries to find the revision of the
// running process. it depends on the binary ability
// to respond to ./<binary> -v and respond in a
// valid version string as defined by `modules/version`
func currentRevision() string {
	revision, err := revisionOf(os.Args[0], 1*time.Second)
	if err != nil {
		panic(err)
	}

	return revision
}

func currentBinPath() string {
	bin, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(errors.Wrap(err, "failed to resolve the upgrade daemon path"))
	}

	bin, err = filepath.Abs(bin)
	if err != nil {
		panic(errors.Wrap(err, "failed to resolve the upgrade daemon path"))
	}

	return bin
}
