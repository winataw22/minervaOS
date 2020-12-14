package identity

import (
	"io/ioutil"
	"net/url"
	"os"
	"testing"

	"encoding/base64"

	"golang.org/x/crypto/ed25519"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	keypair, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotNil(t, keypair.PrivateKey)
	assert.NotNil(t, keypair.PublicKey)
}

func TestSerialize(t *testing.T) {
	keypair, err := GenerateKeyPair()
	require.NoError(t, err)

	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	err = keypair.Save(f.Name())
	require.NoError(t, err)

	keypair2, err := LoadKeyPair(f.Name())
	require.NoError(t, err)

	assert.Equal(t, keypair.PrivateKey, keypair2.PrivateKey)
	assert.Equal(t, keypair.PublicKey, keypair2.PublicKey)
}

func TestIdentity(t *testing.T) {
	sk := ed25519.NewKeyFromSeed([]byte("helloworldhelloworldhelloworld12"))
	keypair := KeyPair{
		PrivateKey: sk,
		PublicKey:  sk.Public().(ed25519.PublicKey),
	}
	id := keypair.Identity()
	assert.Equal(t, "FkUfMueBVSK6V1DCHVAtzzaqPqCPVzGguDzCQxq7Ep85", id)
	assert.Equal(t, "FkUfMueBVSK6V1DCHVAtzzaqPqCPVzGguDzCQxq7Ep85", url.PathEscape(id), "identity should be url friendly")
}

func TestLoadSeed110(t *testing.T) {
	seedfilecontent := `"1.1.0"{"mnemonic":"crop orient animal script safe inquiry neglect tumble maple board degree you intact busy birth west crack cabin lizard embark seed adjust around talk"}`
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	_, err = f.WriteString(seedfilecontent)
	require.NoError(t, err)
	s, err := LoadSeed(f.Name())
	require.NoError(t, err)
	assert.Equal(t, len(s), ed25519.SeedSize)

}

func TestLoadSeed100(t *testing.T) {

	seedfilebase64 := `IjEuMC4wIkiwBpAxl8Xpc4fgQ4Wq3Is5ssEkObXDJANf7KoOw153` // matches `"1.0.0"H°1ÅésàCªÜ9²Á$9µÃ$_ìªÃ^w`
	seedfilebytes, err := base64.StdEncoding.DecodeString(seedfilebase64)
	require.NoError(t, err)

	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	_, err = f.Write(seedfilebytes)
	require.NoError(t, err)
	s, err := LoadSeed(f.Name())
	require.NoError(t, err)
	assert.Equal(t, len(s), ed25519.SeedSize)

}
