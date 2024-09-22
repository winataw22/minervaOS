package power

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type Flag string

const (
	SupportsWakeOn Flag = "Supports Wake-on"
	WakeOn         Flag = "Wake-on"
)

type WolMode string

const (
	MagicPacket WolMode = "g"
)

var (
	ErrFlagNotFound = fmt.Errorf("flag not found")
)

func ethtool(ctx context.Context, arg ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "ethtool", arg...).CombinedOutput()
}

func ValueOfFlag(ctx context.Context, nic string, flag Flag) (string, error) {
	output, err := ethtool(ctx, nic)
	if err != nil {
		return "", err
	}

	return valueOfFlag(output, flag)
}

func valueOfFlag(output []byte, flag Flag) (string, error) {
	buf := bytes.NewBuffer(output)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}

		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ":")
		if parts[0] != string(flag) {
			continue
		}
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid ethtool output format (%s)", line)
		}
		return strings.TrimSpace(parts[1]), nil
	}

	return "", ErrFlagNotFound
}

func SetWol(ctx context.Context, nic string, mode WolMode) error {
	_, err := ethtool(ctx, "-s", nic, "wol", string(mode))
	return errors.Wrap(err, "failed to set nic wol")
}
