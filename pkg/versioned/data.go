package versioned

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

var (
	// ErrNotVersioned error is raised if the underlying reader has no version
	ErrNotVersioned = fmt.Errorf("no version information")
)

// IsNotVersioned checks if error is caused by a 'not versioned' stream
func IsNotVersioned(err error) bool {
	return errors.Cause(err) == ErrNotVersioned
}

// Reader is a versioned reader
// The Versioned Reader is a reader that can load the version of the data from a stream
// without assuming anything regarding the underlying encoding of your data object
type Reader struct {
	io.Reader
	version Version
}

// Version return the version of the data
func (r *Reader) Version() Version {
	return r.version
}

// NewVersionedReader creates a versioned reader from an un-versioned
// reader. It's usually used to unify the data migration work flow
// in case the older data file didn't have a version stamp
// example:
//  reader, err := NewReader(file)
//  if IsNotVersioned(err) {
//      file.Seek(0, 0) // this is important to make u reading from start
//      reader = NewVersionedReader(MustParse("0.0.0"), file)
//  } else err != nil {
//    // probably io error
// }
func NewVersionedReader(version Version, r io.Reader) *Reader {
	return &Reader{Reader: r, version: version}
}

// NewReader creates a new versioned reader from a stream. It fails
// if the reader can not read the version from the stream.
// On success, the reader will have a version, and then can be used
// to load the data.
func NewReader(r io.Reader) (*Reader, error) {
	dec := json.NewDecoder(r)

	// to differentiate between IO errors
	// and version parsing errors. we going to
	// json load string first. then we can
	// parse the string as a version

	var ver string
	if err := dec.Decode(&ver); err != nil {
		switch err.(type) {
		case *json.SyntaxError:
			err = ErrNotVersioned
		case *json.UnmarshalTypeError:
			err = ErrNotVersioned
		}

		if err == io.ErrUnexpectedEOF || err == io.EOF {
			err = ErrNotVersioned
		}

		return nil, err
	}

	version, err := Parse(ver)
	if err != nil {
		return nil, errors.Wrap(ErrNotVersioned, err.Error())
	}

	return &Reader{
		Reader:  io.MultiReader(dec.Buffered(), r),
		version: version,
	}, nil
}

// NewWriter creates a versioned writer that marks data with the
// Provided version.
func NewWriter(w io.Writer, version Version) (io.Writer, error) {
	data, err := json.Marshal(version)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(data)
	return w, err
}

// ReadFile content
func ReadFile(path string) (Version, []byte, error) {
	all, err := ioutil.ReadFile(path)
	if err != nil {
		return MustParse("0.0.0"), nil, err
	}

	buf := bytes.NewBuffer(all)
	reader, err := NewReader(buf)
	if err != nil {
		return MustParse("0.0.0"), all, ErrNotVersioned
	}
	data, err := ioutil.ReadAll(reader)
	return reader.Version(), data, err
}

// WriteFile versioned data to file
func WriteFile(filename string, version Version, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	defer file.Close()
	writer, err := NewWriter(file, version)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)

	return err
}
