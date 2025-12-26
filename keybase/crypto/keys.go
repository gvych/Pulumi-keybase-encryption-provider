package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/keybase/saltpack"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// BoxPublicKey wraps saltpack.BoxPublicKey for easier usage
type BoxPublicKey interface {
	saltpack.BoxPublicKey
}

// BoxSecretKey wraps saltpack.BoxSecretKey for easier usage
type BoxSecretKey interface {
	saltpack.BoxSecretKey
}

// KeyPair represents a public/private key pair
type KeyPair struct {
	PublicKey  saltpack.BoxPublicKey
	SecretKey  saltpack.BoxSecretKey
	Identifier []byte
}

// SimpleKeyring is a basic implementation of saltpack.Keyring
// It holds a set of secret keys for decryption and public keys for verification
type SimpleKeyring struct {
	secretKeys map[string]saltpack.BoxSecretKey
	publicKeys map[string]saltpack.BoxPublicKey
}

// NewSimpleKeyring creates a new SimpleKeyring
func NewSimpleKeyring() *SimpleKeyring {
	return &SimpleKeyring{
		secretKeys: make(map[string]saltpack.BoxSecretKey),
		publicKeys: make(map[string]saltpack.BoxPublicKey),
	}
}

// AddKey adds a secret key to the keyring
func (k *SimpleKeyring) AddKey(secretKey saltpack.BoxSecretKey) {
	if secretKey == nil {
		return
	}
	
	publicKey := secretKey.GetPublicKey()
	keyID := keyToString(publicKey.ToKID())
	k.secretKeys[keyID] = secretKey
	k.publicKeys[keyID] = publicKey
}

// AddPublicKey adds a public key to the keyring (for sender verification)
func (k *SimpleKeyring) AddPublicKey(publicKey saltpack.BoxPublicKey) {
	if publicKey == nil {
		return
	}
	
	keyID := keyToString(publicKey.ToKID())
	k.publicKeys[keyID] = publicKey
}

// AddKeyPair adds a key pair to the keyring (using the secret key)
func (k *SimpleKeyring) AddKeyPair(keyPair *KeyPair) {
	if keyPair != nil && keyPair.SecretKey != nil {
		k.AddKey(keyPair.SecretKey)
	}
}

// LookupBoxSecretKey implements saltpack.Keyring interface
// It finds the secret key corresponding to the given key identifier
func (k *SimpleKeyring) LookupBoxSecretKey(kids [][]byte) (int, saltpack.BoxSecretKey) {
	for i, kid := range kids {
		keyID := keyToString(kid)
		if secretKey, ok := k.secretKeys[keyID]; ok {
			return i, secretKey
		}
	}
	return -1, nil
}

// LookupBoxPublicKey implements saltpack.Keyring interface
// Returns the public key for a given key identifier
func (k *SimpleKeyring) LookupBoxPublicKey(kid []byte) saltpack.BoxPublicKey {
	keyID := keyToString(kid)
	// Check if we have it as a public key
	if publicKey, ok := k.publicKeys[keyID]; ok {
		return publicKey
	}
	// Otherwise check if we have it as a secret key and derive the public key
	if secretKey, ok := k.secretKeys[keyID]; ok {
		return secretKey.GetPublicKey()
	}
	return nil
}

// ImportBoxSecretKey implements saltpack.Keyring interface
// Imports a secret key from bytes
func (k *SimpleKeyring) ImportBoxSecretKey(keyBytes []byte) saltpack.BoxSecretKey {
	if len(keyBytes) != 32 {
		return nil
	}
	
	var keyArray [32]byte
	copy(keyArray[:], keyBytes)
	
	return &naclBoxSecretKey{key: keyArray}
}

// GetAllBoxSecretKeys implements saltpack.Keyring interface
// Returns all secret keys in the keyring
func (k *SimpleKeyring) GetAllBoxSecretKeys() []saltpack.BoxSecretKey {
	var secretKeys []saltpack.BoxSecretKey
	for _, secretKey := range k.secretKeys {
		secretKeys = append(secretKeys, secretKey)
	}
	return secretKeys
}

// ImportBoxEphemeralKey implements saltpack.Keyring interface
// Imports an ephemeral public key
func (k *SimpleKeyring) ImportBoxEphemeralKey(kid []byte) saltpack.BoxPublicKey {
	if len(kid) != 32 {
		return nil
	}
	
	var keyArray [32]byte
	copy(keyArray[:], kid)
	
	return &naclBoxPublicKey{key: keyArray}
}

// CreateEphemeralKey implements saltpack.EphemeralKeyCreator interface
// Creates a new ephemeral key pair
func (k *SimpleKeyring) CreateEphemeralKey() (saltpack.BoxSecretKey, error) {
	// Generate a new key pair
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	
	publicKey := &naclBoxPublicKey{key: *pub}
	secretKey := &naclBoxSecretKey{
		key:       *priv,
		publicKey: publicKey,
	}
	
	return secretKey, nil
}

