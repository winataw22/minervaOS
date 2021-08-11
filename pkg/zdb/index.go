package zdb

import (
	"encoding/binary"
	"io"
)

// IndexMode represens the mode in which the 0-db is running
// Adapted from https://github.com/threefoldtech/0-db/blob/development/libzdb/index.h#L4
type IndexMode uint8

// Enum values for IndexMode
const (
	IndexModeKeyValue    IndexMode = 0
	IndexModeSequential  IndexMode = 1
	IndexModeDirectKey   IndexMode = 2
	IndexModeDirectBlock IndexMode = 3
)

func (i IndexMode) String() string {
	switch i {
	case IndexModeKeyValue:
		return "key-value"
	case IndexModeSequential:
		return "sequential"
	case IndexModeDirectKey:
		return "direct-key"
	case IndexModeDirectBlock:
		return "direct-block"
	}
	return "unknown"
}

// IndexHeader is the structure contains information about an index
// adapted from https://github.com/threefoldtech/0-db/blob/development/libzdb/index.h#L31
type IndexHeader struct {
	Magic   [4]byte   // four bytes magic bytes to recognize the file
	Version uint32    // file version, for possible upgrade compatibility
	Created uint64    // unix timestamp of creation time
	Opened  uint64    // unix timestamp of last opened time
	Fileid  uint16    // current index file id (sync with dataid)
	Mode    IndexMode // running mode when index was create
}

// ReadIndex reads index header
func ReadIndex(r io.Reader) (header IndexHeader, err error) {
	err = binary.Read(r, binary.LittleEndian, &header)
	return
}
