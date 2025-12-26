package crypto

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keybase/saltpack"
)

func TestKeyringLoader_LoadKeyring(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}
	
	// Save the key to the temp directory
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test key: %v", err)
	}
	
	// Create a config.json file with the username
	configFile := filepath.Join(tempDir, "config.json")
	configData := []byte(`{"current_user": "testuser"}`)
	if err := os.WriteFile(configFile, configData, 0600); err != nil {
		t.Fatalf("Failed to write config.json: %v", err)
	}
	
	// Create keyring loader with custom config directory
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Load keyring for explicit user (avoid needing actual Keybase CLI)
	keyring, err := loader.LoadKeyringForUser("testuser")
	if err != nil {
		t.Fatalf("Failed to load keyring: %v", err)
	}
	
	if keyring == nil {
		t.Fatal("Keyring is nil")
	}
	
	// Verify the keyring can be used for decryption
	// Create a simple encryptor and encrypt some data
	encryptor, err := NewEncryptor(&EncryptorConfig{
		SenderKey: testKey.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	plaintext := []byte("Hello, World!")
	receivers := []saltpack.BoxPublicKey{testKey.PublicKey}
	
	ciphertext, err := encryptor.Encrypt(plaintext, receivers)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	
	// Create decryptor with the loaded keyring
	decryptor, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	// Decrypt
	decrypted, _, err := decryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}
	
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match. Expected %q, got %q", plaintext, decrypted)
	}
}

func TestKeyringLoader_Caching(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}
	
	// Save the key to the temp directory
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test key: %v", err)
	}
	
	// Create a config.json file with the username
	configFile := filepath.Join(tempDir, "config.json")
	configData := []byte(`{"current_user": "testuser"}`)
	if err := os.WriteFile(configFile, configData, 0600); err != nil {
		t.Fatalf("Failed to write config.json: %v", err)
	}
	
	// Create keyring loader with short TTL
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// First load
	keyring1, err := loader.LoadKeyringForUser("testuser")
	if err != nil {
		t.Fatalf("Failed to load keyring (first): %v", err)
	}
	
	// Check cache stats
	stats := loader.GetCacheStats()
	if stats.TotalCached != 1 {
		t.Errorf("Expected 1 cached key, got %d", stats.TotalCached)
	}
	if stats.ValidCount != 1 {
		t.Errorf("Expected 1 valid key, got %d", stats.ValidCount)
	}
	
	// Second load should use cache
	keyring2, err := loader.LoadKeyringForUser("testuser")
	if err != nil {
		t.Fatalf("Failed to load keyring (second): %v", err)
	}
	
	// Both keyrings should work
	if keyring1 == nil || keyring2 == nil {
		t.Fatal("One or both keyrings are nil")
	}
	
	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)
	
	// Check that cache has expired entries
	stats = loader.GetCacheStats()
	if stats.ExpiredCount != 1 {
		t.Errorf("Expected 1 expired key, got %d", stats.ExpiredCount)
	}
	
	// Third load should reload from disk
	keyring3, err := loader.LoadKeyringForUser("testuser")
	if err != nil {
		t.Fatalf("Failed to load keyring (third): %v", err)
	}
	
	if keyring3 == nil {
		t.Fatal("Keyring is nil")
	}
	
	// Cache should be refreshed
	stats = loader.GetCacheStats()
	if stats.ValidCount != 1 {
		t.Errorf("Expected 1 valid key after refresh, got %d", stats.ValidCount)
	}
}

