package filesystem

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestExecuter struct {
	mock.Mock
}

func (t *TestExecuter) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	inputs := []interface{}{ctx, name}
	for _, arg := range args {
		inputs = append(inputs, arg)
	}

	result := t.Called(inputs...)
	return result.Get(0).([]byte), result.Error(1)
}

type TestMap map[string]interface{}

func (m TestMap) Bytes() []byte {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return bytes
}

func TestGetMountTarget(t *testing.T) {
	buf := bytes.NewBufferString(`
/dev/sda2 /home btrfs rw,relatime,ssd,space_cache,subvolid=258,subvol=/home 0 0
/dev/sda1 /boot vfat rw,relatime,fmask=0022,dmask=0022,codepage=437,iocharset=iso8859-1,shortname=mixed,utf8,errors=remount-ro 0 0
/dev/sda2 /var/lib/docker/btrfs btrfs rw,relatime,ssd,space_cache,subvolid=257,subvol=/root/var/lib/docker/btrfs 0 0
tmpfs /run/user/1000 tmpfs rw,nosuid,nodev,relatime,size=1626092k,mode=700,uid=1000,gid=1000 0 0
fusectl /sys/fs/fuse/connections fusectl rw,nosuid,nodev,noexec,relatime 0 0
gvfsd-fuse /run/user/1000/gvfs fuse.gvfsd-fuse rw,nosuid,nodev,relatime,user_id=1000,group_id=1000 0 0
	`)

	target, ok := getMountTarget(buf, "/dev/sda1")
	require.True(t, ok)
	require.Equal(t, "/boot", target)

	target, ok = getMountTarget(buf, "/dev/sda3")
	require.False(t, ok)
	require.Equal(t, "", target)
}
