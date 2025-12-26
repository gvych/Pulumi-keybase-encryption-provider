package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
)

// TestIntegrationCacheHit tests that cached keys are returned without API call
func TestIntegrationCacheHit(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		t.Error("API should not be called when key is in cache")
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Pre-populate cache
	err = manager.cache.Set("alice", "cached_key", "cached_kid")
	if err != nil {
		t.Fatalf("cache.Set() failed: %v", err)
	}
	
	// Get key (should use cache)
	key, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() failed: %v", err)
	}
	
	if key.PublicKey != "cached_key" {
		t.Errorf("Expected cached_key, got %s", key.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 0 {
		t.Error("API was called despite cache hit")
	}
}

// TestIntegrationCacheMiss tests that API is called on cache miss and result is cached
func TestIntegrationCacheMiss(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "alice"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "api_kid",
							Bundle: "api_bundle",
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// First call - cache miss, should call API
	key, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() failed: %v", err)
	}
	
	if key.PublicKey != "api_bundle" {
		t.Errorf("Expected api_bundle, got %s", key.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCallCount)
	}
	
	// Second call - should use cache
	key2, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("Second GetPublicKey() failed: %v", err)
	}
	
	if key2.PublicKey != "api_bundle" {
		t.Errorf("Expected api_bundle from cache, got %s", key2.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Error("API was called twice despite second call being cache hit")
	}
}