// naclBoxPublicKey implements saltpack.BoxPublicKey using NaCl box
type naclBoxPublicKey struct {
	key [32]byte
}

// CreatePublicKey creates a public key from raw bytes
func CreatePublicKey(keyBytes []byte) (saltpack.BoxPublicKey, error) {
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("public key must be 32 bytes, got %d", len(keyBytes))
	}
	
	var keyArray [32]byte
	copy(keyArray[:], keyBytes)
	
	return &naclBoxPublicKey{key: keyArray}, nil
}

// CreatePublicKeyFromHex creates a public key from a hex-encoded string
func CreatePublicKeyFromHex(hexKey string) (saltpack.BoxPublicKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}
	
	return CreatePublicKey(keyBytes)
}

func (k *naclBoxPublicKey) ToKID() []byte {
	return k.key[:]
}

func (k *naclBoxPublicKey) ToRawBoxKeyPointer() *saltpack.RawBoxKey {
	return (*saltpack.RawBoxKey)(&k.key)
}

func (k *naclBoxPublicKey) CreateEphemeralKey() (saltpack.BoxSecretKey, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &naclBoxSecretKey{
		key:       *priv,
		publicKey: &naclBoxPublicKey{key: *pub},
	}, nil
}

func (k *naclBoxPublicKey) HideIdentity() bool {
	return false
}

// naclBoxSecretKey implements saltpack.BoxSecretKey using NaCl box
type naclBoxSecretKey struct {
	key       [32]byte
	publicKey *naclBoxPublicKey
}

// CreateSecretKey creates a secret key from raw bytes
func CreateSecretKey(keyBytes []byte) (saltpack.BoxSecretKey, error) {
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("secret key must be 32 bytes, got %d", len(keyBytes))
	}
	
	var keyArray [32]byte
	copy(keyArray[:], keyBytes)
	
	// Derive public key from secret key
	var publicKeyArray [32]byte
	curve25519ScalarBaseMult(&publicKeyArray, &keyArray)
	
	return &naclBoxSecretKey{
		key:       keyArray,
		publicKey: &naclBoxPublicKey{key: publicKeyArray},
	}, nil
}

// CreateSecretKeyFromHex creates a secret key from a hex-encoded string
func CreateSecretKeyFromHex(hexKey string) (saltpack.BoxSecretKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}
	
	return CreateSecretKey(keyBytes)
}

func (k *naclBoxSecretKey) GetPublicKey() saltpack.BoxPublicKey {
	if k.publicKey == nil {
		// Derive public key if not set
		var publicKeyArray [32]byte
		curve25519ScalarBaseMult(&publicKeyArray, &k.key)
		k.publicKey = &naclBoxPublicKey{key: publicKeyArray}
	}
	return k.publicKey
}

func (k *naclBoxSecretKey) ToRawBoxKeyPointer() *saltpack.RawBoxKey {
	return (*saltpack.RawBoxKey)(&k.key)
}

func (k *naclBoxSecretKey) Precompute(publicKey saltpack.BoxPublicKey) saltpack.BoxPrecomputedSharedKey {
	var sharedKey [32]byte
	box.Precompute(&sharedKey, (*[32]byte)(publicKey.ToRawBoxKeyPointer()), &k.key)
	return &naclBoxPrecomputedSharedKey{key: sharedKey}
}

func (k *naclBoxSecretKey) Box(receiver saltpack.BoxPublicKey, nonce saltpack.Nonce, msg []byte) []byte {
	noncePtr := (*[24]byte)(&nonce)
	return box.Seal(nil, msg, noncePtr, (*[32]byte)(receiver.ToRawBoxKeyPointer()), &k.key)
}

func (k *naclBoxSecretKey) Unbox(sender saltpack.BoxPublicKey, nonce saltpack.Nonce, msg []byte) ([]byte, error) {
	noncePtr := (*[24]byte)(&nonce)
	out, ok := box.Open(nil, msg, noncePtr, (*[32]byte)(sender.ToRawBoxKeyPointer()), &k.key)
	if !ok {
		return nil, fmt.Errorf("unbox failed")
	}
	return out, nil
}

// naclBoxPrecomputedSharedKey implements saltpack.BoxPrecomputedSharedKey
type naclBoxPrecomputedSharedKey struct {
	key [32]byte
}

func (k *naclBoxPrecomputedSharedKey) ToRawBoxKeyPointer() *saltpack.RawBoxKey {
	return (*saltpack.RawBoxKey)(&k.key)
}

func (k *naclBoxPrecomputedSharedKey) Unbox(nonce saltpack.Nonce, msg []byte) ([]byte, error) {
	noncePtr := (*[24]byte)(&nonce)
	out, ok := box.OpenAfterPrecomputation(nil, msg, noncePtr, (*[32]byte)(&k.key))
	if !ok {
		return nil, fmt.Errorf("unbox failed")
	}
	return out, nil
}

