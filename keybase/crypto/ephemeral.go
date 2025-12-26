package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/box"
)

var (
	// ErrInsufficientEntropy is returned when the system doesn't have enough entropy
	// to generate secure random keys
	ErrInsufficientEntropy = errors.New("insufficient entropy for key generation")

	// ErrKeyGenerationFailed is returned when key generation fails for any reason
	ErrKeyGenerationFailed = errors.New("ephemeral key generation failed")
)

// BoxPublicKey represents a NaCl box public key (32 bytes)
type BoxPublicKey [32]byte

// BoxSecretKey represents a NaCl box secret key (32 bytes)
type BoxSecretKey [32]byte

// EphemeralKeyPair represents a generated ephemeral key pair
type EphemeralKeyPair struct {
	// PublicKey is the public key component
	PublicKey BoxPublicKey
	
	// SecretKey is the secret key component
	SecretKey BoxSecretKey
}

// EphemeralKeyCreator provides methods for generating ephemeral key pairs
type EphemeralKeyCreator struct {
	// randReader is the source of randomness for key generation
	// Defaults to crypto/rand.Reader but can be overridden for testing
	randReader io.Reader
}

// NewEphemeralKeyCreator creates a new EphemeralKeyCreator with default settings
// Uses crypto/rand.Reader as the source of randomness
func NewEphemeralKeyCreator() *EphemeralKeyCreator {
	return &EphemeralKeyCreator{
		randReader: rand.Reader,
	}
}

// NewEphemeralKeyCreatorWithReader creates a new EphemeralKeyCreator with a custom
// randomness source. This is primarily useful for testing.
func NewEphemeralKeyCreatorWithReader(reader io.Reader) *EphemeralKeyCreator {
	if reader == nil {
		reader = rand.Reader
	}
	return &EphemeralKeyCreator{
		randReader: reader,
	}
}

// GenerateKey generates a new ephemeral key pair using the configured randomness source
// Returns ErrInsufficientEntropy if the system doesn't have enough entropy
// Returns ErrKeyGenerationFailed for other generation failures
func (ekc *EphemeralKeyCreator) GenerateKey() (*EphemeralKeyPair, error) {
	if ekc.randReader == nil {
		return nil, fmt.Errorf("%w: randomness source not configured", ErrKeyGenerationFailed)
	}

	// Generate a new NaCl box key pair
	publicKey, secretKey, err := box.GenerateKey(ekc.randReader)
	if err != nil {
		// Check if the error is related to insufficient entropy
		// The crypto/rand package typically returns an error when it can't read enough random data
		if isEntropyError(err) {
			return nil, fmt.Errorf("%w: %v", ErrInsufficientEntropy, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	if publicKey == nil || secretKey == nil {
		return nil, fmt.Errorf("%w: generated nil keys", ErrKeyGenerationFailed)
	}

	return &EphemeralKeyPair{
		PublicKey: BoxPublicKey(*publicKey),
		SecretKey: BoxSecretKey(*secretKey),
	}, nil
}

// GenerateKeys generates multiple ephemeral key pairs
// This is useful when you need to generate keys in batch
// Returns an error if any key generation fails
func (ekc *EphemeralKeyCreator) GenerateKeys(count int) ([]*EphemeralKeyPair, error) {
	if count <= 0 {
		return nil, fmt.Errorf("%w: count must be positive, got %d", ErrKeyGenerationFailed, count)
	}

	pairs := make([]*EphemeralKeyPair, 0, count)
	
	for i := 0; i < count; i++ {
		pair, err := ekc.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key %d of %d: %w", i+1, count, err)
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// isEntropyError checks if an error is related to insufficient entropy
// This helps distinguish between different types of key generation failures
func isEntropyError(err error) bool {
	if err == nil {
		return false
	}
	
	// Common entropy-related error messages
	errMsg := err.Error()
	return contains(errMsg, "entropy") || 
		   contains(errMsg, "random") ||
		   contains(errMsg, "urandom") ||
		   contains(errMsg, "RNG") ||
		   contains(errMsg, "PRNG")
}

// contains is a simple helper to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	
	// Convert to lowercase for case-insensitive comparison
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			sLower[i] = s[i] + 32
		} else {
			sLower[i] = s[i]
		}
	}
	
	for i := 0; i < len(substr); i++ {
		if substr[i] >= 'A' && substr[i] <= 'Z' {
			substrLower[i] = substr[i] + 32
		} else {
			substrLower[i] = substr[i]
		}
	}
	
	// Check if substrLower is in sLower
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	
	return false
}

// Bytes returns the public key as a byte slice
func (pk BoxPublicKey) Bytes() []byte {
	return pk[:]
}

// Bytes returns the secret key as a byte slice
func (sk BoxSecretKey) Bytes() []byte {
	return sk[:]
}

// Zero securely zeroes out the secret key in memory
// This should be called when the key is no longer needed
func (sk *BoxSecretKey) Zero() {
	for i := range sk {
		sk[i] = 0
	}
}

// Zero securely zeroes out both keys in the pair
func (pair *EphemeralKeyPair) Zero() {
	if pair == nil {
		return
	}
	pair.SecretKey.Zero()
	// Public keys don't need to be zeroed, but we do it anyway for completeness
	for i := range pair.PublicKey {
		pair.PublicKey[i] = 0
	}
}
