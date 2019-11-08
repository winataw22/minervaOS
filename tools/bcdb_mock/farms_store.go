package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

type farmStore struct {
	Farms []*directory.TfgridFarm1 `json:"farms"`
	m     sync.RWMutex
}

func loadfarmStore() (*farmStore, error) {
	store := &farmStore{
		Farms: []*directory.TfgridFarm1{},
	}
	f, err := os.OpenFile("farms.json", os.O_RDONLY, 0660)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		return store, err
	}
	return store, nil
}

func (s *farmStore) Save() error {
	s.m.RLock()
	defer s.m.RUnlock()

	f, err := os.OpenFile("farms.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *farmStore) List() []*directory.TfgridFarm1 {
	s.m.RLock()
	defer s.m.RUnlock()

	out := make([]*directory.TfgridFarm1, len(s.Farms))
	copy(out, s.Farms)
	return out
}

func (s *farmStore) GetByName(name string) (*directory.TfgridFarm1, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	for _, f := range s.Farms {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("farm %s not found", name)
}

func (s *farmStore) GetByID(id int) (*directory.TfgridFarm1, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	if id <= 0 || id > len(s.Farms) {
		return nil, fmt.Errorf("farm with id %d not found", id)
	}
	return s.Farms[id-1], nil
}

func (s *farmStore) Add(farm directory.TfgridFarm1) (pkg.FarmID, error) {
	s.m.Lock()
	defer s.m.Unlock()

	for _, f := range s.Farms {
		if f.Name == farm.Name {
			f.WalletAddresses = farm.WalletAddresses
			f.Location = farm.Location
			f.Email = farm.Email
			f.ResourcePrices = farm.ResourcePrices
			return pkg.FarmID(f.ID), nil
		}
	}

	farm.ID = uint64(len(s.Farms) + 1) // ids starts at 1
	s.Farms = append(s.Farms, &farm)
	return pkg.FarmID(farm.ID), nil
}
