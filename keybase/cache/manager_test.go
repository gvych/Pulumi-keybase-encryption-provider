package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: api.DefaultClientConfig(),
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()
	
	if config.CacheConfig == nil {
		t.Error("CacheConfig is nil")
	}
	
	if config.APIConfig == nil {
		t.Error("APIConfig is nil")
	}
}

func TestGetPublicKeyFromAPI(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.LookupResponse{
			Status: api.Status{
				Code: 0,
				Name: "OK",
			},
			Them: []api.User{
				{
					Basics: api.Basics{
						Username: "alice",
					},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "test_kid_alice",
							Bundle: "test_bundle_alice",
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
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
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
		t.Fatalf("NewManager() error = %v", err)
	}
	
	key, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() error = %v", err)
	}
	
	if key.Username != "alice" {
		t.Errorf("Username = %v, want alice", key.Username)
	}
	
	if key.PublicKey != "test_bundle_alice" {
		t.Errorf("PublicKey = %v, want test_bundle_alice", key.PublicKey)
	}
}

func TestGetPublicKeyFromCache(t *testing.T) {
	// Create mock server that should not be called
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		t.Error("API should not be called when key is in cache")
	}))
	defer server.Close()
	
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
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
		t.Fatalf("NewManager() error = %v", err)
	}
	
	// Pre-populate cache
	err = manager.cache.Set("alice", "cached_key", "cached_kid")
	if err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	
	// Get from cache
	key, err := manager.GetPublicKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetPublicKey() error = %v", err)
	}
	
	if key.PublicKey != "cached_key" {
		t.Errorf("PublicKey = %v, want cached_key", key.PublicKey)
	}
	
	if callCount > 0 {
		t.Error("API was called when it should have used cache")
	}
}

func TestGetPublicKeysMultiple(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.LookupResponse{
			Status: api.Status{
				Code: 0,
				Name: "OK",
			},
			Them: []api.User{
				{
					Basics: api.Basics{
						Username: "alice",
					},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "kid_alice",
							Bundle: "bundle_alice",
						},
					},
				},
				{
					Basics: api.Basics{
						Username: "bob",
					},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "kid_bob",
							Bundle: "bundle_bob",
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
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
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
		t.Fatalf("NewManager() error = %v", err)
	}
	
	keys, err := manager.GetPublicKeys(context.Background(), []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("GetPublicKeys() error = %v", err)
	}
	
	if len(keys) != 2 {
		t.Fatalf("GetPublicKeys() returned %d keys, want 2", len(keys))
	}
	
	if keys[0].Username != "alice" {
		t.Errorf("keys[0].Username = %v, want alice", keys[0].Username)
	}
	
	if keys[1].Username != "bob" {
		t.Errorf("keys[1].Username = %v, want bob", keys[1].Username)
	}
}

func TestGetPublicKeysMixedCache(t *testing.T) {
	// Create mock server that returns only bob (alice is cached)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that only bob is requested
		if r.URL.Query().Get("usernames") != "bob" {
			t.Errorf("Expected only bob to be requested, got: %s", r.URL.Query().Get("usernames"))
		}
		
		response := api.LookupResponse{
			Status: api.Status{
				Code: 0,
				Name: "OK",
			},
			Them: []api.User{
				{
					Basics: api.Basics{
						Username: "bob",
					},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "kid_bob",
							Bundle: "bundle_bob",
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
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
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
		t.Fatalf("NewManager() error = %v", err)
	}
	
	// Pre-populate cache with alice
	err = manager.cache.Set("alice", "cached_alice", "cached_kid_alice")
	if err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	
	// Request both alice and bob
	keys, err := manager.GetPublicKeys(context.Background(), []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("GetPublicKeys() error = %v", err)
	}
	
	if len(keys) != 2 {
		t.Fatalf("GetPublicKeys() returned %d keys, want 2", len(keys))
	}
	
	// Verify alice came from cache
	if keys[0].PublicKey != "cached_alice" {
		t.Errorf("keys[0].PublicKey = %v, want cached_alice", keys[0].PublicKey)
	}
	
	// Verify bob came from API
	if keys[1].PublicKey != "bundle_bob" {
		t.Errorf("keys[1].PublicKey = %v, want bundle_bob", keys[1].PublicKey)
	}
}

func TestInvalidateUser(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: api.DefaultClientConfig(),
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	
	// Add to cache
	err = manager.cache.Set("alice", "key", "kid")
	if err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	
	// Verify it's there
	if manager.cache.Get("alice") == nil {
		t.Fatal("Entry not found after Set()")
	}
	
	// Invalidate
	err = manager.InvalidateUser("alice")
	if err != nil {
		t.Fatalf("InvalidateUser() error = %v", err)
	}
	
	// Verify it's gone
	if manager.cache.Get("alice") != nil {
		t.Error("Entry still exists after InvalidateUser()")
	}
}

func TestInvalidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: api.DefaultClientConfig(),
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	
	// Add multiple entries
	for _, user := range []string{"alice", "bob", "charlie"} {
		err = manager.cache.Set(user, "key_"+user, "kid_"+user)
		if err != nil {
			t.Fatalf("cache.Set() error = %v", err)
		}
	}
	
	// Invalidate all
	err = manager.InvalidateAll()
	if err != nil {
		t.Fatalf("InvalidateAll() error = %v", err)
	}
	
	// Verify all gone
	stats := manager.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("TotalEntries = %v, want 0", stats.TotalEntries)
	}
}

func TestPruneExpired(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
			TTL:      100 * time.Millisecond,
		},
		APIConfig: api.DefaultClientConfig(),
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	
	// Add entry with short TTL
	err = manager.cache.Set("alice", "key", "kid")
	if err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Add another entry (not expired)
	manager.cache.TTL = 1 * time.Hour
	err = manager.cache.Set("bob", "key", "kid")
	if err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	
	// Prune
	err = manager.PruneExpired()
	if err != nil {
		t.Fatalf("PruneExpired() error = %v", err)
	}
	
	// Should have only bob left
	stats := manager.Stats()
	if stats.ValidEntries != 1 {
		t.Errorf("ValidEntries = %v, want 1", stats.ValidEntries)
	}
}

func TestRefreshUser(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.LookupResponse{
			Status: api.Status{
				Code: 0,
				Name: "OK",
			},
			Them: []api.User{
				{
					Basics: api.Basics{
						Username: "alice",
					},
					PublicKeys: api.PublicKeys{
						Primary: api.PrimaryKey{
							KID:    "new_kid",
							Bundle: "new_bundle",
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
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
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
		t.Fatalf("NewManager() error = %v", err)
	}
	
	// Pre-populate cache with old key
	err = manager.cache.Set("alice", "old_bundle", "old_kid")
	if err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	
	// Refresh
	key, err := manager.RefreshUser(context.Background(), "alice")
	if err != nil {
		t.Fatalf("RefreshUser() error = %v", err)
	}
	
	// Should have new key
	if key.PublicKey != "new_bundle" {
		t.Errorf("PublicKey = %v, want new_bundle", key.PublicKey)
	}
}

func TestManagerClose(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ManagerConfig{
		CacheConfig: &CacheConfig{
			FilePath: filepath.Join(tmpDir, "test_cache.json"),
			TTL:      1 * time.Hour,
		},
		APIConfig: api.DefaultClientConfig(),
	}
	
	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	
	err = manager.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
