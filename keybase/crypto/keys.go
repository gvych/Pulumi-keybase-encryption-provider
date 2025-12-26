package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/keybase/saltpack"
	"golang.org/x/crypto/nacl/box"
)

// BoxPublicKey is a wrapper around saltpack.RawBoxKey that implements saltpack.BoxPublicKey
type BoxPublicKey struct {
	key saltpack.RawBoxKey
}

// NewBoxPublicKey creates a BoxPublicKey from a RawBoxKey
func NewBoxPublicKey(key saltpack.RawBoxKey) BoxPublicKey {
	return BoxPublicKey{key: key}
}

// ToKID returns the key ID (the key bytes themselves)
func (k BoxPublicKey) ToKID() []byte {
	return k.key[:]
}

// ToRawBoxKeyPointer returns a pointer to the underlying RawBoxKey
func (k BoxPublicKey) ToRawBoxKeyPointer() *saltpack.RawBoxKey {
	return &k.key
}

// HideIdentity returns false (we don't hide recipient identities by default)
func (k BoxPublicKey) HideIdentity() bool {
	return false
}

// CreateEphemeralKey creates a new ephemeral keypair
// This is required by the EphemeralKeyCreator interface
func (k BoxPublicKey) CreateEphemeralKey() (saltpack.BoxSecretKey, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	return &naclBoxSecretKey{publicKey: *pub, secretKey: *priv}, nil
}

// naclBoxSecretKey is a minimal implementation of saltpack.BoxSecretKey
type naclBoxSecretKey struct {
	publicKey [32]byte
	secretKey [32]byte
}

func (k *naclBoxSecretKey) Box(receiver saltpack.BoxPublicKey, nonce saltpack.Nonce, msg []byte) []byte {
	noncePtr := (*[24]byte)(nonce[:])
	receiverKey := (*[32]byte)(receiver.ToRawBoxKeyPointer())
	return box.Seal(nil, msg, noncePtr, receiverKey, &k.secretKey)
}

func (k *naclBoxSecretKey) Unbox(sender saltpack.BoxPublicKey, nonce saltpack.Nonce, msg []byte) ([]byte, error) {
	noncePtr := (*[24]byte)(nonce[:])
	senderKey := (*[32]byte)(sender.ToRawBoxKeyPointer())
	out, ok := box.Open(nil, msg, noncePtr, senderKey, &k.secretKey)
	if !ok {
		return nil, fmt.Errorf("nacl.box.Open failed")
	}
	return out, nil
}

func (k *naclBoxSecretKey) GetPublicKey() saltpack.BoxPublicKey {
	return &BoxPublicKey{key: k.publicKey}
}

func (k *naclBoxSecretKey) Precompute(peer saltpack.BoxPublicKey) saltpack.BoxPrecomputedSharedKey {
	// We don't implement precomputation for now
	return nil
}

func (k *naclBoxSecretKey) CreateEphemeralKey() (saltpack.BoxSecretKey, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	return &naclBoxSecretKey{publicKey: *pub, secretKey: *priv}, nil
}

// KeyIDPrefix is the prefix for Keybase NaCl encryption key IDs
// Format: 0120 (key type) + 64 hex chars (32 bytes)
const (
	KeyIDPrefix       = "0120"
	KeyIDHexLength    = 68 // 4 (prefix) + 64 (32 bytes as hex)
	PublicKeyByteSize = 32 // Curve25519 public key size
)

// PGPToBoxPublicKey converts a PGP public key bundle or Keybase KID to a saltpack BoxPublicKey
//
// This function handles multiple input formats:
// 1. Keybase KID (Key ID): Format "0120<64-hex-chars>" - directly extracts the key bytes
// 2. PGP public key bundle: ASCII-armored PGP key - extracts the encryption subkey
// 3. Raw hex string: 64 hex characters representing a 32-byte key
// 4. Raw binary: 32-byte slice representing the key directly
//
// The KID format is preferred as it's the most direct and reliable method.
func PGPToBoxPublicKey(keyData string, kid string) (BoxPublicKey, error) {
	// Strategy 1: Try to extract key from KID first (most reliable)
	if kid != "" {
		key, err := extractKeyFromKID(kid)
		if err == nil {
			return key, nil
		}
		// If KID parsing fails, try other methods
	}

	// Strategy 2: Check if keyData is a raw hex string (64 hex chars = 32 bytes)
	keyData = strings.TrimSpace(keyData)
	if len(keyData) == 64 {
		key, err := parseHexKey(keyData)
		if err == nil {
			return key, nil
		}
	}

	// Strategy 3: Check if it looks like a PGP key bundle
	if strings.Contains(keyData, "BEGIN PGP PUBLIC KEY") {
		return extractKeyFromPGPBundle(keyData)
	}

	// Strategy 4: Try direct binary conversion if exactly 32 bytes
	if len(keyData) == PublicKeyByteSize {
		var rawKey saltpack.RawBoxKey
		copy(rawKey[:], []byte(keyData))
		return NewBoxPublicKey(rawKey), nil
	}

	return BoxPublicKey{}, fmt.Errorf("unsupported key format: expected Keybase KID, PGP bundle, or 32-byte key")
}