func (k *naclBoxPrecomputedSharedKey) Box(nonce saltpack.Nonce, msg []byte) []byte {
	noncePtr := (*[24]byte)(&nonce)
	return box.SealAfterPrecomputation(nil, msg, noncePtr, (*[32]byte)(&k.key))
}

// GenerateKeyPair generates a new random key pair
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	
	publicKey := &naclBoxPublicKey{key: *pub}
	secretKey := &naclBoxSecretKey{key: *priv, publicKey: publicKey}
	
	return &KeyPair{
		PublicKey:  publicKey,
		SecretKey:  secretKey,
		Identifier: pub[:],
	}, nil
}

// ParseKeybasePublicKey attempts to parse a public key from Keybase API format
// The Keybase API returns PGP public keys, but for Saltpack we need NaCl keys
// 
// Note: This is a simplified parser. In production, you would need to:
// 1. Parse the PGP key bundle properly
// 2. Extract the Curve25519 encryption subkey (if available)
// 3. Handle different PGP key types
//
// For now, this returns an error indicating that key conversion is needed
func ParseKeybasePublicKey(pgpKeyBundle string) (saltpack.BoxPublicKey, error) {
	// Check if it's a PGP key
	if strings.Contains(pgpKeyBundle, "BEGIN PGP") {
		return nil, fmt.Errorf("PGP key conversion not yet implemented; Keybase API returns PGP keys but Saltpack requires NaCl/Curve25519 keys")
	}
	
	// If it's already a raw key in hex format, parse it
	if len(pgpKeyBundle) == 64 { // 32 bytes in hex = 64 chars
		return CreatePublicKeyFromHex(pgpKeyBundle)
	}
	
	return nil, fmt.Errorf("unsupported key format")
}

// ParseKeybaseKeyID attempts to parse a key ID from Keybase format
// Keybase key IDs (KIDs) are hex-encoded strings
func ParseKeybaseKeyID(kid string) ([]byte, error) {
	// Remove any prefix
	kid = strings.TrimPrefix(kid, "0x")
	
	// Decode hex
	keyID, err := hex.DecodeString(kid)
	if err != nil {
		return nil, fmt.Errorf("invalid key ID format: %w", err)
	}
	
	return keyID, nil
}

// keyToString converts a key identifier to a string for map lookups
func keyToString(kid []byte) string {
	return hex.EncodeToString(kid)
}

// curve25519ScalarBaseMult computes the Curve25519 base point scalar multiplication
// This derives the public key from a secret key using the Curve25519 elliptic curve
func curve25519ScalarBaseMult(publicKey, secretKey *[32]byte) {
	// Use the proper curve25519 scalar base multiplication
	// This computes publicKey = secretKey * G where G is the base point
	curve25519.ScalarBaseMult(publicKey, secretKey)
}

// ValidatePublicKey checks if a public key is valid
func ValidatePublicKey(key saltpack.BoxPublicKey) error {
	if key == nil {
		return fmt.Errorf("public key is nil")
	}
	
	kid := key.ToKID()
	if len(kid) != 32 {
		return fmt.Errorf("invalid public key length: expected 32 bytes, got %d", len(kid))
	}
	
	// Check for all-zero key (invalid)
	allZero := true
	for _, b := range kid {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return fmt.Errorf("public key cannot be all zeros")
	}
	
	return nil
}

// ValidateSecretKey checks if a secret key is valid
func ValidateSecretKey(key saltpack.BoxSecretKey) error {
	if key == nil {
		return fmt.Errorf("secret key is nil")
	}
	
	// Check if we can export the key bytes for our implementation
	keyBytes := ExportSecretKeyBytes(key)
	if keyBytes != nil {
		// Check for all-zero secret key (invalid)
		allZero := true
		for _, b := range keyBytes {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			return fmt.Errorf("secret key cannot be all zeros")
		}
	}
	
	// Get the public key to verify the secret key is valid
	pubKey := key.GetPublicKey()
	if pubKey == nil {
		return fmt.Errorf("secret key has no public key")
	}
	
	// Validate the public key
	return ValidatePublicKey(pubKey)
}

// KeysEqual checks if two public keys are equal
func KeysEqual(k1, k2 saltpack.BoxPublicKey) bool {
	if k1 == nil || k2 == nil {
		return k1 == k2
	}
	
	kid1 := k1.ToKID()
	kid2 := k2.ToKID()
	
	return bytes.Equal(kid1, kid2)
}

// ExportSecretKeyBytes exports the raw bytes of a secret key
// This is useful for serialization and testing
// Returns nil if the key cannot be exported
func ExportSecretKeyBytes(key saltpack.BoxSecretKey) []byte {
	if key == nil {
		return nil
	}
	
	// Try to cast to our implementation
	if naclKey, ok := key.(*naclBoxSecretKey); ok {
		return naclKey.key[:]
	}
	
	// For other implementations, we can't export the key
	return nil
}
