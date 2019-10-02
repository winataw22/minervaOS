package upgrade

import (
	"os"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
