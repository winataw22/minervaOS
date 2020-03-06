package filesystem

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	require := require.New(t)
	var exec TestExecuter
	mgr := TestDeviceManager{
		devices: DeviceCache{
			Device{Path: "/tmp/fakea", Label: "1234"},
			Device{Path: "/tmp/fakeb", Label: "sp_1234"},
			Device{Path: "/tmp/fakec", Children: []Device{
				Device{Path: "/tmp/fakec1", Label: "sp_1234"},
			}},
			Device{Path: "/tmp/faked", Children: []Device{
				Device{Path: "/tmp/faked1", Label: "1234"},
			}},
			Device{Path: "/tmp/fakee"},
		},
	}

	ctx := context.Background()
	exec.On("run", ctx, "wipefs", "-a", "-f", mock.Anything).Return([]byte{}, nil).Run(func(args mock.Arguments) {
		f, err := os.Create(args.String(4))
		if err != nil {
			panic(err)
		}
		f.Close()
	})
	exec.On("run", ctx, "partprobe").Return([]byte{}, nil)
	_, err := migrate(ctx, &mgr, &exec)
	require.NoError(err)

	exec.AssertCalled(t, "run", ctx, "partprobe")

	for _, fake := range []string{"/tmp/fakeb", "/tmp/fakec"} {
		_, err := os.Stat(fake)
		require.NoError(err)
		os.Remove(fake)
	}

	for _, fake := range []string{"/tmp/fakea", "/tmp/faked", "/tmp/fakee"} {
		_, err := os.Stat(fake)
		require.True(os.IsNotExist(err))
	}
}
