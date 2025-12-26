package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	if cache.FilePath != config.FilePath {
		t.Errorf("FilePath = %v, want %v", cache.FilePath, config.FilePath)
	}
	
	if cache.TTL != config.TTL {
		t.Errorf("TTL = %v, want %v", cache.TTL, config.TTL)
	}
}

func TestCacheSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Test Set
	username := "alice"
	publicKey := "test_public_key"
	keyID := "test_key_id"
	
	err = cache.Set(username, publicKey, keyID)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Test Get
	entry := cache.Get(username)
	if entry == nil {
		t.Fatal("Get() returned nil, want entry")
	}
	
	if entry.Username != username {
		t.Errorf("Username = %v, want %v", entry.Username, username)
	}
	
	if entry.PublicKey != publicKey {
		t.Errorf("PublicKey = %v, want %v", entry.PublicKey, publicKey)
	}
	
	if entry.KeyID != keyID {
		t.Errorf("KeyID = %v, want %v", entry.KeyID, keyID)
	}
}

func TestCacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      100 * time.Millisecond,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	username := "bob"
	err = cache.Set(username, "key", "id")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Should be available immediately
	entry := cache.Get(username)
	if entry == nil {
		t.Fatal("Get() returned nil immediately after Set()")
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Should be expired now
	entry = cache.Get(username)
	if entry != nil {
		t.Error("Get() returned entry after expiration, want nil")
	}
}

func TestCacheDelete(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	username := "charlie"
	err = cache.Set(username, "key", "id")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Verify it's there
	if cache.Get(username) == nil {
		t.Fatal("Get() returned nil before delete")
	}
	
	// Delete
	err = cache.Delete(username)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	
	// Verify it's gone
	if cache.Get(username) != nil {
		t.Error("Get() returned entry after delete, want nil")
	}
}

func TestCacheClear(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Add multiple entries
	users := []string{"alice", "bob", "charlie"}
	for _, user := range users {
		err = cache.Set(user, "key_"+user, "id_"+user)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}
	
	// Verify they're there
	stats := cache.Stats()
	if stats.TotalEntries != len(users) {
		t.Errorf("TotalEntries = %v, want %v", stats.TotalEntries, len(users))
	}
	
	// Clear
	err = cache.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	
	// Verify all gone
	stats = cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("TotalEntries after Clear() = %v, want 0", stats.TotalEntries)
	}
}

func TestCachePruneExpired(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      100 * time.Millisecond,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Add entries
	err = cache.Set("alice", "key1", "id1")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Add another entry (not expired)
	config.TTL = 1 * time.Hour
	cache.TTL = 1 * time.Hour
	err = cache.Set("bob", "key2", "id2")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Should have 2 entries total (1 expired, 1 valid)
	stats := cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("TotalEntries = %v, want 2", stats.TotalEntries)
	}
	
	// Prune expired
	err = cache.PruneExpired()
	if err != nil {
		t.Fatalf("PruneExpired() error = %v", err)
	}
	
	// Should have 1 entry left
	stats = cache.Stats()
	if stats.TotalEntries != 1 {
		t.Errorf("TotalEntries after prune = %v, want 1", stats.TotalEntries)
	}
	
	if stats.ValidEntries != 1 {
		t.Errorf("ValidEntries after prune = %v, want 1", stats.ValidEntries)
	}
}

func TestCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "test_cache.json")
	config := &CacheConfig{
		FilePath: cacheFile,
		TTL:      1 * time.Hour,
	}
	
	// Create cache and add entry
	cache1, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	username := "dave"
	publicKey := "persistent_key"
	keyID := "persistent_id"
	
	err = cache1.Set(username, publicKey, keyID)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}
	
	// Create new cache instance (should load from disk)
	cache2, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() for second instance error = %v", err)
	}
	
	// Verify entry is loaded
	entry := cache2.Get(username)
	if entry == nil {
		t.Fatal("Get() returned nil after reload, want entry")
	}
	
	if entry.PublicKey != publicKey {
		t.Errorf("PublicKey after reload = %v, want %v", entry.PublicKey, publicKey)
	}
}

