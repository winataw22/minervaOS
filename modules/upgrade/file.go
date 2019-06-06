package upgrade

import (
	"os"
)

var ModeExecutable os.FileMode = 0111

func IsExecutable(perm os.FileMode) bool {
	return (perm & ModeExecutable) != 0
}
