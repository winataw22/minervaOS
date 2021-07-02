package substrate

import (
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

var (
	//ErrInvalidVersion is returned if version 4bytes is invalid
	ErrInvalidVersion = fmt.Errorf("invalid version")
	//ErrUnknownVersion is returned if version number is not supported
	ErrUnknownVersion = fmt.Errorf("unknown version")
	//ErrNotFound is returned if an object is not found
	ErrNotFound = fmt.Errorf("object not found")
)

// Versioned base for all types
type Versioned struct {
	Version uint32
}

// Substrate client
type Substrate struct {
	cl   *gsrpc.SubstrateAPI
	meta *types.Metadata
}

// NewSubstrate creates a substrate client
func NewSubstrate(url string) (*Substrate, error) {
	cl, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}
	meta, err := cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	return &Substrate{
		cl:   cl,
		meta: meta,
	}, nil
}

// Refresh reloads meta from chain!
// not thread safe
func (s *Substrate) Refresh() error {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}

	s.meta = meta
	return nil
}

func (s *Substrate) getVersion(b types.StorageDataRaw) (uint32, error) {
	var ver Versioned
	if err := types.DecodeFromBytes(b, &ver); err != nil {
		return 0, errors.Wrapf(ErrInvalidVersion, "failed to load version (reason: %s)", err)
	}

	return ver.Version, nil
}
