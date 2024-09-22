package crypto

import (
	"crypto/rand"
	"encoding/hex"
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
	_, err := fmt.Sscanf("0bfe9e3b9ce17fe6d570b165ea2a01034326b8c81d5f2c5384c8fe886552f074ec43017465598c4f5a857b495b445be46c3df48d14878bd0b1b907", "%x", &chiper)
	require.NoError(t, err)

	decrypted, err := Decrypt([]byte(chiper), sk)
	require.NoError(t, err)

	assert.Equal(t, "hello world", string(decrypted))

	x, err := Encrypt([]byte("hello world"), sk.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	fmt.Printf("%x\n", x)
}

func TestECDH(t *testing.T) {
	alicePubkey, alicePrivkey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	bobPubkey, bobPrivkey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	msg := []byte("hello world")
	encrypted, err := EncryptECDH(msg, alicePrivkey, bobPubkey)
	require.NoError(t, err)

	msg2, err := DecryptECDH(encrypted, bobPrivkey, alicePubkey)
	require.NoError(t, err)

	assert.Equal(t, msg, msg2)
}

func TestECDHPyNACLCompatibility(t *testing.T) {
	// import nacl.public
	// import nacl.encoding
	// from nacl.secret import SecretBox
	// from nacl.bindings import crypto_scalarmult
	// from hashlib import blake2b
	// alice_private = nacl.public.PrivateKey.from_seed(b"11111111111111111111111111111111")
	// bob_private = nacl.public.PrivateKey.from_seed(b"22222222222222222222222222222222")
	// shared_secret = crypto_scalarmult(alice_private.encode(), bob_private.public_key.encode())
	// h = blake2b(shared_secret,digest_size=32)
	// key = h.digest()
	// box = SecretBox(key)
	// encrypted = box.encrypt(b'hello world')
	// print(nacl.encoding.HexEncoder().encode(encrypted))
	// b'8a246cd20d2d29b8f45d7a32e469cd914707bf3abed5747bcd9b54383e56e9be97b940df5a6826400f36a829ce10c618979ee2'

	alicePrivate := ed25519.NewKeyFromSeed([]byte("11111111111111111111111111111111"))
	bobPrivate := ed25519.NewKeyFromSeed([]byte("22222222222222222222222222222222"))

	encrypted, err := hex.DecodeString("8a246cd20d2d29b8f45d7a32e469cd914707bf3abed5747bcd9b54383e56e9be97b940df5a6826400f36a829ce10c618979ee2")
	require.NoError(t, err)

	decrypted, err := DecryptECDH(encrypted, bobPrivate, alicePrivate.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), decrypted)
}
