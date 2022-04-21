package rotate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// RotateAll will rotate all files in the directory if a set of
// names is given only named files will be rotated, other unknown
// files will be deleted
func (r *Rotator) RotateAll(dir string, names ...string) error {
	files, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to list directory '%s' files: %w", dir, err)
	}

	namesMap := make(map[string]struct{})
	for _, n := range names {
		namesMap[n] = struct{}{}
	}

	for _, file := range files {
		name := file.Name()
		log.Debug().Str("file", name).Msg("checking file for rotation")
		path := filepath.Join(dir, name)
		base := strings.TrimSuffix(name, r.suffix)
		if _, ok := namesMap[base]; !ok {
			log.Debug().Str("file", name).Msg("log file not tracked, deleting...")
			_ = os.Remove(path)
			continue
		}

		if strings.HasSuffix(name, r.suffix) {
			// a tail file
			continue
		}

		log.Debug().Str("file", name).Msg("rotating file")
		if err := r.Rotate(path); err != nil {
			log.Error().Str("file", name).Err(err).Msg("error while rotating file")
		}
	}

	return nil
}
