package zdb

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Prefix is a string used as prefix in the filesystem volume used
// to storge 0-db namespaces
const (
	Prefix = "zdb"
)

var (
	nRegex = regexp.MustCompile(`^[\w-]+$`)
)

// Index represent a part of a disk that is reserved to store 0-db data
type Index struct {
	root string
}

// NewIndex creates an Index with root. the root is a directory
// which has both 'index' and 'data' folders under.
func NewIndex(root string) Index {
	return Index{root}
}

// NSInfo is a struct containing information about a 0-db namespace
type NSInfo struct {
	Name string
	Size gridtypes.Unit
}

func (p *Index) index() string {
	return filepath.Join(p.root, "index")
}

func (p *Index) data() string {
	return filepath.Join(p.root, "data")
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

	header, err := ReadHeaderV2(f)
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

	path := filepath.Join(p.index(), name, "i0")

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

func IsZDBVersion1(ctx context.Context, root string) (bool, error) {
	// detect if we have files created by v1
	searchPath := filepath.Join(root, "index")
	_, err := os.Stat(searchPath)
	if os.IsNotExist(err) {
		// empty root, so this can be anything
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to check index directory")
	}

	found, err := exec.CommandContext(ctx,
		"find", searchPath,
		"-name", "zdb-index-*",
		"-print",
		"-quit",
	).CombinedOutput()
	if err != nil {
		return false, errors.Wrap(err, "failed to detect version")
	}
	// nothing was found
	if len(found) == 0 {
		return false, nil
	}

	return true, nil
}
