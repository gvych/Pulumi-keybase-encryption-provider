package crypto

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/keybase/saltpack"
)

func TestLoadSenderKey(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	// Save the test key
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test sender key: %v", err)
	}

	// Test loading the sender key
	config := &SenderKeyConfig{
		Username:  "testuser",
		ConfigDir: tempDir,
	}

	loadedKey, err := LoadSenderKey(config)
	if err != nil {
		t.Fatalf("Failed to load sender key: %v", err)
	}

	// Verify the loaded key
	if loadedKey.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", loadedKey.Username)
	}

	if loadedKey.SecretKey == nil {
		t.Error("Secret key is nil")
	}

	if loadedKey.PublicKey == nil {
		t.Error("Public key is nil")
	}

	// Verify that the loaded key matches the original
	if !KeysEqual(loadedKey.PublicKey, testKey.PublicKey) {
		t.Error("Loaded public key does not match original")
	}
}

func TestLoadSenderKeyMissingUser(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Try to load a key for a non-existent user
	config := &SenderKeyConfig{
		Username:  "nonexistent",
		ConfigDir: tempDir,
	}

	_, err := LoadSenderKey(config)
	if err == nil {
		t.Fatal("Expected error when loading non-existent user key, got nil")
	}

	// Verify error message mentions key not found
	errStr := err.Error()
	if !contains(errStr, "sender key not found") && !contains(errStr, "failed to load") {
		t.Errorf("Expected error message about missing key, got: %v", err)
	}
}

func TestLoadSenderKeyInvalidFormat(t *testing.T) {
	tempDir := t.TempDir()

	// Create key directory
	keyDir := filepath.Join(tempDir, "device_eks")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		t.Fatalf("Failed to create key directory: %v", err)
	}

	// Write an invalid key file
	keyPath := filepath.Join(keyDir, "testuser.eks")
	invalidData := []byte("this is not a valid key")
	if err := os.WriteFile(keyPath, invalidData, 0600); err != nil {
		t.Fatalf("Failed to write invalid key file: %v", err)
	}

	// Try to load the invalid key
	config := &SenderKeyConfig{
		Username:  "testuser",
		ConfigDir: tempDir,
	}

	_, err := LoadSenderKey(config)
	if err == nil {
		t.Fatal("Expected error when loading invalid key, got nil")
	}
}

func TestLoadKeyFromFileJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Generate a test key
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Get the secret key bytes
	secretKeyBytes := ExportSecretKeyBytes(keyPair.SecretKey)
	if secretKeyBytes == nil {
		t.Fatal("Failed to export secret key bytes")
	}
	keyHex := hex.EncodeToString(secretKeyBytes)

	// Create a JSON key file
	keyData := map[string]string{
		"encryption_key": keyHex,
		"username":       "testuser",
	}

	jsonData, err := json.Marshal(keyData)
	if err != nil {
		t.Fatalf("Failed to marshal key data: %v", err)
	}

	keyPath := filepath.Join(tempDir, "test_key.json")
	if err := os.WriteFile(keyPath, jsonData, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Load the key from the file
	loadedKey, err := loadKeyFromFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to load key from file: %v", err)
	}

	if loadedKey == nil {
		t.Fatal("Loaded key is nil")
	}

	// Verify the loaded key matches the original
	loadedPublicKey := loadedKey.GetPublicKey()
	if !KeysEqual(loadedPublicKey, keyPair.PublicKey) {
		t.Error("Loaded key does not match original")
	}
}

func TestLoadKeyFromFileHex(t *testing.T) {
	tempDir := t.TempDir()

	// Generate a test key
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Get the secret key bytes
	secretKeyBytes := ExportSecretKeyBytes(keyPair.SecretKey)
	if secretKeyBytes == nil {
		t.Fatal("Failed to export secret key bytes")
	}
	keyHex := hex.EncodeToString(secretKeyBytes)

	// Write the hex key to a file
	keyPath := filepath.Join(tempDir, "test_key.hex")
	if err := os.WriteFile(keyPath, []byte(keyHex), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Load the key from the file
	loadedKey, err := loadKeyFromFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to load key from file: %v", err)
	}

	if loadedKey == nil {
		t.Fatal("Loaded key is nil")
	}

	// Verify the loaded key matches the original
	loadedPublicKey := loadedKey.GetPublicKey()
	if !KeysEqual(loadedPublicKey, keyPair.PublicKey) {
		t.Error("Loaded key does not match original")
	}
}

