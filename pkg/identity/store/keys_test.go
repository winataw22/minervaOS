package store

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"golang.org/x/crypto/ed25519"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerialize(t *testing.T) {
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	store := NewFileStore(f.Name())
	err = store.Set(sk)
	assert.NoError(t, err)

	loaded, err := store.Get()
	assert.NoError(t, err)

	assert.Equal(t, sk, loaded)
}

func TestLoadSeed110(t *testing.T) {
	seedfilecontent := `"1.1.0"{"mnemonic":"crop orient animal script safe inquiry neglect tumble maple board degree you intact busy birth west crack cabin lizard embark seed adjust around talk"}`
	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	_, err = f.WriteString(seedfilecontent)
	require.NoError(t, err)
	f.Close()

	store := NewFileStore(f.Name())
	sk, err := store.Get()
	require.NoError(t, err)
	assert.NotNil(t, sk)
	assert.Equal(t, len(sk), ed25519.PrivateKeySize)
}

func TestLoadSeed100(t *testing.T) {

	seedfilebase64 := `IjEuMC4wIkiwBpAxl8Xpc4fgQ4Wq3Is5ssEkObXDJANf7KoOw153` // matches `"1.0.0"H°1ÅésàCªÜ9²Á$9µÃ$_ìªÃ^w`
	seedfilebytes, err := base64.StdEncoding.DecodeString(seedfilebase64)
	require.NoError(t, err)

	f, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	_, err = f.Write(seedfilebytes)
	require.NoError(t, err)
	f.Close()

	store := NewFileStore(f.Name())
	sk, err := store.Get()
	require.NoError(t, err)
	assert.NotNil(t, sk)
	assert.Equal(t, len(sk), ed25519.PrivateKeySize)
}
