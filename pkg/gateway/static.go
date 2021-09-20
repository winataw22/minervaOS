package gateway

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

//go:embed static/config.yaml
var config string

//go:embed static/dnsmasq.conf
var dConfig string

//go:embed static/cert.sh
var certScript string

// staticConfig write static config to file
func staticConfig(p, root, email string) (bool, error) {
	config := fmt.Sprintf(config, root, email)

	var update bool
	if oldConfig, err := os.ReadFile(p); os.IsNotExist(err) {
		update = true
	} else if err != nil {
		return false, errors.Wrap(err, "failed to read traefik config")
	} else {
		// no errors
		update = !bytes.Equal([]byte(config), oldConfig)
	}

	return update, os.WriteFile(p, []byte(config), 0644)
}

func updateCertScript(p, root string) error {
	certScript := fmt.Sprintf(certScript, root)
	return os.WriteFile(p, []byte(certScript), 0744)
}

func dnsmasqConfig(p string) error {
	return os.WriteFile(p, []byte(dConfig), 0644)
}