func TestKeyringLoader_LoadKeyringForUser(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create test keys for multiple users
	users := []string{"alice", "bob", "charlie"}
	keys := make(map[string]*SenderKey)
	
	for _, username := range users {
		testKey, err := CreateTestSenderKey(username)
		if err != nil {
			t.Fatalf("Failed to create test key for %s: %v", username, err)
		}
		keys[username] = testKey
		
		// Save the key to the temp directory
		if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
			t.Fatalf("Failed to save test key for %s: %v", username, err)
		}
	}
	
	// Create keyring loader
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Load keyrings for each user
	for _, username := range users {
		keyring, err := loader.LoadKeyringForUser(username)
		if err != nil {
			t.Fatalf("Failed to load keyring for %s: %v", username, err)
		}
		
		if keyring == nil {
			t.Fatalf("Keyring for %s is nil", username)
		}
		
		// Verify the keyring contains the correct key by trying to decrypt
		testKey := keys[username]
		
		// Encrypt data with this user's public key
		encryptor, err := NewEncryptor(&EncryptorConfig{
			SenderKey: testKey.SecretKey,
		})
		if err != nil {
			t.Fatalf("Failed to create encryptor for %s: %v", username, err)
		}
		
		plaintext := []byte("Hello, " + username + "!")
		receivers := []saltpack.BoxPublicKey{testKey.PublicKey}
		
		ciphertext, err := encryptor.Encrypt(plaintext, receivers)
		if err != nil {
			t.Fatalf("Failed to encrypt for %s: %v", username, err)
		}
		
		// Decrypt with the loaded keyring
		decryptor, err := NewDecryptor(&DecryptorConfig{
			Keyring: keyring,
		})
		if err != nil {
			t.Fatalf("Failed to create decryptor for %s: %v", username, err)
		}
		
		decrypted, _, err := decryptor.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Failed to decrypt for %s: %v", username, err)
		}
		
		if string(decrypted) != string(plaintext) {
			t.Errorf("Decrypted text doesn't match for %s. Expected %q, got %q", 
				username, plaintext, decrypted)
		}
	}
	
	// Check that all users are cached
	cachedUsers := loader.GetCachedUsers()
	if len(cachedUsers) != len(users) {
		t.Errorf("Expected %d cached users, got %d", len(users), len(cachedUsers))
	}
}

func TestKeyringLoader_GetSecretKey(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}
	
	// Save the key to the temp directory
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test key: %v", err)
	}
	
	// Create keyring loader
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Get secret key
	secretKey, err := loader.GetSecretKey("testuser")
	if err != nil {
		t.Fatalf("Failed to get secret key: %v", err)
	}
	
	if secretKey == nil {
		t.Fatal("Secret key is nil")
	}
	
	// Verify the key is valid
	if err := ValidateSecretKey(secretKey); err != nil {
		t.Errorf("Secret key validation failed: %v", err)
	}
	
	// Verify the public key matches
	publicKey := secretKey.GetPublicKey()
	if !KeysEqual(publicKey, testKey.PublicKey) {
		t.Error("Public key doesn't match expected key")
	}
}

func TestKeyringLoader_InvalidateCache(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}
	
	// Save the key to the temp directory
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test key: %v", err)
	}
	
	// Create a config.json file with the username
	configFile := filepath.Join(tempDir, "config.json")
	configData := []byte(`{"current_user": "testuser"}`)
	if err := os.WriteFile(configFile, configData, 0600); err != nil {
		t.Fatalf("Failed to write config.json: %v", err)
	}
	
	// Create keyring loader
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Load keyring
	_, err = loader.LoadKeyringForUser("testuser")
	if err != nil {
		t.Fatalf("Failed to load keyring: %v", err)
	}
	
	// Verify cache has entry
	stats := loader.GetCacheStats()
	if stats.TotalCached != 1 {
		t.Errorf("Expected 1 cached key, got %d", stats.TotalCached)
	}
	
	// Invalidate cache
	loader.InvalidateCache()
	
	// Verify cache is empty
	stats = loader.GetCacheStats()
	if stats.TotalCached != 0 {
		t.Errorf("Expected 0 cached keys after invalidation, got %d", stats.TotalCached)
	}
}

func TestKeyringLoader_InvalidateCacheForUser(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create test keys for multiple users
	users := []string{"alice", "bob"}
	
	for _, username := range users {
		testKey, err := CreateTestSenderKey(username)
		if err != nil {
			t.Fatalf("Failed to create test key for %s: %v", username, err)
		}
		
		// Save the key to the temp directory
		if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
			t.Fatalf("Failed to save test key for %s: %v", username, err)
		}
	}
	
	// Create keyring loader
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Load keyrings for both users
	for _, username := range users {
		_, err := loader.LoadKeyringForUser(username)
		if err != nil {
			t.Fatalf("Failed to load keyring for %s: %v", username, err)
		}
	}
	
	// Verify both are cached
	stats := loader.GetCacheStats()
	if stats.TotalCached != 2 {
		t.Errorf("Expected 2 cached keys, got %d", stats.TotalCached)
	}
	
	// Invalidate cache for alice
	loader.InvalidateCacheForUser("alice")
	
	// Verify only alice's cache is removed
	cachedUsers := loader.GetCachedUsers()
	if len(cachedUsers) != 1 {
		t.Errorf("Expected 1 cached user, got %d", len(cachedUsers))
	}
	if len(cachedUsers) > 0 && cachedUsers[0] != "bob" {
		t.Errorf("Expected bob to be cached, got %s", cachedUsers[0])
	}
}

