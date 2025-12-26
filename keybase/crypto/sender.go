package crypto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
)

// SenderKeyConfig holds configuration for sender key loading
type SenderKeyConfig struct {
	// Username specifies the sender's Keybase username
	// If empty, the currently logged-in user is used
	Username string

	// ConfigDir is the Keybase configuration directory
	// If empty, the default directory is used (~/.config/keybase on Linux/macOS)
	ConfigDir string
}

// SenderKey represents a loaded sender key
type SenderKey struct {
	// Username is the Keybase username for this key
	Username string

	// SecretKey is the loaded secret key
	SecretKey saltpack.BoxSecretKey

	// PublicKey is the corresponding public key
	PublicKey saltpack.BoxPublicKey

	// KeyID is the key identifier
	KeyID []byte
}

// LoadSenderKey loads the sender's private key from the Keybase configuration directory
//
// This function:
// 1. Determines the sender identity (current Keybase user or configured username)
// 2. Locates the Keybase configuration directory
// 3. Loads the sender's private key from the keyring
// 4. Validates the key format
// 5. Returns a SenderKey struct with the loaded key
//
// If config is nil, it uses default settings (current logged-in user, default config directory)
func LoadSenderKey(config *SenderKeyConfig) (*SenderKey, error) {
	if config == nil {
		config = &SenderKeyConfig{}
	}

	// Step 1: Determine sender identity
	username := config.Username
	if username == "" {
		// Use currently logged-in user
		currentUser, err := credentials.GetUsername()
		if err != nil {
			return nil, fmt.Errorf("failed to determine sender identity: %w", err)
		}
		username = currentUser
	}

	// Step 2: Get Keybase config directory
	configDir := config.ConfigDir
	if configDir == "" {
		status, err := credentials.DiscoverCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed to discover Keybase configuration: %w", err)
		}
		configDir = status.ConfigDir
	}

	// Step 3: Load the sender's private key
	secretKey, err := loadPrivateKey(configDir, username)
	if err != nil {
		return nil, fmt.Errorf("failed to load sender private key for user '%s': %w", username, err)
	}

	// Step 4: Validate the key format
	if err := ValidateSecretKey(secretKey); err != nil {
		return nil, fmt.Errorf("invalid sender secret key for user '%s': %w", username, err)
	}

	// Get the public key from the secret key
	publicKey := secretKey.GetPublicKey()
	if publicKey == nil {
		return nil, fmt.Errorf("failed to derive public key from secret key for user '%s'", username)
	}

	// Get the key ID
	keyID := publicKey.ToKID()

	return &SenderKey{
		Username:  username,
		SecretKey: secretKey,
		PublicKey: publicKey,
		KeyID:     keyID,
	}, nil
}

// loadPrivateKey loads a private key from the Keybase configuration directory
//
// The Keybase client stores keys in various formats. This function attempts to:
// 1. Read the user's device keys from the config directory
// 2. Parse the key file to extract the NaCl encryption key
// 3. Convert the key to saltpack.BoxSecretKey format
//
// Note: The actual Keybase key storage format is complex and may vary.
// This is a simplified implementation that handles the common case.
func loadPrivateKey(configDir, username string) (saltpack.BoxSecretKey, error) {
	// Try multiple possible key locations
	possiblePaths := []string{
		// Modern Keybase stores keys in the device_eks directory
		filepath.Join(configDir, "device_eks", fmt.Sprintf("%s.eks", username)),
		// Legacy location
		filepath.Join(configDir, "secretkeys", username),
		// Alternative location
		filepath.Join(configDir, username, "device_keys"),
	}

	var lastErr error
	for _, keyPath := range possiblePaths {
		secretKey, err := loadKeyFromFile(keyPath)
		if err == nil {
			return secretKey, nil
		}
		lastErr = err
	}

	// If we couldn't find the key in any location, provide a helpful error
	if os.IsNotExist(lastErr) {
		return nil, fmt.Errorf("sender key not found for user '%s': ensure Keybase is properly configured and you have encryption keys set up. Run 'keybase pgp gen' to generate keys if needed", username)
	}

	return nil, fmt.Errorf("failed to load sender key: %w", lastErr)
}

