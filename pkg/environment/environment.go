package environment

import (
	"os"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/kernel"
)

const (
	baseExtendedURL = "https://raw.githubusercontent.com/threefoldtech/zos-config/main/"
)

// Environment holds information about running environment of a node
// it defines the different constant based on the running mode (dev, test, prod)
type Environment struct {
	RunningMode RunningMode

	FlistURL string
	BinRepo  string

	FarmID pkg.FarmID
	Orphan bool

	FarmSecret    string
	SubstrateURL  []string
	RelayURL      string
	ActivationURL string

	ExtendedConfigURL string
}

// RunningMode type
type RunningMode string

func (r RunningMode) String() string {
	switch r {
	case RunningDev:
		return "development"
	case RunningQA:
		return "qa"
	case RunningMain:
		return "production"
	case RunningTest:
		return "testing"
	}

	return "unknown"
}

// Possible running mode of a node
const (
	// RunningDev mode
	RunningDev RunningMode = "dev"
	// RunningQA mode
	RunningQA RunningMode = "qa"
	// RunningTest mode
	RunningTest RunningMode = "test"
	// RunningMain mode
	RunningMain RunningMode = "prod"

	// Orphanage is the default farmid where nodes are registered
	// if no farmid were specified on the kernel command line
	OrphanageDev  pkg.FarmID = 0
	OrphanageTest pkg.FarmID = 0
	OrphanageMain pkg.FarmID = 0
)

var (
	pool     substrate.Manager
	poolOnce sync.Once

	envDev = Environment{
		RunningMode: RunningDev,
		SubstrateURL: []string{
			"wss://tfchain.dev.grid.tf/",
		},
		RelayURL:      "wss://relay.dev.grid.tf",
		ActivationURL: "https://activation.dev.grid.tf/activation/activate",
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins.dev",
	}

	envTest = Environment{
		RunningMode: RunningTest,
		SubstrateURL: []string{
			"wss://tfchain.test.grid.tf/",
		},
		RelayURL:      "wss://relay.test.grid.tf",
		ActivationURL: "https://activation.test.grid.tf/activation/activate",
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins.test",
	}

	envQA = Environment{
		RunningMode: RunningQA,
		SubstrateURL: []string{
			"wss://tfchain.qa.grid.tf/",
		},
		RelayURL:      "wss://relay.qa.grid.tf",
		ActivationURL: "https://activation.qa.grid.tf/activation/activate",
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins.qanet",
	}

	envProd = Environment{
		RunningMode: RunningMain,
		SubstrateURL: []string{
			"wss://tfchain.grid.tf/",
			"wss://02.tfchain.grid.tf/",
			"wss://03.tfchain.grid.tf/",
			"wss://04.tfchain.grid.tf/",
		},
		RelayURL:      "wss://relay.grid.tf",
		ActivationURL: "https://activation.grid.tf/activation/activate",
		FlistURL:      "redis://hub.grid.tf:9900",
		BinRepo:       "tf-zos-v3-bins",
	}
)

// MustGet returns the running environment of the node
// panics on error
func MustGet() Environment {
	env, err := Get()
	if err != nil {
		panic(err)
	}

	return env
}

// Get return the running environment of the node
func Get() (Environment, error) {
	params := kernel.GetParams()
	return getEnvironmentFromParams(params)
}

// GetSubstrate gets a client to subsrate blockchain
func GetSubstrate() (substrate.Manager, error) {
	env, err := Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get boot environment")
	}

	poolOnce.Do(func() {
		pool = substrate.NewManager(env.SubstrateURL...)
	})

	return pool, nil
}

func getEnvironmentFromParams(params kernel.Params) (Environment, error) {
	var env Environment
	runmode := ""
	if modes, ok := params.Get("runmode"); ok {
		if len(modes) >= 1 {
			runmode = modes[0]
		}
	} else {
		runmode = os.Getenv("ZOS_RUNMODE")
	}

	if len(runmode) == 0 {
		runmode = string(RunningMain)
	}

	switch RunningMode(runmode) {
	case RunningDev:
		env = envDev
	case RunningQA:
		env = envQA
	case RunningTest:
		env = envTest
	case RunningMain:
		env = envProd
	default:
		env = envProd
	}

	extended, found := params.Get("config_url")
	if found && len(extended) >= 1 {
		env.ExtendedConfigURL = extended[len(extended)-1]
	}

	if substrate, ok := params.Get("substrate"); ok {
		if len(substrate) > 0 {
			env.SubstrateURL = substrate
		}
	}

	if relay, ok := params.Get("relay"); ok {
		if len(relay) > 0 {
			env.RelayURL = relay[len(relay)-1]
		}
	}

	if activation, ok := params.Get("activation"); ok {
		if len(activation) > 0 {
			env.ActivationURL = activation[len(activation)-1]
		}
	}

	if farmSecret, ok := params.Get("secret"); ok {
		if len(farmSecret) > 0 {
			env.FarmSecret = farmSecret[len(farmSecret)-1]
		}
	}

	farmerID, found := params.Get("farmer_id")
	if !found || len(farmerID) < 1 || farmerID[0] == "" {
		// fmt.Println("Warning: no valid farmer_id found in kernel parameter, fallback to orphanage")
		env.Orphan = true

		switch env.RunningMode {
		case RunningDev:
			env.FarmID = OrphanageDev
		case RunningTest:
			env.FarmID = OrphanageTest
		case RunningMain:
			env.FarmID = OrphanageMain
		}

	} else {
		env.Orphan = false
		id, err := strconv.ParseUint(farmerID[0], 10, 32)
		if err != nil {
			return env, errors.Wrap(err, "wrong format for farm ID")
		}
		env.FarmID = pkg.FarmID(id)
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_SUBSTRATE_URL"); e != "" {
		env.SubstrateURL = []string{e}
	}

	if e := os.Getenv("ZOS_FLIST_URL"); e != "" {
		env.FlistURL = e
	}

	if e := os.Getenv("ZOS_BIN_REPO"); e != "" {
		env.BinRepo = e
	}

	return env, nil
}