func TestCacheStats(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      100 * time.Millisecond,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Add entries with short TTL
	err = cache.Set("alice", "key1", "id1")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	err = cache.Set("bob", "key2", "id2")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Check initial stats
	stats := cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("TotalEntries = %v, want 2", stats.TotalEntries)
	}
	if stats.ValidEntries != 2 {
		t.Errorf("ValidEntries = %v, want 2", stats.ValidEntries)
	}
	if stats.ExpiredEntries != 0 {
		t.Errorf("ExpiredEntries = %v, want 0", stats.ExpiredEntries)
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Check stats after expiration
	stats = cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("TotalEntries after expiration = %v, want 2", stats.TotalEntries)
	}
	if stats.ExpiredEntries != 2 {
		t.Errorf("ExpiredEntries = %v, want 2", stats.ExpiredEntries)
	}
	if stats.ValidEntries != 0 {
		t.Errorf("ValidEntries = %v, want 0", stats.ValidEntries)
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			username := string(rune('a' + id))
			err := cache.Set(username, "key", "id")
			if err != nil {
				t.Errorf("Concurrent Set() error = %v", err)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			username := string(rune('a' + id))
			entry := cache.Get(username)
			if entry == nil {
				t.Errorf("Concurrent Get(%s) returned nil", username)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()
	
	if config.FilePath == "" {
		t.Error("Default FilePath is empty")
	}
	
	if config.TTL != 24*time.Hour {
		t.Errorf("Default TTL = %v, want %v", config.TTL, 24*time.Hour)
	}
}

func TestCacheEntryIsExpired(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "future expiration",
			expiresAt: now.Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "past expiration",
			expiresAt: now.Add(-1 * time.Hour),
			want:      true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &CacheEntry{
				ExpiresAt: tt.expiresAt,
			}
			
			if got := entry.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCacheWithInvalidDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a file where we'll try to create a directory
	invalidPath := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(invalidPath, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	
	// Try to create cache in a path that requires a directory where a file exists
	config := &CacheConfig{
		FilePath: filepath.Join(invalidPath, "subdir", "cache.json"),
		TTL:      1 * time.Hour,
	}
	
	_, err = NewCache(config)
	if err == nil {
		t.Error("NewCache() should fail when directory path contains a file")
	}
}

func TestLoadCorruptedCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "corrupted_cache.json")
	
	// Write corrupted JSON
	err := os.WriteFile(cacheFile, []byte("invalid json {{{"), 0600)
	if err != nil {
		t.Fatalf("Failed to write corrupted cache file: %v", err)
	}
	
	config := &CacheConfig{
		FilePath: cacheFile,
		TTL:      1 * time.Hour,
	}
	
	_, err = NewCache(config)
	if err == nil {
		t.Error("NewCache() should fail with corrupted cache file")
	}
}

func TestLoadEmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "empty_cache.json")
	
	// Write empty JSON object
	err := os.WriteFile(cacheFile, []byte(`{"entries":null}`), 0600)
	if err != nil {
		t.Fatalf("Failed to write empty cache file: %v", err)
	}
	
	config := &CacheConfig{
		FilePath: cacheFile,
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() failed with empty cache: %v", err)
	}
	
	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("TotalEntries = %v, want 0", stats.TotalEntries)
	}
}

func TestSaveErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "test_cache.json")
	
	config := &CacheConfig{
		FilePath: cacheFile,
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Add an entry
	err = cache.Set("test", "key", "id")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Make directory read-only to cause save error
	err = os.Chmod(tmpDir, 0500)
	if err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}
	
	// Restore permissions after test
	defer os.Chmod(tmpDir, 0700)
	
	// Try to set another entry (should fail to save)
	err = cache.Set("another", "key2", "id2")
	if err == nil {
		t.Error("Set() should fail when cache file cannot be written")
	}
}

func TestPruneExpiredNoModification(t *testing.T) {
	tmpDir := t.TempDir()
	config := &CacheConfig{
		FilePath: filepath.Join(tmpDir, "test_cache.json"),
		TTL:      1 * time.Hour,
	}
	
	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}
	
	// Add entries that won't expire
	err = cache.Set("alice", "key1", "id1")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Prune when nothing is expired (should not modify)
	err = cache.PruneExpired()
	if err != nil {
		t.Fatalf("PruneExpired() error = %v", err)
	}
	
	stats := cache.Stats()
	if stats.ValidEntries != 1 {
		t.Errorf("ValidEntries = %v, want 1", stats.ValidEntries)
	}
}
