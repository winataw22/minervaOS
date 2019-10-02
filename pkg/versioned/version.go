package versioned

import "github.com/blang/semver"

// Version type
type Version = semver.Version

// Range type
type Range = semver.Range

// Parse version string
func Parse(s string) (Version, error) {
	return semver.Parse(s)
}

// MustParse version
func MustParse(s string) Version {
	return semver.MustParse(s)
}

// ParseRange parses a range
func ParseRange(s string) (Range, error) {
	return semver.ParseRange(s)
}

// MustParseRange parses range
func MustParseRange(s string) Range {
	return semver.MustParseRange(s)
}
