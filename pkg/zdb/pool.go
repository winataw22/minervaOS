package zdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Prefix is a string used as prefix in the filesystem volume used
// to storge 0-db namespaces
const Prefix = "zdb"

var (
	nRegex = regexp.MustCompile(`^[\w-]+$`)
)

// Index represent a part of a disk that is reserved to store 0-db data
type Index struct {
	path string
}

// NewIndex creates a ZDBPool with path. the path should point
// where both 'index' and 'data' folders exists.
func NewIndex(path string) Index {
	return Index{path}
}

// NSInfo is a struct containing information about a 0-db namespace
type NSInfo struct {
	Name string
	Size gridtypes.Unit
}

func (p *Index) index() string {
	return filepath.Join(p.path, "index")
}

func (p *Index) data() string {
	return filepath.Join(p.path, "data")
}

func (p *Index) valid(n string) error {
	if len(n) == 0 {
		return fmt.Errorf("invalid name can't be empty")
	}

	if !nRegex.MatchString(n) {
		return fmt.Errorf("name contains invalid characters")
	}

	return nil
}

// Reserved return the amount of storage that has been reserved by all the
// namespace in the pool
func (p *Index) Reserved() (uint64, error) {
	ns, err := p.Namespaces()
	if err != nil {
		return 0, err
	}

	var total uint64
	for _, n := range ns {
		total += uint64(n.Size)
	}

	return total, nil
}

// Create a namespace. Note that this create only reserve the name
// space size (and create namespace descriptor) this must be followed
// by an actual zdb NSNEW call to create the database files.
func (p *Index) Create(name, password string, size gridtypes.Unit) error {
	if err := p.valid(name); err != nil {
		return err
	}

	dir := filepath.Join(p.index(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrapf(err, "namespace '%s' directory creation failed", dir)
	}
	path := filepath.Join(dir, "zdb-namespace")
	writer, err := os.Create(path)
	if err != nil {
		return err
	}
	defer writer.Close()

	return WriteHeader(writer, Header{
		Name:     name,
		Password: password,
		MaxSize:  size,
	})
}

// Namespace gets a namespace info from pool
func (p *Index) Namespace(name string) (info NSInfo, err error) {
	if err := p.valid(name); err != nil {
		return NSInfo{}, err
	}

	path := filepath.Join(p.index(), name, "zdb-namespace")
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}

	defer f.Close()
	header, err := ReadHeader(f)
	if err != nil {
		return info, errors.Wrapf(err, "failed to read namespace header at %s", path)
	}

	info = NSInfo{
		Name: name,
		Size: header.MaxSize,
	}

	return
}

// Namespaces returns a list of NSinfo of all the namespace present in the pool
func (p *Index) Namespaces() ([]NSInfo, error) {
	dirs, err := ioutil.ReadDir(p.index())
	if err != nil {
		return nil, err
	}

	var namespaces []NSInfo
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		if dir.Name() == "default" {
			continue
		}

		info, err := p.Namespace(dir.Name())
		if os.IsNotExist(err) {
			// not a valid namespace directory
			continue
		} else if err != nil {
			return nil, err
		}

		namespaces = append(namespaces, info)
	}

	return namespaces, nil
}

// Exists checks if a namespace exists in the pool or not
// this method is way faster then using Namespaces cause it doesn't have to read any data
func (p *Index) Exists(name string) bool {
	path := filepath.Join(p.index(), name, "zdb-namespace")

	_, err := os.Stat(path)
	return err == nil
}

// IndexMode return the mode of the index of the namespace called name
func (p *Index) IndexMode(name string) (mode IndexMode, err error) {
	if err := p.valid(name); err != nil {
		return IndexMode(0), err
	}

	path := filepath.Join(p.index(), name, "zdb-index-00000")

	f, err := os.Open(path)
	if err != nil {
		return mode, err
	}

	defer f.Close()
	index, err := ReadIndex(f)
	if err != nil {
		return mode, errors.Wrapf(err, "failed to read namespace index header at %s", path)
	}

	return index.Mode, nil
}

// Delete both the data and an index of a namespace
func (p *Index) Delete(name string) error {
	if err := p.valid(name); err != nil {
		return err
	}
	idx := filepath.Join(p.index(), name)
	dat := filepath.Join(p.data(), name)

	for _, dir := range []string{idx, dat} {
		if err := os.RemoveAll(dir); err != nil {
			return errors.Wrap(err, "failed to delete directory")
		}
	}

	return nil
}
