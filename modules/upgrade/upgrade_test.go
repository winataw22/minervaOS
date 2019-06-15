package upgrade

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPublisher struct {
	upgrades map[string]Upgrade
	lastest  semver.Version
}

func (p *testPublisher) Get(version semver.Version) (Upgrade, error) {
	u, ok := p.upgrades[version.String()]
	if !ok {
		return Upgrade{}, fmt.Errorf("upgrade not found")
	}
	return u, nil
}
func (p *testPublisher) Latest() (semver.Version, error) {
	return p.lastest, nil
}
func (p *testPublisher) List() ([]semver.Version, error) {
	versions := make([]semver.Version, 0, len(p.upgrades))
	for v := range p.upgrades {
		versions = append(versions, semver.MustParse(v))
	}
	return versions, nil
}

func TestIsNewVersionAvailable(t *testing.T) {
	type testcase struct {
		name      string
		current   semver.Version
		latest    semver.Version
		available bool
	}

	for _, tc := range []testcase{
		{
			name:      "same version",
			current:   semver.MustParse("0.0.1"),
			latest:    semver.MustParse("0.0.1"),
			available: false,
		},
		{
			name:      "upgrade available",
			current:   semver.MustParse("0.0.1"),
			latest:    semver.MustParse("0.1.0"),
			available: true,
		},
		{
			name:      "current higher then latest",
			current:   semver.MustParse("1.1.1"),
			latest:    semver.MustParse("0.1.0"),
			available: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &testPublisher{
				lastest: tc.latest,
			}

			available, found, err := isNewVersionAvailable(tc.current, p)
			require.NoError(t, err)
			assert.Equal(t, tc.available, available)
			if tc.available {
				assert.Equal(t, tc.latest, found)
			}
		})
	}

}

func TestVersionsToApply(t *testing.T) {
	type testcase struct {
		name     string
		current  semver.Version
		latest   semver.Version
		err      error
		upgrades map[string]Upgrade
		versions []semver.Version
	}

	for _, tc := range []testcase{
		{
			name:    "same version",
			current: semver.MustParse("0.0.1"),
			latest:  semver.MustParse("0.0.1"),
			upgrades: map[string]Upgrade{
				"0.0.1": Upgrade{},
			},
			versions: []semver.Version{},
		},
		{
			name:    "upgrade available",
			current: semver.MustParse("0.0.1"),
			latest:  semver.MustParse("0.1.0"),
			upgrades: map[string]Upgrade{
				"0.0.1": Upgrade{},
				"0.0.2": Upgrade{},
				"0.1.0": Upgrade{},
			},
			versions: []semver.Version{
				semver.MustParse("0.0.2"),
				semver.MustParse("0.1.0"),
			},
		},
		{
			name:    "upgrade available2",
			current: semver.MustParse("0.1.1"),
			latest:  semver.MustParse("1.7.2"),
			upgrades: map[string]Upgrade{
				"0.0.1": Upgrade{},
				"0.0.2": Upgrade{},
				"0.1.0": Upgrade{},
				"0.1.1": Upgrade{},
				"0.2.1": Upgrade{},
				"0.3.1": Upgrade{},
				"1.0.0": Upgrade{},
				"1.6.0": Upgrade{},
				"1.7.1": Upgrade{},
				"1.7.2": Upgrade{},
			},
			versions: []semver.Version{
				semver.MustParse("0.2.1"),
				semver.MustParse("0.3.1"),
				semver.MustParse("1.0.0"),
				semver.MustParse("1.6.0"),
				semver.MustParse("1.7.1"),
				semver.MustParse("1.7.2"),
			},
		},
		{
			name:    "current higher then latest",
			current: semver.MustParse("1.1.1"),
			latest:  semver.MustParse("0.1.0"),
			upgrades: map[string]Upgrade{
				"0.0.1": Upgrade{},
				"0.0.2": Upgrade{},
				"0.1.0": Upgrade{},
				"0.1.1": Upgrade{},
			},
			versions: []semver.Version{},
		},
		{
			name:    "lastest is not highest available",
			current: semver.MustParse("0.0.1"),
			latest:  semver.MustParse("0.1.0"),
			upgrades: map[string]Upgrade{
				"0.0.1": Upgrade{},
				"0.0.2": Upgrade{},
				"0.1.0": Upgrade{},
				"0.1.1": Upgrade{},
			},
			versions: []semver.Version{
				semver.MustParse("0.0.2"),
				semver.MustParse("0.1.0"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &testPublisher{
				lastest:  tc.latest,
				upgrades: tc.upgrades,
			}

			toApply, err := versionsToApply(tc.current, tc.latest, p)
			if tc.err != nil {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.EqualValues(t, tc.versions, toApply)
			}
		})
	}
}

func TestLoadVersion(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)

	t.Run("fresh start", func(t *testing.T) {
		version, err := ensureVersionFile(root)
		require.NoError(t, err)
		assert.Equal(t, semver.MustParse("0.0.1"), version)
	})

	t.Run("load existing", func(t *testing.T) {
		writeVersion(filepath.Join(root, "version"), semver.MustParse("0.1.0"))
		version, err := ensureVersionFile(root)
		require.NoError(t, err)
		assert.Equal(t, semver.MustParse("0.1.0"), version)
	})
}

func TestChangeBase(t *testing.T) {

	for _, test := range []struct {
		name   string
		base   string
		path   string
		result string
		err    error
	}{
		{
			name:   "valid",
			base:   "/mnt/containers/1",
			path:   "/etc/modules/module.conf",
			result: "/mnt/containers/1/etc/modules/module.conf",
			err:    nil,
		},
		{
			name: "base not absolute",
			base: "containers/1",
			err:  errNotAbsolute,
		},
		{
			name: "path not absolute",
			base: "",
			path: "etc/modules/module.conf",
			err:  errNotAbsolute,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			result, err := changeRoot(test.base, test.path)
			if test.err != nil {
				assert.Error(t, err)
				assert.Equal(t, test.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.result, result)
			}
		})
	}
}
