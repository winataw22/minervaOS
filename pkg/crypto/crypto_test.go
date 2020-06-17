package crypto

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
)

func TestPublicKeyEncryption(t *testing.T) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	msg := []byte("hello world")

	cipher, err := Encrypt(msg, pk)
	require.NoError(t, err)
	clear, err := Decrypt(cipher, sk)
	require.NoError(t, err)
	assert.Equal(t, clear, msg)
}

func TestSignature(t *testing.T) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	message := []byte("hello world")

	signature, err := Sign(sk, message)
	require.NoError(t, err)

	assert.NoError(t, Verify(pk, message, signature))

	// mess up with the signature
	signature[0] = 'a'
	signature[1] = 'b'

	assert.Error(t, Verify(pk, message, signature))
}

// TestPyNACLCompatibility test the compatibility between
// golang.org/x/crypto/ed25519 and https://github.com/pyca/pynacl/
func TestPyNACLCompatibilityKeyGeneration(t *testing.T) {
	seed := []byte("12345678901234567890123456789012")

	singingKey := ed25519.NewKeyFromSeed(seed)
	verifyKey := singingKey.Public().(ed25519.PublicKey)

	privateKey := PrivateKeyToCurve25519(singingKey)
	publicKey := PublicKeyToCurve25519(verifyKey)

	// in ed25519 library ed25519.PrivateKey is represented by the privateKey+publicKey, so we slice it to compare with
	// what pynacl produces
	assert.Equal(t, "3132333435363738393031323334353637383930313233343536373839303132", fmt.Sprintf("%x", []byte(singingKey)[:32]))
	assert.Equal(t, "2f8c6129d816cf51c374bc7f08c3e63ed156cf78aefb4a6550d97b87997977ee", fmt.Sprintf("%x", verifyKey))

	assert.Equal(t, "f0106660c3dda23f16daa9ac5b811b963077f5bc0af89f85804f0de8e424f050", fmt.Sprintf("%x", privateKey))
	assert.Equal(t, "f22aa4a9ec6ff1e591d83dc97d8cdfceea8f7cd7453281e07672d415b9637454", fmt.Sprintf("%x", publicKey))
}

func TestPyNACLCompatibilityEncryption(t *testing.T) {
	// import nacl.public
	// import nacl.encoding
	// seed = b"12345678901234567890123456789012"
	// sk = nacl.public.PrivateKey.from_seed(seed)
	// box = nacl.public.SealedBox(sk.public_key)
	// encrypted = box.encrypt(b'hello world')
	// hex_encoder = nacl.encoding.HexEncoder()
	// print(hex_encoder.encode(encrypted))
	// 0bfe9e3b9ce17fe6d570b165ea2a01034326b8c81d5f2c5384c8fe886552f074ec43017465598c4f5a857b495b445be46c3df48d14878bd0b1b907

	seed := []byte("12345678901234567890123456789012")
	sk := ed25519.NewKeyFromSeed(seed)

	chiper := ""
	fmt.Sscanf("0bfe9e3b9ce17fe6d570b165ea2a01034326b8c81d5f2c5384c8fe886552f074ec43017465598c4f5a857b495b445be46c3df48d14878bd0b1b907", "%x", &chiper)

	decrypted, err := Decrypt([]byte(chiper), sk)
	require.NoError(t, err)

	assert.Equal(t, "hello world", string(decrypted))

	x, err := Encrypt([]byte("hello world"), sk.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	fmt.Printf("%x\n", x)
}
