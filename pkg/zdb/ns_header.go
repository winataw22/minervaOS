package zdb

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	nsFlagsPublic   = 1
	nsFlagsExtended = 4
)

// Header is the structure contains information about a Namespace
type baseHeader struct {
	NameLength     uint8  // length of the namespace name
	PasswordLength uint8  // length of the password
	MaxSize        uint32 // maximum datasize allowed on that namespace
	Flags          uint8
}

type extendedHeader struct {
	Version uint32
	MaxSize uint64
}

// Header is the namespace header
type Header struct {
	Version  uint32
	Name     string
	Password string
	MaxSize  gridtypes.Unit
}

// WriteHeader writes header data to writer
func WriteHeader(w io.Writer, h Header) error {

	if err := binary.Write(
		w,
		binary.LittleEndian,
		baseHeader{
			NameLength:     uint8(len(h.Name)),
			PasswordLength: uint8(len(h.Password)),
			Flags:          nsFlagsPublic | nsFlagsExtended,
		}); err != nil {
		return err
	}

	for _, str := range []string{h.Name, h.Password} {
		if _, err := io.WriteString(w, str); err != nil {
			return err
		}
	}

	if err := binary.Write(
		w,
		binary.LittleEndian,
		extendedHeader{
			Version: 1,
			MaxSize: uint64(h.MaxSize),
		}); err != nil {
		return err
	}

	return nil
}

// ReadHeader reands namespace header
func ReadHeader(r io.Reader) (header Header, err error) {
	var bh baseHeader
	if err := binary.Read(r, binary.LittleEndian, &bh); err != nil {
		return header, err
	}
	header.Version = 0
	header.MaxSize = gridtypes.Unit(bh.MaxSize)
	name := make([]byte, bh.NameLength)
	passwrd := make([]byte, bh.PasswordLength)
	// the next reads are important to advance the reader to the position
	// of the extended header
	if _, err := io.ReadAtLeast(r, name, int(bh.NameLength)); err != nil {
		return header, errors.Wrapf(err, "invalid header. bad name")
	}
	if _, err := io.ReadAtLeast(r, passwrd, int(bh.PasswordLength)); err != nil {
		return header, errors.Wrapf(err, "invalid header. bad name")
	}

	header.Name = string(name)
	header.Password = string(passwrd)

	if bh.Flags&nsFlagsExtended == 0 {
		return
	}

	var eh extendedHeader
	if err := binary.Read(r, binary.LittleEndian, &eh); err != nil {
		return header, err
	}

	header.Version = eh.Version
	header.MaxSize = gridtypes.Unit(eh.MaxSize)

	return
}
