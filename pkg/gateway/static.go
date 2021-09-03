package gateway

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed static/config.yaml
var config string

// staticConfig write static config to file
func staticConfig(p, root string) error {
	config := fmt.Sprintf(config, root)
	return os.WriteFile(p, []byte(config), 0644)
}