// loadKeyFromFile loads a key from a specific file path
func loadKeyFromFile(path string) (saltpack.BoxSecretKey, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	// Read the key file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Try to parse as JSON first (common Keybase format)
	var keyData struct {
		// Keybase typically stores keys in base64 or hex format
		EncryptionKey string `json:"encryption_key"`
		BoxKey        string `json:"box_key"`
		NaclKey       string `json:"nacl_key"`
		// Raw key field
		Key string `json:"key"`
		// Hex-encoded key
		KeyHex string `json:"key_hex"`
	}

	if err := json.Unmarshal(data, &keyData); err == nil {
		// Successfully parsed as JSON, try to extract the key
		keyHex := ""
		switch {
		case keyData.EncryptionKey != "":
			keyHex = keyData.EncryptionKey
		case keyData.BoxKey != "":
			keyHex = keyData.BoxKey
		case keyData.NaclKey != "":
			keyHex = keyData.NaclKey
		case keyData.KeyHex != "":
			keyHex = keyData.KeyHex
		case keyData.Key != "":
			keyHex = keyData.Key
		}

		if keyHex != "" {
			// Remove any common prefixes
			keyHex = trimKeyPrefix(keyHex)

			// Try to parse as hex
			secretKey, err := CreateSecretKeyFromHex(keyHex)
			if err == nil {
				return secretKey, nil
			}
		}
	}

	// If JSON parsing failed or no key found, try to parse as raw hex
	hexStr := string(data)
	hexStr = trimKeyPrefix(hexStr)

	// Remove whitespace and newlines
	hexStr = stripWhitespace(hexStr)

	// Try to parse as hex
	secretKey, err := CreateSecretKeyFromHex(hexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key file: %w", err)
	}

	return secretKey, nil
}

// trimKeyPrefix removes common key prefixes
func trimKeyPrefix(key string) string {
	prefixes := []string{"0x", "0X"}
	for _, prefix := range prefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return key[len(prefix):]
		}
	}
	return key
}

// stripWhitespace removes all whitespace characters from a string
func stripWhitespace(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		// Skip whitespace characters (space, tab, newline, carriage return)
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			result = append(result, ch)
		}
	}
	return string(result)
}

// GetSenderIdentity determines the sender identity (username) to use
//
// If username is provided, it validates that the user exists and returns it.
// Otherwise, it returns the currently logged-in Keybase user.
func GetSenderIdentity(username string) (string, error) {
	if username != "" {
		// Validate that we can access this user's keys
		_, err := LoadSenderKey(&SenderKeyConfig{Username: username})
		if err != nil {
			return "", fmt.Errorf("cannot use username '%s' as sender: %w", username, err)
		}
		return username, nil
	}

	// Use current logged-in user
	currentUser, err := credentials.GetUsername()
	if err != nil {
		return "", fmt.Errorf("failed to determine current Keybase user: %w", err)
	}

	return currentUser, nil
}

// ValidateSenderKey validates that a sender key is properly formatted and usable
func ValidateSenderKey(key *SenderKey) error {
	if key == nil {
		return fmt.Errorf("sender key is nil")
	}

	if key.Username == "" {
		return fmt.Errorf("sender key has no username")
	}

	if key.SecretKey == nil {
		return fmt.Errorf("sender key has no secret key")
	}

	// Validate the secret key
	if err := ValidateSecretKey(key.SecretKey); err != nil {
		return fmt.Errorf("sender secret key validation failed: %w", err)
	}

	if key.PublicKey == nil {
		return fmt.Errorf("sender key has no public key")
	}

	// Validate the public key
	if err := ValidatePublicKey(key.PublicKey); err != nil {
		return fmt.Errorf("sender public key validation failed: %w", err)
	}

	// Verify that the public key matches the secret key
	derivedPublicKey := key.SecretKey.GetPublicKey()
	if !KeysEqual(key.PublicKey, derivedPublicKey) {
		return fmt.Errorf("public key does not match secret key")
	}

	return nil
}

// CreateTestSenderKey creates a sender key for testing purposes
// This generates a random key pair and should only be used in tests
func CreateTestSenderKey(username string) (*SenderKey, error) {
	if username == "" {
		username = "testuser"
	}

	// Generate a new key pair
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate test key pair: %w", err)
	}

	return &SenderKey{
		Username:  username,
		SecretKey: keyPair.SecretKey,
		PublicKey: keyPair.PublicKey,
		KeyID:     keyPair.Identifier,
	}, nil
}

// KeyIDToHex converts a key ID to a hex string for display
func KeyIDToHex(keyID []byte) string {
	return hex.EncodeToString(keyID)
}

// SaveSenderKeyForTesting saves a sender key to a file for testing
// This should only be used in tests
func SaveSenderKeyForTesting(key *SenderKey, configDir string) error {
	if key == nil || key.SecretKey == nil {
		return fmt.Errorf("invalid key")
	}

	// Create the directory structure
	keyDir := filepath.Join(configDir, "device_eks")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Get the secret key bytes
	secretKeyBytes := ExportSecretKeyBytes(key.SecretKey)
	if secretKeyBytes == nil {
		return fmt.Errorf("failed to export secret key bytes")
	}

	// Create key data structure
	keyData := struct {
		EncryptionKey string `json:"encryption_key"`
		Username      string `json:"username"`
	}{
		EncryptionKey: hex.EncodeToString(secretKeyBytes),
		Username:      key.Username,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	// Write to file
	keyPath := filepath.Join(keyDir, fmt.Sprintf("%s.eks", key.Username))
	if err := os.WriteFile(keyPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}
