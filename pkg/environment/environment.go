package environment

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/farmer"
	"github.com/threefoldtech/zos/pkg/kernel"
)

// Environment holds information about running environment of a node
// it defines the different constant based on the running mode (dev, test, prod)
type Environment struct {
	RunningMode RunningMode

	FlistURL string
	BinRepo  string

	FarmerID pkg.FarmID
	Orphan   bool

	FarmSecret   string
	SubstrateURL string
}

// RunningMode type
type RunningMode string

func (r RunningMode) String() string {
	switch r {
	case RunningDev3:
		return "development"
	case RunningMain:
		return "production"
	case RunningTest3:
		return "testing"
	}

	return "unknown"
}

// Possible running mode of a node
const (
	//RunningDev   RunningMode = "dev"
	RunningDev3 RunningMode = "dev3"
	//RunningTest  RunningMode = "test"
	RunningTest3 RunningMode = "test3"
	RunningMain  RunningMode = "prod"

	// Orphanage is the default farmid where nodes are registered
	// if no farmid were specified on the kernel command line
	OrphanageDev  pkg.FarmID = 0
	OrphanageTest pkg.FarmID = 0
	OrphanageMain pkg.FarmID = 0
)

var (
	envDev = Environment{
		RunningMode:  RunningDev3,
		SubstrateURL: "wss://explorer.devnet.grid.tf/ws",
		FlistURL:     "zdb://hub.grid.tf:9900",
		BinRepo:      "tf-zos-bins.dev",
	}

	envTest = Environment{
		RunningMode: RunningTest3,
		// TODO: this should become a different substrate ?
		SubstrateURL: "wss://explorer.devnet.grid.tf/ws",
		FlistURL:     "zdb://hub.grid.tf:9900",
		BinRepo:      "tf-zos-bins.test",
	}

	// same as testnet for now. will be updated the day of the launch of production network
	envProd = Environment{
		RunningMode:  RunningMain,
		SubstrateURL: "wss://explorer.devnet.grid.tf/ws",
		FlistURL:     "zdb://hub.grid.tf:9900",
		BinRepo:      "tf-zos-bins",
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

// FarmerClient gets a client to the farm
func (v *Environment) FarmerClient() (*farmer.Client, error) {
	return farmer.NewClientFromSubstrate(v.SubstrateURL, uint32(v.FarmerID))
}

func getEnvironmentFromParams(params kernel.Params) (Environment, error) {
	var runmode []string
	var env Environment

	runmode, found := params.Get("runmode")
	if !found || len(runmode) < 1 {
		// Fallback to default production mode
		runmode = []string{string(RunningMain)}
	}

	switch RunningMode(runmode[0]) {
	case RunningDev3:
		env = envDev
	case RunningTest3:
		env = envTest
	case RunningMain:
		env = envProd
	default:
		env = envProd
	}

	if substrate, ok := params.Get("substrate"); ok {
		if len(substrate) > 0 {
			env.SubstrateURL = substrate[len(substrate)-1]
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
		case RunningDev3:
			env.FarmerID = OrphanageDev
		case RunningTest3:
			env.FarmerID = OrphanageTest
		case RunningMain:
			env.FarmerID = OrphanageMain
		}

	} else {
		env.Orphan = false
		id, err := strconv.ParseUint(farmerID[0], 10, 32)
		if err != nil {
			return env, errors.Wrap(err, "wrong format for farm ID")
		}
		env.FarmerID = pkg.FarmID(id)
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_SUBSTRATE_URL"); e != "" {
		env.SubstrateURL = e
	}

	if e := os.Getenv("ZOS_FLIST_URL"); e != "" {
		env.FlistURL = e
	}

	if e := os.Getenv("ZOS_BIN_REPO"); e != "" {
		env.BinRepo = e
	}

	return env, nil
}