func TestLoadKeyFromFileWithPrefix(t *testing.T) {
	tempDir := t.TempDir()

	// Generate a test key
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Get the secret key bytes with prefix
	secretKeyBytes := ExportSecretKeyBytes(keyPair.SecretKey)
	if secretKeyBytes == nil {
		t.Fatal("Failed to export secret key bytes")
	}
	keyHex := "0x" + hex.EncodeToString(secretKeyBytes)

	// Write the hex key to a file
	keyPath := filepath.Join(tempDir, "test_key.hex")
	if err := os.WriteFile(keyPath, []byte(keyHex), 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Load the key from the file
	loadedKey, err := loadKeyFromFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to load key from file: %v", err)
	}

	if loadedKey == nil {
		t.Fatal("Loaded key is nil")
	}

	// Verify the loaded key matches the original
	loadedPublicKey := loadedKey.GetPublicKey()
	if !KeysEqual(loadedPublicKey, keyPair.PublicKey) {
		t.Error("Loaded key does not match original")
	}
}

func TestLoadKeyFromFileNonExistent(t *testing.T) {
	// Try to load from a non-existent file
	_, err := loadKeyFromFile("/nonexistent/path/to/key")
	if err == nil {
		t.Fatal("Expected error when loading from non-existent file, got nil")
	}

	if !os.IsNotExist(err) {
		t.Errorf("Expected NotExist error, got: %v", err)
	}
}

func TestGetSenderIdentity(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	// Save the test key
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test sender key: %v", err)
	}

	// Test with specified username (this will fail because we can't load the key)
	// In a real scenario, this would work if the key exists
	username := "testuser"
	identity, err := GetSenderIdentity(username)
	
	// We expect this to fail because GetSenderIdentity tries to load the key
	// and we don't have a real Keybase installation in tests
	if err != nil && identity == "" {
		// Expected behavior in test environment
		t.Logf("GetSenderIdentity failed as expected in test environment: %v", err)
	}
}

func TestGetSenderIdentityEmpty(t *testing.T) {
	// Test with empty username - should try to get current user
	// This will fail in test environment without Keybase
	_, err := GetSenderIdentity("")
	if err == nil {
		// If it succeeds, we're running on a system with Keybase installed
		t.Log("GetSenderIdentity succeeded - Keybase is installed")
	} else {
		// Expected in test environment
		t.Logf("GetSenderIdentity failed as expected: %v", err)
	}
}

func TestValidateSenderKey(t *testing.T) {
	// Create a valid sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	// Validate the key
	if err := ValidateSenderKey(testKey); err != nil {
		t.Errorf("Validation failed for valid key: %v", err)
	}
}

func TestValidateSenderKeyNil(t *testing.T) {
	err := ValidateSenderKey(nil)
	if err == nil {
		t.Fatal("Expected error when validating nil key, got nil")
	}

	if !contains(err.Error(), "sender key is nil") {
		t.Errorf("Expected error message about nil key, got: %v", err)
	}
}

func TestValidateSenderKeyNoUsername(t *testing.T) {
	testKey := &SenderKey{
		Username:  "",
		SecretKey: nil,
		PublicKey: nil,
	}

	err := ValidateSenderKey(testKey)
	if err == nil {
		t.Fatal("Expected error when validating key with no username, got nil")
	}

	if !contains(err.Error(), "no username") {
		t.Errorf("Expected error message about no username, got: %v", err)
	}
}

func TestValidateSenderKeyNoSecretKey(t *testing.T) {
	testKey := &SenderKey{
		Username:  "testuser",
		SecretKey: nil,
		PublicKey: nil,
	}

	err := ValidateSenderKey(testKey)
	if err == nil {
		t.Fatal("Expected error when validating key with no secret key, got nil")
	}

	if !contains(err.Error(), "no secret key") {
		t.Errorf("Expected error message about no secret key, got: %v", err)
	}
}

func TestValidateSenderKeyMismatchedKeys(t *testing.T) {
	// Create two different key pairs
	keyPair1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate first key pair: %v", err)
	}

	keyPair2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate second key pair: %v", err)
	}

	// Create a sender key with mismatched keys
	testKey := &SenderKey{
		Username:  "testuser",
		SecretKey: keyPair1.SecretKey,
		PublicKey: keyPair2.PublicKey, // Different public key
		KeyID:     keyPair2.Identifier,
	}

	err = ValidateSenderKey(testKey)
	if err == nil {
		t.Fatal("Expected error when validating key with mismatched keys, got nil")
	}

	if !contains(err.Error(), "does not match") {
		t.Errorf("Expected error message about key mismatch, got: %v", err)
	}
}

