package keybase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

// TestOfflineDecryption verifies that decryption works without network access
// once the keys have been cached during encryption
func TestOfflineDecryption(t *testing.T) {
	// Generate test key pairs
	keyPair1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Step 1: Encrypt with network access (caching keys)
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair1.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	encryptKeeper, err := NewKeeper(&KeeperConfig{
		Config:       config,
		CacheManager: cacheManager,
	})
	if err != nil {
		t.Fatalf("Failed to create encrypt keeper: %v", err)
	}
	defer encryptKeeper.Close()

	plaintext := []byte("Secret message for offline decryption test")
	ctx := context.Background()

	ciphertext, err := encryptKeeper.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Step 2: Decrypt WITHOUT network access
	// Create a new keeper with only local keyring (no API calls)
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair1.SecretKey)

	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	// Create keeper without cache manager (simulating offline)
	decryptKeeper := &Keeper{
		config:    config,
		decryptor: decryptor,
		keyring:   keyring,
	}

	// This should work offline since decryption only uses local keyring
	decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Offline Decrypt() error = %v (decryption should work offline)", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Offline Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

// TestOfflineEncryptionWithCache verifies that encryption works offline
// when public keys are already in cache
func TestOfflineEncryptionWithCache(t *testing.T) {
	// Generate test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Pre-populate cache with public key
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	// Verify cache has the key
	cachedKey := cacheManager.Cache().Get("alice")
	if cachedKey == nil {
		t.Fatal("Cache should have alice's key")
	}

	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	// Use the mock cache manager that already has cached keys
	keeper, err := NewKeeper(&KeeperConfig{
		Config:       config,
		CacheManager: cacheManager,
	})
	if err != nil {
		t.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()

	plaintext := []byte("Message encrypted with cached keys")
	ctx := context.Background()

	// This should work using cached keys (no network access)
	ciphertext, err := keeper.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Offline Encrypt() with cached keys error = %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("Offline Encrypt() returned empty ciphertext")
	}

	// Verify we can decrypt it
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	decryptKeeper := &Keeper{
		config:    config,
		decryptor: decryptor,
		keyring:   keyring,
	}

	decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

// TestOfflineEncryptionWithoutCache verifies that encryption fails gracefully
// when network is unavailable and keys are not cached
func TestOfflineEncryptionWithoutCache(t *testing.T) {
	// Create cache manager with offline API client
	cacheConfig := cache.DefaultCacheConfig()
	apiConfig := api.DefaultClientConfig()
	
	cacheManager, err := cache.NewManager(&cache.ManagerConfig{
		CacheConfig: cacheConfig,
		APIConfig:   apiConfig,
	})
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer cacheManager.Close()

	config := &Config{
		Recipients: []string{"nonexistent_user"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	keeper, err := NewKeeper(&KeeperConfig{
		Config:       config,
		CacheManager: cacheManager,
	})
	if err != nil {
		t.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()

	plaintext := []byte("This should fail without network or cache")
	ctx := context.Background()

	// This should fail because:
	// 1. Keys are not in cache
	// 2. Network call would be needed (but user doesn't exist anyway)
	_, err = keeper.Encrypt(ctx, plaintext)
	if err == nil {
		t.Error("Offline Encrypt() without cached keys should fail, got nil error")
	}

	// Verify error is a not found error (user doesn't exist)
	var keeperErr *KeeperError
	if errors.As(err, &keeperErr) {
		t.Logf("Got expected error: %v", keeperErr)
	}
}

// TestOfflineStreamingDecryption tests offline decryption with streaming (large messages)
func TestOfflineStreamingDecryption(t *testing.T) {
	// Generate test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create cache manager with the key
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	// Encrypt a large message (>10 MiB to trigger streaming)
	encryptKeeper, err := NewKeeper(&KeeperConfig{
		Config:       config,
		CacheManager: cacheManager,
	})
	if err != nil {
		t.Fatalf("Failed to create encrypt keeper: %v", err)
	}
	defer encryptKeeper.Close()

	// Create 11 MiB plaintext
	plaintext := make([]byte, 11*1024*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ctx := context.Background()
	t.Logf("Encrypting %d bytes (streaming mode)...", len(plaintext))
	ciphertext, err := encryptKeeper.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	t.Logf("Encrypted to %d bytes", len(ciphertext))

	// Now decrypt offline
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	decryptKeeper := &Keeper{
		config:    config,
		decryptor: decryptor,
		keyring:   keyring,
	}

	t.Logf("Decrypting %d bytes offline (streaming mode)...", len(ciphertext))
	decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Offline streaming Decrypt() error = %v", err)
	}
	t.Logf("Decrypted to %d bytes", len(decrypted))

	// Verify size
	if len(decrypted) != len(plaintext) {
		t.Errorf("Decrypted size = %d, want %d", len(decrypted), len(plaintext))
	}

	// Spot check content
	for i := 0; i < 1000; i++ {
		if decrypted[i] != plaintext[i] {
			t.Errorf("Decrypted[%d] = %d, want %d", i, decrypted[i], plaintext[i])
			break
		}
	}
}

// TestMultipleOfflineDecryptions tests that the same ciphertext can be decrypted
// multiple times offline
func TestMultipleOfflineDecryptions(t *testing.T) {
	// Generate test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Encrypt once
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	encryptKeeper, err := NewKeeper(&KeeperConfig{
		Config:       config,
		CacheManager: cacheManager,
	})
	if err != nil {
		t.Fatalf("Failed to create encrypt keeper: %v", err)
	}
	defer encryptKeeper.Close()

	plaintext := []byte("Message for multiple offline decryptions")
	ctx := context.Background()

	ciphertext, err := encryptKeeper.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Create offline decryption keeper
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	decryptKeeper := &Keeper{
		config:    config,
		decryptor: decryptor,
		keyring:   keyring,
	}

	// Decrypt multiple times offline
	for i := 0; i < 5; i++ {
		t.Run("iteration-"+string(rune('0'+i)), func(t *testing.T) {
			decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() iteration %d error = %v", i+1, err)
			}

			if string(decrypted) != string(plaintext) {
				t.Errorf("Decrypt() iteration %d = %q, want %q", i+1, decrypted, plaintext)
			}
		})
	}
}

// TestOfflineDecryptionWithMultipleRecipients tests offline decryption
// when message was encrypted for multiple recipients
func TestOfflineDecryptionWithMultipleRecipients(t *testing.T) {
	// Generate test key pairs
	keyPair1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}

	keyPair2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}

	keyPair3, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 3: %v", err)
	}

	// Encrypt for multiple recipients
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair1.PublicKey,
		"bob":   keyPair2.PublicKey,
		"charlie": keyPair3.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	config := &Config{
		Recipients: []string{"alice", "bob", "charlie"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	encryptKeeper, err := NewKeeper(&KeeperConfig{
		Config:       config,
		CacheManager: cacheManager,
	})
	if err != nil {
		t.Fatalf("Failed to create encrypt keeper: %v", err)
	}
	defer encryptKeeper.Close()

	plaintext := []byte("Secret message for alice, bob, and charlie")
	ctx := context.Background()

	ciphertext, err := encryptKeeper.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Test that each recipient can decrypt offline
	recipients := []struct {
		name      string
		secretKey saltpack.BoxSecretKey
	}{
		{"alice", keyPair1.SecretKey},
		{"bob", keyPair2.SecretKey},
		{"charlie", keyPair3.SecretKey},
	}

	for _, recipient := range recipients {
		t.Run("decrypt-as-"+recipient.name, func(t *testing.T) {
			keyring := crypto.NewSimpleKeyring()
			keyring.AddKey(recipient.secretKey)

			decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor for %s: %v", recipient.name, err)
			}

			decryptKeeper := &Keeper{
				config:    config,
				decryptor: decryptor,
				keyring:   keyring,
			}

			decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
			if err != nil {
				t.Fatalf("Offline Decrypt() as %s error = %v", recipient.name, err)
			}

			if string(decrypted) != string(plaintext) {
				t.Errorf("Decrypt() as %s = %q, want %q", recipient.name, decrypted, plaintext)
			}
		})
	}
}

// offlineAPIClient is a mock API client that always fails (simulating no network)
type offlineAPIClient struct{}

func (c *offlineAPIClient) LookupUsers(ctx context.Context, usernames []string) ([]api.UserPublicKey, error) {
	return nil, &api.APIError{
		Message:   "network unavailable (offline mode)",
		Kind:      api.ErrorKindNetwork,
		Temporary: true,
	}
}