func TestKeyringLoader_CleanupExpiredKeys(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create test keys for multiple users
	users := []string{"alice", "bob", "charlie"}
	
	for _, username := range users {
		testKey, err := CreateTestSenderKey(username)
		if err != nil {
			t.Fatalf("Failed to create test key for %s: %v", username, err)
		}
		
		// Save the key to the temp directory
		if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
			t.Fatalf("Failed to save test key for %s: %v", username, err)
		}
	}
	
	// Create keyring loader with short TTL
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Load keyrings for all users
	for _, username := range users {
		_, err := loader.LoadKeyringForUser(username)
		if err != nil {
			t.Fatalf("Failed to load keyring for %s: %v", username, err)
		}
	}
	
	// Verify all are cached
	stats := loader.GetCacheStats()
	if stats.TotalCached != 3 {
		t.Errorf("Expected 3 cached keys, got %d", stats.TotalCached)
	}
	
	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)
	
	// Cleanup expired keys
	removed := loader.CleanupExpiredKeys()
	if removed != 3 {
		t.Errorf("Expected to remove 3 expired keys, removed %d", removed)
	}
	
	// Verify cache is empty
	stats = loader.GetCacheStats()
	if stats.TotalCached != 0 {
		t.Errorf("Expected 0 cached keys after cleanup, got %d", stats.TotalCached)
	}
}

func TestKeyringLoader_SetTTL(t *testing.T) {
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: t.TempDir(),
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Check initial TTL
	if loader.GetTTL() != 1*time.Hour {
		t.Errorf("Expected TTL of 1 hour, got %v", loader.GetTTL())
	}
	
	// Change TTL
	loader.SetTTL(30 * time.Minute)
	
	// Check updated TTL
	if loader.GetTTL() != 30*time.Minute {
		t.Errorf("Expected TTL of 30 minutes, got %v", loader.GetTTL())
	}
}

func TestKeyringLoader_DefaultTTL(t *testing.T) {
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: t.TempDir(),
		// TTL not specified, should use default
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Check that default TTL is 1 hour
	if loader.GetTTL() != 1*time.Hour {
		t.Errorf("Expected default TTL of 1 hour, got %v", loader.GetTTL())
	}
}

func TestKeyringLoader_ConcurrentAccess(t *testing.T) {
	// Create a temporary config directory for testing
	tempDir := t.TempDir()
	
	// Create a test sender key
	testKey, err := CreateTestSenderKey("testuser")
	if err != nil {
		t.Fatalf("Failed to create test key: %v", err)
	}
	
	// Save the key to the temp directory
	if err := SaveSenderKeyForTesting(testKey, tempDir); err != nil {
		t.Fatalf("Failed to save test key: %v", err)
	}
	
	// Create keyring loader
	loader, err := NewKeyringLoader(&KeyringLoaderConfig{
		ConfigDir: tempDir,
		TTL:       1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create keyring loader: %v", err)
	}
	
	// Run concurrent loads
	const numGoroutines = 10
	done := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := loader.LoadKeyringForUser("testuser")
			done <- err
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent load failed: %v", err)
		}
	}
	
	// Verify only one entry is cached
	stats := loader.GetCacheStats()
	if stats.TotalCached != 1 {
		t.Errorf("Expected 1 cached key after concurrent access, got %d", stats.TotalCached)
	}
}

func TestLoadDefaultKeyring_Integration(t *testing.T) {
	// This test requires actual Keybase installation
	// Skip if Keybase is not available
	
	// Try to load the default keyring
	keyring, err := LoadDefaultKeyring()
	
	// If Keybase is not installed or no user is logged in, this should fail gracefully
	if err != nil {
		// This is expected if Keybase is not set up
		t.Logf("LoadDefaultKeyring failed (expected if Keybase not installed): %v", err)
		return
	}
	
	// If we get here, Keybase is available
	if keyring == nil {
		t.Fatal("Keyring is nil despite no error")
	}
	
	t.Log("Successfully loaded default keyring (Keybase is installed and configured)")
}

func TestLoadKeyringForUsername_Integration(t *testing.T) {
	// This test requires actual Keybase installation
	// Skip if Keybase is not available
	
	// Try to load keyring for a non-existent user
	_, err := LoadKeyringForUsername("nonexistentuser12345")
	
	// This should fail
	if err == nil {
		t.Error("Expected error when loading keyring for non-existent user")
	}
	
	t.Logf("LoadKeyringForUsername correctly failed for non-existent user: %v", err)
}
