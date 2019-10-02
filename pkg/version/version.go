package version

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

/*
The constants in this file are auto-replaced with the actual values
during the build of the module.
*/

var (
	// Branch of the code
	Branch = "{branch}"
	// Revision of the code
	Revision = "{revision}"
	// Dirty flag shows if the binary is built from a
	// repo with uncommitted changes
	Dirty = "{dirty}"
)

var (
	re = regexp.MustCompile(`^Version:([^@]*)@Revision:([^\(]+)`)
)

// Version interface
type Version interface {
	Short() string
	String() string
}

type version struct{}

func (v *version) String() string {
	s := fmt.Sprintf("Version: %s @Revision: %s", Branch, Revision)
	if Dirty != "" {
		s += " (dirty-repo)"
	}

	return s
}

func (v *version) Short() string {
	s := fmt.Sprintf("%s@%s", Branch, Revision[0:7])
	if Dirty != "" {
		s += "(D)"
	}
	return s
}

// Current get current version
func Current() Version {
	return &version{}
}

// ShowAndExit prints the version and exits
func ShowAndExit(short bool) {
	if short {
		fmt.Println(Current().Short())
	} else {
		fmt.Println(Current())
	}

	os.Exit(0)
}

// Parse version string
func Parse(v string) (version string, revision string, err error) {
	m := re.FindStringSubmatch(v)
	if m == nil {
		return version, revision, fmt.Errorf("invalid version string")
	}

	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), nil
}
