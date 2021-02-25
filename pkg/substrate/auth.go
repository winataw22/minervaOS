package substrate

import (
	"crypto/ed25519"
	"fmt"
	"strconv"

	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type substrateUsers struct {
	sub Substrate
	mem *lru.Cache
}

// NewSubstrateUsers creates a substrate users db that implements the provision.Users interface.
func NewSubstrateUsers(url string) (provision.Users, error) {
	sub, err := NewSubstrate(url)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New(1024)
	if err != nil {
		return nil, err
	}

	return &substrateUsers{
		sub: sub,
		mem: cache,
	}, nil
}

func (s *substrateUsers) GetKey(id gridtypes.ID) (ed25519.PublicKey, error) {
	if value, ok := s.mem.Get(id); ok {
		return value.(ed25519.PublicKey), nil
	}

	idUint, err := strconv.ParseUint(id.String(), 10, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse user id '%s'", id.String())
	}

	user, err := s.sub.GetUser(uint32(idUint))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get user with id '%d'", idUint)
	}

	key := user.Address.PublicKey()
	s.mem.Add(id, key)
	return key, nil
}

type substrateAdmins struct {
	sub  Substrate
	twin uint32
	mem  *lru.Cache
}

// NewSubstrateAdmins creates a substrate users db that implements the provision.Users interface.
// but it also make sure the user is an admin
func NewSubstrateAdmins(url string, farmID uint32) (provision.Users, error) {
	sub, err := NewSubstrate(url)
	if err != nil {
		return nil, err
	}

	farm, err := sub.GetFarm(farmID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get farm")
	}

	cache, err := lru.New(128)
	if err != nil {
		return nil, err
	}

	return &substrateAdmins{
		sub:  sub,
		twin: uint32(farm.TwinID),
		mem:  cache,
	}, nil
}

func (s *substrateAdmins) GetKey(id gridtypes.ID) (ed25519.PublicKey, error) {
	if value, ok := s.mem.Get(id); ok {
		return value.(ed25519.PublicKey), nil
	}

	idUint, err := strconv.ParseUint(id.String(), 10, 32)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse user id '%s'", id.String())
	}

	user, err := s.sub.GetUser(uint32(idUint))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get user with id '%d'", idUint)
	}

	key := user.Address.PublicKey()

	twin, err := s.sub.GetTwin(s.twin)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get twin object")
	}
	found := false
	for _, entry := range twin.Entities {
		if entry.EntityID == types.U32(idUint) {
			found = true
			break
		}
	}

	if found {
		s.mem.Add(id, key)
		return key, nil
	}

	return nil, fmt.Errorf("user is not a twin manager")
}
