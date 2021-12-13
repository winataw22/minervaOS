package zos

import (
	"crypto/aes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type EncryptionAlgorithm string
type EncryptionKey []byte

func (k EncryptionKey) Valid() error {
	if len(k) != 32 {
		return aes.KeySizeError(len(k))
	}

	return nil
}

func (k EncryptionKey) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(k)), nil
}

func (k *EncryptionKey) UnmarshalText(data []byte) error {
	b, err := hex.DecodeString(string(data))
	if err != nil {
		return err
	}
	*k = b
	return nil
}

type Encryption struct {
	Algorithm EncryptionAlgorithm `json:"algorithm" toml:"algorithm"`
	Key       EncryptionKey       `json:"key" toml:"key"`
}

func (c *Encryption) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", c.Algorithm); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%x", c.Key); err != nil {
		return err
	}
	return nil
}

type ZdbBackend struct {
	Address   string `json:"address" toml:"address"`
	Namespace string `json:"namespace" toml:"namespace"`
	Password  string `json:"password" toml:"password"`
}

func (z *ZdbBackend) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", z.Address); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", z.Namespace); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", z.Password); err != nil {
		return err
	}

	return nil
}

type QuantumSafeConfig struct {
	Prefix     string       `json:"prefix" toml:"prefix"`
	Encryption Encryption   `json:"encryption" toml:"encryption"`
	Backends   []ZdbBackend `json:"backends" toml:"backends"`
}

func (m *QuantumSafeConfig) Challenge(w io.Writer) error {

	if _, err := fmt.Fprintf(w, "%s", m.Prefix); err != nil {
		return err
	}

	if err := m.Encryption.Challenge(w); err != nil {
		return err
	}

	for _, be := range m.Backends {
		if err := be.Challenge(w); err != nil {
			return err
		}
	}
	return nil
}

// TODO: fix challenge (and validation?)
type QuantumSafeMeta struct {
	Type   string            `json:"type" toml:"type"`
	Config QuantumSafeConfig `json:"config" toml:"config"`
}

func (m *QuantumSafeMeta) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", m.Type); err != nil {
		return err
	}

	if err := m.Config.Challenge(w); err != nil {
		return err
	}

	return nil
}

type ZdbGroup struct {
	Backends []ZdbBackend `json:"backends" toml:"backends"`
}

func (z *ZdbGroup) Challenge(w io.Writer) error {
	for _, be := range z.Backends {
		if err := be.Challenge(w); err != nil {
			return err
		}
	}
	return nil
}

type QuantumCompression struct {
	Algorithm string `json:"algorithm" toml:"algorithm"`
}

func (c *QuantumCompression) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", c.Algorithm); err != nil {
		return err
	}
	return nil
}

type QuantumSafeFSConfig struct {
	MinimalShards     uint32             `json:"minimal_shards" toml:"minimal_shards"`
	ExpectedShards    uint32             `json:"expected_shards" toml:"expected_shards"`
	RedundantGroups   uint32             `json:"redundant_groups" toml:"redundant_groups"`
	RedundantNodes    uint32             `json:"redundant_nodes" toml:"redundant_nodes"`
	MaxZDBDataDirSize uint32             `json:"max_zdb_data_dir_size" toml:"max_zdb_data_dir_size"`
	Encryption        Encryption         `json:"encryption" toml:"encryption"`
	Meta              QuantumSafeMeta    `json:"meta" toml:"meta"`
	Groups            []ZdbGroup         `json:"groups" toml:"groups"`
	Compression       QuantumCompression `json:"compression" toml:"compression"`
}

func (c *QuantumSafeFSConfig) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", c.MinimalShards); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", c.ExpectedShards); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", c.RedundantGroups); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", c.RedundantNodes); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", c.MaxZDBDataDirSize); err != nil {
		return err
	}

	if err := c.Encryption.Challenge(w); err != nil {
		return err
	}

	if err := c.Meta.Challenge(w); err != nil {
		return err
	}

	for _, g := range c.Groups {
		if err := g.Challenge(w); err != nil {
			return err
		}
	}

	if err := c.Compression.Challenge(w); err != nil {
		return err
	}

	return nil
}

type QuantumSafeFS struct {
	Cache  gridtypes.Unit      `json:"cache"`
	Config QuantumSafeFSConfig `json:"config"`
}

func (q QuantumSafeFS) Valid(getter gridtypes.WorkloadGetter) error {
	if q.Config.MinimalShards > q.Config.ExpectedShards {
		return fmt.Errorf("minimal shards can't be greater than expected shards")
	}
	return nil
}

func (q QuantumSafeFS) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", q.Cache); err != nil {
		return err
	}

	if err := q.Config.Challenge(w); err != nil {
		return err
	}

	return nil
}

func (q QuantumSafeFS) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{
		CRU: 1,
		MRU: 1 * gridtypes.Gigabyte,
		SRU: q.Cache, // is it HRU or SRU?
	}, nil
}

type QuatumSafeFSResult struct {
	Path            string `json:"path"`
	MetricsEndpoint string `json:"metrics_endpoint"`
}
