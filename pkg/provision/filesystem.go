package provision

import (
	"strconv"
	"strings"

	"github.com/threefoldtech/zos/pkg"
)

// FilesystemName return a string to be used as filesystem name from
// a reservation object
func FilesystemName(r Reservation) string {
	return r.ID
}

// WorkloadIDFromFilesystem parse a filesystem object and return the reservation ID that
// created it
func WorkloadIDFromFilesystem(fs pkg.Filesystem) (int64, error) {
	sid := strings.Split(fs.Name, "-")[0]
	return strconv.ParseInt(sid, 10, 64)
}
