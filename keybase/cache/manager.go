package cache

import (
	"context"
	"fmt"
	"sync"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
)

// Manager manages public key caching with API integration
type Manager struct {
	cache     *Cache
	apiClient *api.Client
	mu        sync.RWMutex
}

// ManagerConfig holds configuration for the cache manager
type ManagerConfig struct {
	CacheConfig *CacheConfig
	APIConfig   *api.ClientConfig
}

// DefaultManagerConfig returns the default cache manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		CacheConfig: DefaultCacheConfig(),
		APIConfig:   api.DefaultClientConfig(),
	}
}

// NewManager creates a new cache manager
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config == nil {
		config = DefaultManagerConfig()
	}
	
	cache, err := NewCache(config.CacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	
	apiClient := api.NewClient(config.APIConfig)
	
	return &Manager{
		cache:     cache,
		apiClient: apiClient,
	}, nil
}

// GetPublicKey retrieves a public key for a username, using cache if available
func (m *Manager) GetPublicKey(ctx context.Context, username string) (*api.UserPublicKey, error) {
	// Check cache first
	if entry := m.cache.Get(username); entry != nil {
		return &api.UserPublicKey{
			Username:  entry.Username,
			PublicKey: entry.PublicKey,
			KeyID:     entry.KeyID,
		}, nil
	}
	
	// Fetch from API
	keys, err := m.apiClient.LookupUsers(ctx, []string{username})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public key for %s: %w", username, err)
	}
	
	if len(keys) == 0 {
		return nil, fmt.Errorf("no public key found for user: %s", username)
	}
	
	key := keys[0]
	
	// Store in cache
	if err := m.cache.Set(key.Username, key.PublicKey, key.KeyID); err != nil {
		// Log error but don't fail the operation
		// The key was fetched successfully, caching is just an optimization
	}
	
	return &key, nil
}

// GetPublicKeys retrieves public keys for multiple usernames
// Uses batch API call for efficiency, with cache fallback per user
func (m *Manager) GetPublicKeys(ctx context.Context, usernames []string) ([]api.UserPublicKey, error) {
	if len(usernames) == 0 {
		return nil, fmt.Errorf("no usernames provided")
	}
	
	// Check which users need to be fetched from API
	var needFetch []string
	results := make([]api.UserPublicKey, 0, len(usernames))
	resultMap := make(map[string]*api.UserPublicKey)
	
	for _, username := range usernames {
		if entry := m.cache.Get(username); entry != nil {
			key := &api.UserPublicKey{
				Username:  entry.Username,
				PublicKey: entry.PublicKey,
				KeyID:     entry.KeyID,
			}
			resultMap[username] = key
		} else {
			needFetch = append(needFetch, username)
		}
	}
	
	// Fetch missing users from API
	if len(needFetch) > 0 {
		keys, err := m.apiClient.LookupUsers(ctx, needFetch)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch public keys: %w", err)
		}
		
		// Cache fetched keys
		for _, key := range keys {
			if err := m.cache.Set(key.Username, key.PublicKey, key.KeyID); err != nil {
				// Log error but continue
			}
			resultMap[key.Username] = &api.UserPublicKey{
				Username:  key.Username,
				PublicKey: key.PublicKey,
				KeyID:     key.KeyID,
			}
		}
	}
	
	// Build results in original order
	for _, username := range usernames {
		if key, ok := resultMap[username]; ok {
			results = append(results, *key)
		} else {
			return nil, fmt.Errorf("no public key found for user: %s", username)
		}
	}
	
	return results, nil
}

// InvalidateUser removes a user's public key from the cache
// Useful when key rotation is detected
func (m *Manager) InvalidateUser(username string) error {
	return m.cache.Delete(username)
}

// InvalidateAll clears the entire cache
func (m *Manager) InvalidateAll() error {
	return m.cache.Clear()
}

// PruneExpired removes expired entries from the cache
func (m *Manager) PruneExpired() error {
	return m.cache.PruneExpired()
}

// Stats returns cache statistics
func (m *Manager) Stats() CacheStats {
	return m.cache.Stats()
}

// Close releases resources held by the manager
func (m *Manager) Close() error {
	// Currently no resources to release
	// Future: close HTTP client, database connections, etc.
	return nil
}

// Cache returns the underlying cache instance
// This is useful for direct cache operations when needed
func (m *Manager) Cache() *Cache {
	return m.cache
}

// RefreshUser forces a refresh of a user's public key from the API
func (m *Manager) RefreshUser(ctx context.Context, username string) (*api.UserPublicKey, error) {
	// Invalidate cache entry
	if err := m.cache.Delete(username); err != nil {
		// Log but continue
	}
	
	// Fetch fresh from API
	return m.GetPublicKey(ctx, username)
}

// RefreshUsers forces a refresh of multiple users' public keys from the API
func (m *Manager) RefreshUsers(ctx context.Context, usernames []string) ([]api.UserPublicKey, error) {
	// Invalidate cache entries
	for _, username := range usernames {
		if err := m.cache.Delete(username); err != nil {
			// Log but continue
		}
	}
	
	// Fetch fresh from API
	return m.GetPublicKeys(ctx, usernames)
}
