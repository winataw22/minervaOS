package container

import (
	"fmt"
	"strings"
)

type startup struct {
	Entries map[string]entry `toml:"startup"`
}

type entry struct {
	Name string
	Args args
}

type args struct {
	Name string
	Dir  string
	Args []string
	Env  map[string]string
}

func (e entry) Entrypoint() string {
	if e.Name == "core.system" ||
		e.Name == "core.base" && e.Args.Name != "" {
		var buf strings.Builder
		buf.WriteString(e.Args.Name)
		for _, arg := range e.Args.Args {
			buf.WriteRune(' ')
			arg = strings.Replace(arg, "\"", "\\\"", -1)
			buf.WriteRune('"')
			buf.WriteString(arg)
			buf.WriteRune('"')
		}

		return buf.String()
	}

	return ""
}

func (e entry) WorkingDir() string {
	return e.Args.Dir
}

func (e entry) Envs() []string {
	envs := make([]string, 0, len(e.Args.Env))
	for k, v := range e.Args.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}

// mergeEnvs merge a into b
// all the key from a will endup in b
// if a key is present in both, key from a are kept
func mergeEnvs(a, b []string) []string {
	m := make(map[string]string, len(a)+len(b))

	for _, s := range b {
		ss := strings.SplitN(s, "=", 2)
		m[ss[0]] = ss[1]
	}
	for _, s := range a {
		ss := strings.SplitN(s, "=", 2)
		m[ss[0]] = ss[1]
	}

	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
