package kernel

import (
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/google/shlex"
)

const (
	// Debug means zos is running in debug mode
	// applications can handle this flag differently
	Debug = "zos-debug"
	// VirtualMachine forces zos to think it's running
	// on a virtual machine. used mainly for development
	VirtualMachine = "zos-debug-vm"
)

// Params represent the parameters passed to the kernel at boot
type Params map[string][]string

// Exists checks if a key is present in the kernel parameters
func (k Params) Exists(key string) bool {
	_, ok := k[key]
	return ok
}

// Get returns the values link to a key and a boolean
// boolean if true when the key exists in the params or false otherwise
// a nil list, and a true will be returned if the `key` is set in kernel params, but with
// no associated value
func (k Params) Get(key string) ([]string, bool) {
	v, ok := k[key]
	return v, ok
}

func (k Params) IsDebug() bool {
	_, ok := k.Get(Debug)
	return ok
}

func (k Params) IsVirtualMachine() bool {
	_, ok := k.Get(VirtualMachine)
	return ok
}

func parseParams(content string) Params {
	options := Params{}
	cmdline, _ := shlex.Split(strings.TrimSpace(content))
	for _, option := range cmdline {
		kv := strings.SplitN(option, "=", 2)
		key := kv[0]

		if len(kv) == 2 {
			options[key] = append(options[key], kv[1])
		} else {
			options[key] = nil
		}
	}

	return options
}

//GetParams Get kernel cmdline arguments
func GetParams() Params {
	content, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		log.Error().Err(err).Msg("Failed to read /proc/cmdline")
		return Params{}
	}

	return parseParams(string(content))
}
