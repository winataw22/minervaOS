package set

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
)

// ErrConflict is return when trying to add a port
// in the set that is already present
type ErrConflict struct {
	Port uint
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("port %d is already in the set", e.Port)
}

// UintSet is a set containing uint
type UintSet struct {
	sync.RWMutex
	root string
}

// NewUint creates a new set for uint
// path if the directory where to store the set on disk
// the directory pointed by path must exists already
func NewUint(path string) *UintSet {
	return &UintSet{
		root: path,
	}
}

func (p *UintSet) path(i uint) string {
	return filepath.Join(p.root, fmt.Sprintf("%d", i))
}

// Add tries to add port to the set. If port is already
// present errPortConflict is return otherwise nil is returned
func (p *UintSet) Add(i uint) error {
	p.Lock()
	defer p.Unlock()

	f, err := os.OpenFile(p.path(i), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0440)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrConflict{Port: i}
		}
		return err
	}

	f.Close()
	return nil
}

// Remove removes a port from the set
// removes never fails cause if the port is not in the set
// remove is a nop-op
func (p *UintSet) Remove(i uint) {
	p.Lock()
	defer p.Unlock()

	_ = os.RemoveAll(p.path(i))
}

// List returns a list of uint present in the set
func (p *UintSet) List() ([]uint, error) {
	p.RLock()
	defer p.RUnlock()

	infos, err := ioutil.ReadDir(p.root)
	if err != nil {
		return nil, err
	}
	l := make([]uint, 0, len(infos))

	for _, info := range infos {
		s := filepath.Base(info.Name())
		i, err := strconv.Atoi(s)
		if err != nil {
			log.Warn().Err(err).Msg("file with wrong formatted name found in reserved wireguard port cache")
			continue
		}
		l = append(l, uint(i))
	}
	return l, nil
}