// TestIntegrationCacheTTLExpiration tests that expired cache entries trigger API call
func TestIntegrationCacheTTLExpiration(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "alice"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "fresh_kid",
							Bundle: "fresh_bundle",
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      100 * time.Millisecond, // Short TTL
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Pre-populate cache
	err = manager.cache.Set("alice", "stale_key", "stale_kid")
	if err != nil {
		t.Fatalf("cache.Set() failed: %v", err)
	}
	
	// Get key immediately (should use cache)
	key1, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() failed: %v", err)
	}
	
	if key1.PublicKey != "stale_key" {
		t.Errorf("Expected stale_key from cache, got %s", key1.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 0 {
		t.Error("API should not be called for fresh cache")
	}
	
	// Wait for cache expiration
	time.Sleep(150 * time.Millisecond)
	
	// Get key after expiration (should call API)
	key2, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() after expiration failed: %v", err)
	}
	
	if key2.PublicKey != "fresh_bundle" {
		t.Errorf("Expected fresh_bundle after expiration, got %s", key2.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Errorf("Expected 1 API call after expiration, got %d", apiCallCount)
	}
}

// TestIntegrationMultipleUsersCacheMix tests mixed cache hits and misses
func TestIntegrationMultipleUsersCacheMix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usernames := r.URL.Query().Get("usernames")
		
		// Should only request bob and charlie (alice is cached)
		if usernames != "bob,charlie" {
			t.Errorf("Expected only bob,charlie to be requested, got %s", usernames)
		}
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "bob"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{KID: "kid_bob", Bundle: "bundle_bob"},
					},
				},
				{
					Basics: api.Basics{Username: "charlie"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{KID: "kid_charlie", Bundle: "bundle_charlie"},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Pre-cache alice
	err = manager.cache.Set("alice", "cached_alice", "cached_kid_alice")
	if err != nil {
		t.Fatalf("cache.Set() failed: %v", err)
	}
	
	// Request all three users
	keys, err := manager.GetPublicKeys(context.Background(), []string{"alice", "bob", "charlie"})
	if err != nil {
		t.Fatalf("GetPublicKeys() failed: %v", err)
	}
	
	if len(keys) != 3 {
		t.Fatalf("Expected 3 keys, got %d", len(keys))
	}
	
	// Verify alice came from cache
	if keys[0].Username != "alice" || keys[0].PublicKey != "cached_alice" {
		t.Errorf("Expected alice from cache, got username=%s key=%s", keys[0].Username, keys[0].PublicKey)
	}
	
	// Verify bob came from API
	if keys[1].Username != "bob" || keys[1].PublicKey != "bundle_bob" {
		t.Errorf("Expected bob from API, got username=%s key=%s", keys[1].Username, keys[1].PublicKey)
	}
	
	// Verify charlie came from API
	if keys[2].Username != "charlie" || keys[2].PublicKey != "bundle_charlie" {
		t.Errorf("Expected charlie from API, got username=%s key=%s", keys[2].Username, keys[2].PublicKey)
	}
	
	// Verify bob and charlie are now cached
	bobCached := manager.cache.Get("bob")
	if bobCached == nil || bobCached.PublicKey != "bundle_bob" {
		t.Error("bob should be cached after API fetch")
	}
	
	charlieCached := manager.cache.Get("charlie")
	if charlieCached == nil || charlieCached.PublicKey != "bundle_charlie" {
		t.Error("charlie should be cached after API fetch")
	}
}

// TestIntegrationCacheInvalidation tests cache invalidation and refresh
func TestIntegrationCacheInvalidation(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "alice"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{KID: "new_kid", Bundle: "new_bundle"},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Pre-cache with old key
	err = manager.cache.Set("alice", "old_bundle", "old_kid")
	if err != nil {
		t.Fatalf("cache.Set() failed: %v", err)
	}
	
	// Invalidate cache
	err = manager.InvalidateUser("alice")
	if err != nil {
		t.Fatalf("InvalidateUser() failed: %v", err)
	}
	
	// Get key (should call API since cache was invalidated)
	key, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() failed: %v", err)
	}
	
	if key.PublicKey != "new_bundle" {
		t.Errorf("Expected new_bundle after invalidation, got %s", key.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Errorf("Expected 1 API call after invalidation, got %d", apiCallCount)
	}
}

// TestIntegrationCacheRefresh tests forced refresh of cached keys
func TestIntegrationCacheRefresh(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "alice"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{KID: "refreshed_kid", Bundle: "refreshed_bundle"},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Pre-cache with old key
	err = manager.cache.Set("alice", "old_bundle", "old_kid")
	if err != nil {
		t.Fatalf("cache.Set() failed: %v", err)
	}
	
	// Force refresh
	key, err := manager.RefreshUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("RefreshUser() failed: %v", err)
	}
	
	if key.PublicKey != "refreshed_bundle" {
		t.Errorf("Expected refreshed_bundle, got %s", key.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Errorf("Expected 1 API call for refresh, got %d", apiCallCount)
	}
	
	// Verify cache now has refreshed key
	cached := manager.cache.Get("alice")
	if cached == nil || cached.PublicKey != "refreshed_bundle" {
		t.Error("Cache should contain refreshed key")
	}
}

// TestIntegrationCachePersistence tests that cache persists across manager instances
func TestIntegrationCachePersistence(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "alice"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{KID: "kid", Bundle: "bundle"},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")
	
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: cacheFile,
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	// First manager instance
	manager1, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Fetch key (will cache it)
	_, err = manager1.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() failed: %v", err)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCallCount)
	}
	
	// Create second manager instance (should load from disk)
	manager2, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() for second instance failed: %v", err)
	}
	
	// Fetch same key (should use persisted cache, no API call)
	key, err := manager2.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() on second manager failed: %v", err)
	}
	
	if key.PublicKey != "bundle" {
		t.Errorf("Expected bundle from persisted cache, got %s", key.PublicKey)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Error("Second manager should use persisted cache, not call API")
	}
}

