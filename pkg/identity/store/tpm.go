package store

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg/identity/store/tpm"
)

const (
	keyAddress = tpm.Address(0x81000000)
)

var (
	selector = tpm.PCRSelector{
		tpm.SHA1: []int{0, 1, 2}, // what values should we use for PCRs
	}
)

type TPMStore struct{}

var _ Store = (*TPMStore)(nil)

func NewTPM() *TPMStore {
	return &TPMStore{}
}

func (f *TPMStore) Kind() string {
	return "tpm-store"
}

// Get returns the key from the store
func (t *TPMStore) Get() (ed25519.PrivateKey, error) {
	exists, err := t.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrKeyDoesNotExist
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	handler, err := tpm.Unseal(ctx, keyAddress, selector)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = handler.Delete()
	}()

	seed, err := handler.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read seed: %w", err)
	}

	return keyFromSeed(seed)
}

// Updates, or overrides the current key
func (t *TPMStore) Set(key ed25519.PrivateKey) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	primary, err := tpm.CreatePrimary(ctx, tpm.SHA1, tpm.RSA)
	if err != nil {
		return fmt.Errorf("failed to create primary key: %w", err)
	}

	defer func() {
		_ = primary.Delete()
	}()

	policy, err := tpm.CreatePCRPolicy(ctx, selector)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	defer func() {
		_ = policy.Delete()
	}()

	// we store the key seed, not the seed itself
	object, err := tpm.Create(ctx, tpm.SHA256, bytes.NewBuffer(key.Seed()), primary, policy)
	if err != nil {
		return err
	}

	defer func() {
		_ = object.Delete()
	}()

	loaded, err := tpm.Load(ctx, primary, object)
	if err != nil {
		return fmt.Errorf("failed to load object: %w", err)
	}

	defer func() {
		_ = loaded.Delete()
	}()

	_ = tpm.EvictControl(ctx, nil, keyAddress)

	if err := tpm.EvictControl(ctx, &loaded, keyAddress); err != nil {
		return fmt.Errorf("failed to evict the key: %w", err)
	}

	return nil
}

// Check if key there is a key stored in the
// store
func (t *TPMStore) Exists() (bool, error) {
	handlers, err := tpm.PersistedHandlers(context.Background())
	if err != nil {
		return false, err
	}

	for _, handler := range handlers {
		if keyAddress == handler {
			return true, nil
		}
	}

	return false, nil
}

// Destroys the key
func (t *TPMStore) Annihilate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return tpm.EvictControl(ctx, nil, keyAddress)
}

func IsTPMEnabled() bool {
	return tpm.IsTPMEnabled(context.Background())
}