// extractKeyFromKID extracts the public key bytes from a Keybase KID
// KID format: "0120" + 64 hex characters (32 bytes)
// Example: 0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567
func extractKeyFromKID(kid string) (BoxPublicKey, error) {
	kid = strings.TrimSpace(kid)

	// Validate KID format
	if len(kid) != KeyIDHexLength {
		return BoxPublicKey{}, fmt.Errorf("invalid KID length: expected %d characters, got %d", KeyIDHexLength, len(kid))
	}

	// Validate prefix
	if !strings.HasPrefix(kid, KeyIDPrefix) {
		return BoxPublicKey{}, fmt.Errorf("invalid KID prefix: expected %q, got %q", KeyIDPrefix, kid[:4])
	}

	// Extract hex-encoded key (skip the 4-character prefix)
	keyHex := kid[4:]

	// Decode hex to bytes
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return BoxPublicKey{}, fmt.Errorf("failed to decode KID hex: %w", err)
	}

	// Validate key size
	if len(keyBytes) != PublicKeyByteSize {
		return BoxPublicKey{}, fmt.Errorf("invalid key size: expected %d bytes, got %d", PublicKeyByteSize, len(keyBytes))
	}

	// Convert to BoxPublicKey (fixed-size array)
	var rawKey saltpack.RawBoxKey
	copy(rawKey[:], keyBytes)

	return NewBoxPublicKey(rawKey), nil
}

// parseHexKey parses a 64-character hex string into a BoxPublicKey
func parseHexKey(hexKey string) (BoxPublicKey, error) {
	hexKey = strings.TrimSpace(hexKey)

	if len(hexKey) != 64 {
		return BoxPublicKey{}, fmt.Errorf("invalid hex key length: expected 64 characters, got %d", len(hexKey))
	}

	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return BoxPublicKey{}, fmt.Errorf("failed to decode hex key: %w", err)
	}

	if len(keyBytes) != PublicKeyByteSize {
		return BoxPublicKey{}, fmt.Errorf("invalid key size: expected %d bytes, got %d", PublicKeyByteSize, len(keyBytes))
	}

	var rawKey saltpack.RawBoxKey
	copy(rawKey[:], keyBytes)

	return NewBoxPublicKey(rawKey), nil
}

// extractKeyFromPGPBundle extracts the Curve25519 encryption key from a PGP public key bundle
//
// Note: Full PGP parsing is complex. For now, we attempt to extract the key using heuristics.
// In production, consider using a full PGP library like golang.org/x/crypto/openpgp
//
// For Keybase keys, the preferred approach is to use the KID directly rather than parsing PGP.
func extractKeyFromPGPBundle(pgpBundle string) (BoxPublicKey, error) {
	// For Keybase, the KID is the authoritative source
	// PGP bundle parsing is provided for compatibility, but may not work for all keys
	
	// This is a placeholder implementation
	// In practice, Keybase's API provides the KID which should be used instead
	return BoxPublicKey{}, fmt.Errorf("PGP bundle parsing not yet implemented; please use the KID field from the API response")
}

// ValidatePublicKey validates that a BoxPublicKey is well-formed
func ValidatePublicKey(key BoxPublicKey) error {
	// Check if key is all zeros (invalid)
	allZeros := true
	for _, b := range key.key {
		if b != 0 {
			allZeros = false
			break
		}
	}

	if allZeros {
		return fmt.Errorf("public key is all zeros (invalid)")
	}

	// Additional validation could include:
	// - Checking if the key is on the curve
	// - Checking for known weak keys
	// For now, we just check it's not all zeros

	return nil
}

// FormatKeyID formats a BoxPublicKey as a Keybase KID string
func FormatKeyID(key BoxPublicKey) string {
	return KeyIDPrefix + hex.EncodeToString(key.key[:])
}

// PublicKeyToHex converts a BoxPublicKey to a hex string (without KID prefix)
func PublicKeyToHex(key BoxPublicKey) string {
	return hex.EncodeToString(key.key[:])
}

// ConvertUserPublicKeys converts a slice of API user public keys to BoxPublicKeys
// This is a convenience function for batch conversion
type UserPublicKey struct {
	Username  string
	PublicKey string // PGP bundle or hex
	KeyID     string // Keybase KID
}

type ConvertedKey struct {
	Username     string
	BoxPublicKey BoxPublicKey
	KeyID        string
}

// ConvertPublicKeys converts multiple user public keys to BoxPublicKeys
func ConvertPublicKeys(users []UserPublicKey) ([]ConvertedKey, error) {
	if len(users) == 0 {
		return nil, fmt.Errorf("no users provided")
	}

	var results []ConvertedKey
	var errors []string

	for _, user := range users {
		key, err := PGPToBoxPublicKey(user.PublicKey, user.KeyID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", user.Username, err))
			continue
		}

		// Validate the converted key
		if err := ValidatePublicKey(key); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", user.Username, err))
			continue
		}

		results = append(results, ConvertedKey{
			Username:     user.Username,
			BoxPublicKey: key,
			KeyID:        user.KeyID,
		})
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("failed to convert some keys: %s", strings.Join(errors, "; "))
	}

	return results, nil
}
