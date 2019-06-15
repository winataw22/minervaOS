package upgrade

import (
	"os"
	"path/filepath"
)

var modeExecutable os.FileMode = 0100

func isExecutable(perm os.FileMode) bool {
	return (perm & modeExecutable) != 0
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func listDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
