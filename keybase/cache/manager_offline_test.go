package cache

import (
	"context"
	"testing"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
)

// TestOfflineModeGetPublicKey tests offline mode for single key lookup
func TestOfflineModeGetPublicKey(t *testing.T) {
	// Create manager in offline mode with a temporary cache
	cacheConfig := DefaultCacheConfig()
	cacheConfig.FilePath = t.TempDir() + "/test_cache.json"
	
	config := &ManagerConfig{
		CacheConfig: cacheConfig,
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create offline manager: %v", err)
	}
	defer manager.Close()

	// Clear any existing cache entries
	err = manager.Cache().Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	ctx := context.Background()

	// Should fail - no cache entry and offline mode
	_, err = manager.GetPublicKey(ctx, "alice_offline_test")
	if err == nil {
		t.Error("Expected error when fetching uncached key in offline mode, got nil")
	}

	// Verify error is an API error with NotFound kind
	var apiErr *api.APIError
	if !errorAs(err, &apiErr) {
		t.Errorf("Expected APIError, got %T: %v", err, err)
	} else if apiErr.Kind != api.ErrorKindNotFound {
		t.Errorf("Expected ErrorKindNotFound, got %v", apiErr.Kind)
	}
}

// TestOfflineModeGetPublicKeyWithCache tests offline mode with cached keys
func TestOfflineModeGetPublicKeyWithCache(t *testing.T) {
	// Create manager (not in offline mode initially)
	config := &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Manually add a key to cache
	testKeyID := "0120abc123def456"
	testPublicKey := "-----BEGIN PGP PUBLIC KEY BLOCK----- test key -----"
	err = manager.Cache().Set("alice", testPublicKey, testKeyID)
	if err != nil {
		t.Fatalf("Failed to cache key: %v", err)
	}

	// Now enable offline mode
	manager.SetOfflineMode(true)

	ctx := context.Background()

	// Should succeed - key is cached
	key, err := manager.GetPublicKey(ctx, "alice")
	if err != nil {
		t.Fatalf("Expected success with cached key in offline mode, got error: %v", err)
	}

	if key.Username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", key.Username)
	}
	if key.KeyID != testKeyID {
		t.Errorf("Expected key ID '%s', got '%s'", testKeyID, key.KeyID)
	}
}

// TestOfflineModeGetPublicKeys tests offline mode for multiple key lookup
func TestOfflineModeGetPublicKeys(t *testing.T) {
	// Create manager in offline mode with a temporary cache
	cacheConfig := DefaultCacheConfig()
	cacheConfig.FilePath = t.TempDir() + "/test_cache.json"
	
	config := &ManagerConfig{
		CacheConfig: cacheConfig,
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create offline manager: %v", err)
	}
	defer manager.Close()

	// Clear any existing cache entries
	err = manager.Cache().Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	ctx := context.Background()

	// Should fail - no cache entries
	_, err = manager.GetPublicKeys(ctx, []string{"alice_test", "bob_test"})
	if err == nil {
		t.Error("Expected error when fetching uncached keys in offline mode, got nil")
	}
}