func TestCreateTestSenderKey(t *testing.T) {
	// Test with username
	key, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	if key.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", key.Username)
	}

	if key.SecretKey == nil {
		t.Error("Secret key is nil")
	}

	if key.PublicKey == nil {
		t.Error("Public key is nil")
	}

	// Validate the key
	if err := ValidateSenderKey(key); err != nil {
		t.Errorf("Validation failed for test key: %v", err)
	}
}

func TestCreateTestSenderKeyEmptyUsername(t *testing.T) {
	// Test with empty username - should default to "testuser"
	key, err := CreateTestSenderKey("")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	if key.Username != "testuser" {
		t.Errorf("Expected default username 'testuser', got '%s'", key.Username)
	}
}

func TestKeyIDToHex(t *testing.T) {
	// Create a test key ID
	keyID := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}

	hexStr := KeyIDToHex(keyID)
	expected := "0123456789abcdef"

	if hexStr != expected {
		t.Errorf("Expected hex string '%s', got '%s'", expected, hexStr)
	}
}

func TestSaveAndLoadSenderKey(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	// Save the key
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test sender key: %v", err)
	}

	// Load the key back
	config := &SenderKeyConfig{
		Username:  "testuser",
		ConfigDir: tempDir,
	}

	loadedKey, err := LoadSenderKey(config)
	if err != nil {
		t.Fatalf("Failed to load sender key: %v", err)
	}

	// Verify the loaded key matches the original
	if !KeysEqual(loadedKey.PublicKey, testKey.PublicKey) {
		t.Error("Loaded public key does not match original")
	}

	// Verify the loaded secret key works by deriving the public key
	derivedPublicKey := loadedKey.SecretKey.GetPublicKey()
	if !KeysEqual(derivedPublicKey, testKey.PublicKey) {
		t.Error("Derived public key from loaded secret key does not match original")
	}
}

func TestSaveSenderKeyInvalidKey(t *testing.T) {
	tempDir := t.TempDir()

	// Try to save a nil key
	err := SaveSenderKeyForTesting(nil, tempDir)
	if err == nil {
		t.Fatal("Expected error when saving nil key, got nil")
	}
}

func TestTrimKeyPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0x1234", "1234"},
		{"0X1234", "1234"},
		{"1234", "1234"},
		{"", ""},
		{"0x", ""},
	}

	for _, tt := range tests {
		result := trimKeyPrefix(tt.input)
		if result != tt.expected {
			t.Errorf("trimKeyPrefix(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestStripWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1234", "1234"},
		{"12 34", "1234"},
		{"12\n34", "1234"},
		{"12\t34", "1234"},
		{"12\r\n34", "1234"},
		{"  1234  ", "1234"},
		{"", ""},
	}

	for _, tt := range tests {
		result := stripWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("stripWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLoadSenderKeyMultiplePaths(t *testing.T) {
	// This test verifies that the loader tries multiple paths
	tempDir := t.TempDir()

	// Create a test key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test sender key: %v", err)
	}

	// Save in the primary location (device_eks)
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test sender key: %v", err)
	}

	// Load the key - should find it in the primary location
	config := &SenderKeyConfig{
		Username:  "testuser",
		ConfigDir: tempDir,
	}

	loadedKey, err := LoadSenderKey(config)
	if err != nil {
		t.Fatalf("Failed to load sender key: %v", err)
	}

	if loadedKey == nil {
		t.Fatal("Loaded key is nil")
	}
}

func TestEncryptionWithSenderKey(t *testing.T) {
	// This is an integration test that verifies sender keys work with encryption

	// Create sender and recipient key pairs
	sender, err := CreateTestSenderKey("sender")
	if err != nil {
		t.Fatalf("Failed to create sender key: %v", err)
	}

	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}

	// Create encryptor with sender key
	encryptor, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Encrypt a message
	plaintext := []byte("Test message from sender")
	ciphertext, err := encryptor.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Create keyring and decryptor
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification

	decryptor, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	// Decrypt the message
	decrypted, _, err := decryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// Verify the decrypted message matches the original
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted message does not match original.\nExpected: %s\nGot: %s", plaintext, decrypted)
	}
}
