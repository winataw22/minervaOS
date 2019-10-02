package upgrade

import (
	"encoding/json"
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/require"
)

func TestInfo(t *testing.T) {
	const input = `{"size": 0, "name": "zos:development:latest.flist", "target": "zos:development:0.2.0.flist", "type": "symlink", "updated": 1569924782, "md5": "9798ef9b930b49ab18c45953cf1d2369"}`

	var info FListInfo
	err := json.Unmarshal([]byte(input), &info)
	require.NoError(t, err)

	ver, err := info.Version()
	require.NoError(t, err)

	require.Equal(t, semver.MustParse("0.2.0"), ver)
}