// TestOfflineModeGetPublicKeysPartialCache tests offline mode with some keys cached
func TestOfflineModeGetPublicKeysPartialCache(t *testing.T) {
	// Create manager in offline mode with a temporary cache
	cacheConfig := DefaultCacheConfig()
	cacheConfig.FilePath = t.TempDir() + "/test_cache.json"
	
	config := &ManagerConfig{
		CacheConfig: cacheConfig,
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create offline manager: %v", err)
	}
	defer manager.Close()

	// Clear any existing cache entries
	err = manager.Cache().Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Cache only alice's key
	testKeyID := "0120abc123def456"
	testPublicKey := "-----BEGIN PGP PUBLIC KEY BLOCK----- test key -----"
	err = manager.Cache().Set("alice_partial", testPublicKey, testKeyID)
	if err != nil {
		t.Fatalf("Failed to cache key: %v", err)
	}

	ctx := context.Background()

	// Should fail - bob's key is not cached
	_, err = manager.GetPublicKeys(ctx, []string{"alice_partial", "bob_partial"})
	if err == nil {
		t.Error("Expected error when some keys are not cached in offline mode, got nil")
	}

	// Verify error message is present
	if err != nil && err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestOfflineModeGetPublicKeysAllCached tests offline mode with all keys cached
func TestOfflineModeGetPublicKeysAllCached(t *testing.T) {
	// Create manager in offline mode
	config := &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create offline manager: %v", err)
	}
	defer manager.Close()

	// Cache keys for alice and bob
	users := []struct {
		username  string
		keyID     string
		publicKey string
	}{
		{"alice", "0120abc123", "-----BEGIN PGP PUBLIC KEY BLOCK----- alice -----"},
		{"bob", "0120def456", "-----BEGIN PGP PUBLIC KEY BLOCK----- bob -----"},
		{"charlie", "0120ghi789", "-----BEGIN PGP PUBLIC KEY BLOCK----- charlie -----"},
	}

	for _, user := range users {
		err = manager.Cache().Set(user.username, user.publicKey, user.keyID)
		if err != nil {
			t.Fatalf("Failed to cache key for %s: %v", user.username, err)
		}
	}

	ctx := context.Background()

	// Should succeed - all keys are cached
	keys, err := manager.GetPublicKeys(ctx, []string{"alice", "bob", "charlie"})
	if err != nil {
		t.Fatalf("Expected success with all cached keys in offline mode, got error: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify keys are in correct order
	expectedUsernames := []string{"alice", "bob", "charlie"}
	for i, key := range keys {
		if key.Username != expectedUsernames[i] {
			t.Errorf("Key[%d]: expected username '%s', got '%s'", i, expectedUsernames[i], key.Username)
		}
	}
}

// TestOfflineModeRefreshUser tests that refresh fails in offline mode
func TestOfflineModeRefreshUser(t *testing.T) {
	// Create manager in offline mode
	config := &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create offline manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Should fail - cannot refresh in offline mode
	_, err = manager.RefreshUser(ctx, "alice")
	if err == nil {
		t.Error("Expected error when refreshing in offline mode, got nil")
	}

	// Verify error indicates offline mode
	var apiErr *api.APIError
	if errorAs(err, &apiErr) {
		if apiErr.Kind != api.ErrorKindNetwork {
			t.Errorf("Expected ErrorKindNetwork, got %v", apiErr.Kind)
		}
	}
}

// TestOfflineModeToggle tests switching between online and offline modes
func TestOfflineModeToggle(t *testing.T) {
	// Create manager in online mode
	config := &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Verify starts in online mode
	if manager.IsOfflineMode() {
		t.Error("Expected online mode, got offline mode")
	}

	// Switch to offline mode
	manager.SetOfflineMode(true)
	if !manager.IsOfflineMode() {
		t.Error("Expected offline mode after SetOfflineMode(true)")
	}

	// Switch back to online mode
	manager.SetOfflineMode(false)
	if manager.IsOfflineMode() {
		t.Error("Expected online mode after SetOfflineMode(false)")
	}
}

// TestOfflineModeWithExpiredCache tests offline mode with expired cache entries
func TestOfflineModeWithExpiredCache(t *testing.T) {
	// Create manager in offline mode with very short TTL
	cacheConfig := DefaultCacheConfig()
	cacheConfig.TTL = 1 * time.Millisecond // Very short TTL

	config := &ManagerConfig{
		CacheConfig: cacheConfig,
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create offline manager: %v", err)
	}
	defer manager.Close()

	// Add a key to cache
	testKeyID := "0120abc123"
	testPublicKey := "-----BEGIN PGP PUBLIC KEY BLOCK----- test -----"
	err = manager.Cache().Set("alice", testPublicKey, testKeyID)
	if err != nil {
		t.Fatalf("Failed to cache key: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	ctx := context.Background()

	// Should fail - cache entry has expired
	_, err = manager.GetPublicKey(ctx, "alice")
	if err == nil {
		t.Error("Expected error with expired cache entry in offline mode, got nil")
	}
}

// TestOfflineModeManagerCreation tests creating manager in offline mode
func TestOfflineModeManagerCreation(t *testing.T) {
	tests := []struct {
		name        string
		offlineMode bool
		wantAPIClient bool
	}{
		{
			name:          "online mode creates API client",
			offlineMode:   false,
			wantAPIClient: true,
		},
		{
			name:          "offline mode skips API client",
			offlineMode:   true,
			wantAPIClient: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ManagerConfig{
				CacheConfig: DefaultCacheConfig(),
				APIConfig:   api.DefaultClientConfig(),
				OfflineMode: tt.offlineMode,
			}

			manager, err := NewManager(config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}
			defer manager.Close()

			// Check offline mode status
			if manager.IsOfflineMode() != tt.offlineMode {
				t.Errorf("IsOfflineMode() = %v, want %v", manager.IsOfflineMode(), tt.offlineMode)
			}

			// In offline mode, apiClient should be nil
			hasAPIClient := manager.apiClient != nil
			if hasAPIClient != tt.wantAPIClient {
				t.Errorf("Has API client = %v, want %v", hasAPIClient, tt.wantAPIClient)
			}
		})
	}
}

// errorAs is a helper function to check error types
func errorAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	
	switch t := target.(type) {
	case **api.APIError:
		if apiErr, ok := err.(*api.APIError); ok {
			*t = apiErr
			return true
		}
	}
	
	return false
}

// BenchmarkOfflineModeGetPublicKey benchmarks offline key lookup
func BenchmarkOfflineModeGetPublicKey(b *testing.B) {
	config := &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, _ := NewManager(config)
	defer manager.Close()

	// Populate cache
	_ = manager.Cache().Set("alice", "test-key", "test-id")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetPublicKey(ctx, "alice")
	}
}

// BenchmarkOfflineModeGetPublicKeys benchmarks offline multi-key lookup
func BenchmarkOfflineModeGetPublicKeys(b *testing.B) {
	config := &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
		OfflineMode: true,
	}

	manager, _ := NewManager(config)
	defer manager.Close()

	// Populate cache with multiple users
	users := []string{"alice", "bob", "charlie", "dave", "eve"}
	for _, user := range users {
		_ = manager.Cache().Set(user, "test-key-"+user, "test-id-"+user)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetPublicKeys(ctx, users)
	}
}
