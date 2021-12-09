package zdb

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type internalHeaderV2 struct {
	Version        uint32 // version of the namespace descriptor
	NameLength     uint8  // length of the namespace name
	PasswordLength uint8  // length of the password
	MaxSize        uint64 // maximum datasize allowed on that namespace
	Flags          uint8
}

// Header is the namespace header
type Header struct {
	Version  uint32
	Name     string
	Password string
	MaxSize  gridtypes.Unit
}

// ReadHeaderV1 reads namespace header (V1)
func ReadHeaderV2(r io.Reader) (header Header, err error) {
	var bh internalHeaderV2
	if err := binary.Read(r, binary.LittleEndian, &bh); err != nil {
		return header, err
	}
	header.Version = bh.Version
	header.MaxSize = gridtypes.Unit(bh.MaxSize)
	name := make([]byte, bh.NameLength)
	password := make([]byte, bh.PasswordLength)
	// the next reads are important to advance the reader to the position
	// of the extended header
	if _, err := io.ReadAtLeast(r, name, int(bh.NameLength)); err != nil {
		return header, errors.Wrapf(err, "invalid header. bad name")
	}
	if _, err := io.ReadAtLeast(r, password, int(bh.PasswordLength)); err != nil {
		return header, errors.Wrapf(err, "invalid header. bad name")
	}

	header.Name = string(name)
	header.Password = string(password)

	return
}