// TestIntegrationCacheAPIError tests that API errors don't corrupt cache
func TestIntegrationCacheAPIError(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&apiCallCount, 1)
		
		if count == 1 {
			// First call succeeds
			response := api.LookupResponse{
				Status: api.Status{Code: 0, Name: "OK"},
				Them: []api.User{
					{
						Basics: api.Basics{Username: "alice"},
						PublicKeys: api.PublicKeys{
							Primary: api.PrimaryKey{KID: "kid", Bundle: "bundle"},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			// Subsequent calls fail
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// First call succeeds and caches
	_, err = manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("First GetPublicKey() failed: %v", err)
	}
	
	// Invalidate to force API call
	manager.InvalidateUser("alice")
	
	// Second call fails (API error)
	_, err = manager.GetPublicKey(context.Background(), "alice")
	if err == nil {
		t.Fatal("Expected error from API, got nil")
	}
	
	// Verify cache is not corrupted (should be empty)
	cached := manager.cache.Get("alice")
	if cached != nil {
		t.Error("Cache should be empty after failed API call")
	}
}

// TestIntegrationCacheStats tests cache statistics during operations
func TestIntegrationCacheStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them: []api.User{
				{
					Basics: api.Basics{Username: "alice"},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{KID: "kid_alice", Bundle: "bundle_alice"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      100 * time.Millisecond,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Initial stats
	stats := manager.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Initial TotalEntries should be 0, got %d", stats.TotalEntries)
	}
	
	// Fetch key (will cache it)
	_, err = manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() failed: %v", err)
	}
	
	// Check stats after caching
	stats = manager.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("TotalEntries should be 1, got %d", stats.TotalEntries)
	}
	if stats.ValidEntries != 1 {
		t.Errorf("ValidEntries should be 1, got %d", stats.ValidEntries)
	}
	if stats.ExpiredEntries != 0 {
		t.Errorf("ExpiredEntries should be 0, got %d", stats.ExpiredEntries)
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Check stats after expiration
	stats = manager.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("TotalEntries should still be 1, got %d", stats.TotalEntries)
	}
	if stats.ExpiredEntries != 1 {
		t.Errorf("ExpiredEntries should be 1, got %d", stats.ExpiredEntries)
	}
	if stats.ValidEntries != 0 {
		t.Errorf("ValidEntries should be 0, got %d", stats.ValidEntries)
	}
}

// TestIntegrationBatchCachingEfficiency tests that batch operations minimize API calls
func TestIntegrationBatchCachingEfficiency(t *testing.T) {
	apiCallCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCallCount, 1)
		
		// Return keys for all 10 users
		users := []api.User{}
		for i := 0; i < 10; i++ {
			users = append(users, api.User{
				Basics: api.Basics{Username: fmt.Sprintf("user%d", i)},
				PublicKeys: api.PublicKeys{
					Primary: api.PrimaryKey{
						KID:    fmt.Sprintf("kid_%d", i),
						Bundle: fmt.Sprintf("bundle_%d", i),
					},
				},
			})
		}
		
		response := api.LookupResponse{
			Status: api.Status{Code: 0, Name: "OK"},
			Them:   users,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		},
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	
	// Fetch 10 users
	usernames := make([]string, 10)
	for i := 0; i < 10; i++ {
		usernames[i] = fmt.Sprintf("user%d", i)
	}
	
	_, err = manager.GetPublicKeys(context.Background(), usernames)
	if err != nil {
		t.Fatalf("GetPublicKeys() failed: %v", err)
	}
	
	// Should make exactly 1 API call (batch lookup)
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Errorf("Expected 1 batch API call, got %d", apiCallCount)
	}
	
	// Fetch same users again (should use cache, no API calls)
	_, err = manager.GetPublicKeys(context.Background(), usernames)
	if err != nil {
		t.Fatalf("Second GetPublicKeys() failed: %v", err)
	}
	
	if atomic.LoadInt32(&apiCallCount) != 1 {
		t.Error("Second batch lookup should use cache, not call API")
	}
	
	// Verify all users are cached
	stats := manager.Stats()
	if stats.ValidEntries != 10 {
		t.Errorf("Expected 10 cached entries, got %d", stats.ValidEntries)
	}
}
