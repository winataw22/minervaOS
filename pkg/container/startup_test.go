package container

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
)

var _startup = `[startup]

[startup.entry]
name = "core.system"

[startup.entry.args]
name = "/start"
dir = "/data"

[startup.entry.args.env]
DIFFICULTY = "easy"
LEVEL = "world"
SERVER_PORT = "25565"
`

func TestParseStartup(t *testing.T) {
	r := strings.NewReader(_startup)
	e := startup{}
	_, err := toml.DecodeReader(r, &e)
	require.NoError(t, err)

	entry, ok := e.Entries["entry"]
	require.True(t, ok)
	assert.Equal(t, "core.system", entry.Name)
	assert.Equal(t, "/start", entry.Args.Name)
	assert.Equal(t, "/data", entry.Args.Dir)
	assert.Equal(t, map[string]string{
		"DIFFICULTY":  "easy",
		"LEVEL":       "world",
		"SERVER_PORT": "25565",
	}, entry.Args.Env)
}

func TestStartupEntrypoint(t *testing.T) {
	r := strings.NewReader(_startup)
	e := startup{}
	_, err := toml.DecodeReader(r, &e)
	require.NoError(t, err)

	entry, ok := e.Entries["entry"]
	require.True(t, ok)
	assert.Equal(t, entry.Entrypoint(), "/start")
}

func TestStartupEnvs(t *testing.T) {
	r := strings.NewReader(_startup)
	e := startup{}
	_, err := toml.DecodeReader(r, &e)
	require.NoError(t, err)

	entry, ok := e.Entries["entry"]
	require.True(t, ok)
	actual := entry.Envs()
	sort.Strings(actual)
	expected := []string{
		"DIFFICULTY=easy",
		"LEVEL=world",
		"SERVER_PORT=25565",
	}
	assert.Equal(t, expected, actual)
}

func TestStartupWorkingDir(t *testing.T) {
	r := strings.NewReader(_startup)
	e := startup{}
	_, err := toml.DecodeReader(r, &e)
	require.NoError(t, err)

	entry, ok := e.Entries["entry"]
	require.True(t, ok)
	assert.Equal(t, entry.WorkingDir(), "/data")
}

func TestMergeEnvs(t *testing.T) {
	actual := mergeEnvs(
		[]string{"FOO=BAR", "HELLO=WORLD"},
		[]string{"HELLO=HELLO"},
	)

	expected := []string{"FOO=BAR", "HELLO=WORLD"}
	sort.Strings(actual)
	assert.Equal(t, expected, actual)
}
