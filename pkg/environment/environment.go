package environment

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/kernel"
)

// Environment holds information about running environment of a node
// it defines the different constant based on the running mode (dev, test, prod)
type Environment struct {
	RunningMode RunningMode

	BcdbURL      string
	BcdbPassword string

	FlistURL string
	BinRepo  string

	FarmerID pkg.FarmID
	Orphan   bool
}

// RunningMode type
type RunningMode string

func (r RunningMode) String() string {
	switch r {
	case RunningDev:
		return "development"
	case RunningMain:
		return "production"
	case RunningTest:
		return "testing"
	}

	return "unknown"
}

// Possible running mode of a node
const (
	RunningDev  RunningMode = "dev"
	RunningTest RunningMode = "test"
	RunningMain RunningMode = "prod"

	// Orphanage is the default farmid where nodes are registered
	// if no farmid were specified on the kernel command line
	OrphanageDev  pkg.FarmID = 0
	OrphanageTest pkg.FarmID = 0
	OrphanageMain pkg.FarmID = 0
)

var (
	envDev = Environment{
		RunningMode: RunningDev,
		BcdbURL:     "https://explorer.devnet.grid.tf/explorer",
		FlistURL:    "zdb://hub.grid.tf:9900",
		BinRepo:     "tf-zos-bins.dev",
	}

	envTest = Environment{
		RunningMode: RunningTest,
		BcdbURL:     "https://explorer.testnet.grid.tf/explorer",
		FlistURL:    "zdb://hub.grid.tf:9900",
		BinRepo:     "tf-zos-bins.test",
	}

	// same as testnet for now. will be updated the day of the launch of production network
	envProd = Environment{
		RunningMode: RunningMain,
		BcdbURL:     "https://explorer.grid.tf/explorer",
		FlistURL:    "zdb://hub.grid.tf:9900",
		BinRepo:     "tf-zos-bins",
	}
)

// Get return the running environment of the node
func Get() (Environment, error) {
	params := kernel.GetParams()
	return getEnvironmentFromParams(params)
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
	case RunningDev:
		env = envDev
	case RunningTest:
		env = envTest
	case RunningMain:
		env = envProd
	default:
		env = envProd
	}

	if RunningMode(runmode[0]) == RunningDev {
		//allow override of the bcdb url in dev mode
		bcdb, found := params.Get("bcdb")
		if found && len(bcdb) >= 1 {
			env.BcdbURL = bcdb[0]
		}
	}

	farmerID, found := params.Get("farmer_id")
	if !found || len(farmerID) < 1 || farmerID[0] == "" {
		// fmt.Println("Warning: no valid farmer_id found in kernel parameter, fallback to orphanage")
		env.Orphan = true

		switch env.RunningMode {
		case RunningDev:
			env.FarmerID = OrphanageDev
		case RunningTest:
			env.FarmerID = OrphanageTest
		case RunningMain:
			env.FarmerID = OrphanageMain
		}

	} else {
		env.Orphan = false
		id, err := strconv.ParseUint(farmerID[0], 10, 64)
		if err != nil {
			return env, errors.Wrap(err, "wrong format for farm ID")
		}
		env.FarmerID = pkg.FarmID(id)
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_BCDB_URL"); e != "" {
		env.BcdbURL = e
	}

	if e := os.Getenv("ZOS_BCDB_PASSWORD"); e != "" {
		env.BcdbPassword = e
	}

	if e := os.Getenv("ZOS_FLIST_URL"); e != "" {
		env.FlistURL = e
	}

	if e := os.Getenv("ZOS_BIN_REPO"); e != "" {
		env.BinRepo = e
	}

	return env, nil
}
