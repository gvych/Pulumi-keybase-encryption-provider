package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached public key with expiration
type CacheEntry struct {
	Username   string    `json:"username"`
	PublicKey  string    `json:"public_key"`
	KeyID      string    `json:"key_id"`
	FetchedAt  time.Time `json:"fetched_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache represents the public key cache
type Cache struct {
	FilePath string                 `json:"-"`
	Entries  map[string]*CacheEntry `json:"entries"`
	TTL      time.Duration          `json:"-"`
	mu       sync.RWMutex
}

// CacheConfig holds configuration for the cache
type CacheConfig struct {
	// FilePath is the path to the cache file
	// Defaults to ~/.config/pulumi/keybase_keyring_cache.json
	FilePath string
	
	// TTL is the time-to-live for cache entries
	// Defaults to 24 hours
	TTL time.Duration
}

// DefaultCacheConfig returns the default cache configuration
func DefaultCacheConfig() *CacheConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to /tmp if home directory is not available
		homeDir = "/tmp"
	}
	
	return &CacheConfig{
		FilePath: filepath.Join(homeDir, ".config", "pulumi", "keybase_keyring_cache.json"),
		TTL:      24 * time.Hour,
	}
}

// NewCache creates a new cache instance
func NewCache(config *CacheConfig) (*Cache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}
	
	cache := &Cache{
		FilePath: config.FilePath,
		Entries:  make(map[string]*CacheEntry),
		TTL:      config.TTL,
	}
	
	// Ensure cache directory exists
	cacheDir := filepath.Dir(cache.FilePath)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	// Load existing cache if it exists
	if err := cache.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}
	
	return cache, nil
}

// Get retrieves a public key from the cache
// Returns nil if the key is not found or has expired
func (c *Cache) Get(username string) *CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.Entries[username]
	if !exists {
		return nil
	}
	
	if entry.IsExpired() {
		return nil
	}
	
	return entry
}

// Set stores a public key in the cache
func (c *Cache) Set(username, publicKey, keyID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	c.Entries[username] = &CacheEntry{
		Username:  username,
		PublicKey: publicKey,
		KeyID:     keyID,
		FetchedAt: now,
		ExpiresAt: now.Add(c.TTL),
	}
	
	return c.save()
}

// Delete removes a cache entry for the given username
func (c *Cache) Delete(username string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.Entries, username)
	return c.save()
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.Entries = make(map[string]*CacheEntry)
	return c.save()
}

// PruneExpired removes all expired entries from the cache
func (c *Cache) PruneExpired() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	modified := false
	for username, entry := range c.Entries {
		if entry.IsExpired() {
			delete(c.Entries, username)
			modified = true
		}
	}
	
	if modified {
		return c.save()
	}
	
	return nil
}

// Load reads the cache from disk
func (c *Cache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	data, err := os.ReadFile(c.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}
	
	var diskCache struct {
		Entries map[string]*CacheEntry `json:"entries"`
	}
	
	if err := json.Unmarshal(data, &diskCache); err != nil {
		return fmt.Errorf("failed to parse cache file: %w", err)
	}
	
	c.Entries = diskCache.Entries
	if c.Entries == nil {
		c.Entries = make(map[string]*CacheEntry)
	}
	
	return nil
}

// save writes the cache to disk (internal, must be called with lock held)
func (c *Cache) save() error {
	diskCache := struct {
		Entries map[string]*CacheEntry `json:"entries"`
	}{
		Entries: c.Entries,
	}
	
	data, err := json.MarshalIndent(diskCache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}
	
	// Write to temporary file first, then rename for atomic operation
	tmpFile := c.FilePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	
	if err := os.Rename(tmpFile, c.FilePath); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename cache file: %w", err)
	}
	
	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := CacheStats{
		TotalEntries:   len(c.Entries),
		ExpiredEntries: 0,
		ValidEntries:   0,
	}
	
	for _, entry := range c.Entries {
		if entry.IsExpired() {
			stats.ExpiredEntries++
		} else {
			stats.ValidEntries++
		}
	}
	
	return stats
}

// CacheStats holds statistics about the cache
type CacheStats struct {
	TotalEntries   int
	ValidEntries   int
	ExpiredEntries int
}
