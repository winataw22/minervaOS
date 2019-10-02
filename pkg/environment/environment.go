package environment

import (
	"os"

	"github.com/threefoldtech/zos/pkg/kernel"
)

// Environment holds information about running environment of a node
// it defines the different constant based on the running mode (dev, test, prod)
type Environment struct {
	RunningMode RunningMode

	BcdbURL       string
	BcdbNamespace string
	BcdbPassword  string

	FarmerID string
	Orphan   bool

	// ProvisionTimeout  int64
	// ProvisionInterval int64
}

// RunningMode type
type RunningMode string

// Possible running mode of a node
const (
	RunningDev  RunningMode = "dev"
	RunningTest RunningMode = "test"
	RunningMain RunningMode = "prod"

	// Orphanage is the default farmid where nodes are registered
	// if no farmid were specified on the kernel command line
	OrphanageDev  string = "FBresPWUedSi5rBdfhVEr969dCinfq2GpBSdjiM6UrAb"
	OrphanageTest string = "FBresPWUedSi5rBdfhVEr969dCinfq2GpBSdjiM6UrAb"
	OrphanageMain string = "undefined-yet"
)

var (
	envDev = Environment{
		RunningMode: RunningDev,
		BcdbURL:     "https://bcdb.dev.grid.tf",
		// ProvisionTimeout:  60,
		// ProvisionInterval: 10,
	}

	envTest = Environment{
		RunningMode:   RunningTest,
		BcdbURL:       "tcp://bcdb.test.grid.tf:8901",
		BcdbNamespace: "default",
		// ProvisionTimeout:  120,
		// ProvisionInterval: 10,
	}

	envProd = Environment{
		RunningMode:   RunningMain,
		BcdbURL:       "tcp://172.17.0.2:8901", //TODO: change once BCDB is online
		BcdbNamespace: "default",
		// ProvisionTimeout:  240,
		// ProvisionInterval: 20,
	}
)

// Get return the running environment of the node
func Get() Environment {
	params := kernel.GetParams()
	return getEnvironmentFromParams(params)
}

func getEnvironmentFromParams(params kernel.Params) Environment {
	var runmode []string
	var env Environment

	runmode, found := params.Get("runmode")
	if !found {
		// Fallback to default production mode
		runmode = make([]string, 1)
		runmode[0] = string(RunningMain)
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

	farmerID, found := params.Get("farmer_id")
	if !found || farmerID[0] == "" {
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
		env.FarmerID = farmerID[0]
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_BCDB_URL"); e != "" {
		env.BcdbURL = e
	}

	if e := os.Getenv("ZOS_BCDB_NAMESPACE"); e != "" {
		env.BcdbNamespace = e
	}

	if e := os.Getenv("ZOS_BCDB_PASSWORD"); e != "" {
		env.BcdbPassword = e
	}

	// if e := os.Getenv("ZOS_PROVISION_INTERVAL"); e != "" {
	// 	if i, err := strconv.ParseInt(e, 10, 64); err == nil {
	// 		env.ProvisionInterval = i
	// 	}
	// }

	// if e := os.Getenv("ZOS_PROVISION_TIMEOUT"); e != "" {
	// 	if i, err := strconv.ParseInt(e, 10, 64); err == nil {
	// 		env.ProvisionTimeout = i
	// 	}
	// }

	return env
}
