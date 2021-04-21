package primitives

import (
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// FilesystemName return a string to be used as filesystem name from
// a reservation object
func FilesystemName(wl *gridtypes.WorkloadWithID) string {
	return wl.ID.String()
}
