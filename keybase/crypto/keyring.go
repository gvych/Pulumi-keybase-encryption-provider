package crypto

import (
	"fmt"
	"sync"
	"time"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
)

// KeyringLoader loads and caches Keybase secret keys from the local configuration
type KeyringLoader struct {
	// mu protects the cache
	mu sync.RWMutex
	
	// cache stores loaded keys with their expiration time
	cache map[string]*cachedKey
	
	// ttl is the time-to-live for cached keys
	ttl time.Duration
	
	// configDir is the Keybase configuration directory
	configDir string
}

// cachedKey represents a cached secret key with expiration
type cachedKey struct {
	secretKey  saltpack.BoxSecretKey
	publicKey  saltpack.BoxPublicKey
	keyID      []byte
	loadedAt   time.Time
	expiresAt  time.Time
}

// KeyringLoaderConfig holds configuration for KeyringLoader
type KeyringLoaderConfig struct {
	// TTL is the time-to-live for cached keys
	// Default: 1 hour
	TTL time.Duration
	
	// ConfigDir is the Keybase configuration directory
	// If empty, uses the default directory from credentials package
	ConfigDir string
}

// NewKeyringLoader creates a new KeyringLoader
func NewKeyringLoader(config *KeyringLoaderConfig) (*KeyringLoader, error) {
	if config == nil {
		config = &KeyringLoaderConfig{}
	}
	
	// Set default TTL
	ttl := config.TTL
	if ttl == 0 {
		ttl = 1 * time.Hour // Default: 1 hour cache
	}
	
	// Get config directory
	configDir := config.ConfigDir
	if configDir == "" {
		status, err := credentials.DiscoverCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed to discover Keybase configuration: %w", err)
		}
		configDir = status.ConfigDir
	}
	
	return &KeyringLoader{
		cache:     make(map[string]*cachedKey),
		ttl:       ttl,
		configDir: configDir,
	}, nil
}

// LoadKeyring loads a keyring with the current user's secret key
// The keyring is cached in memory for the configured TTL
func (kl *KeyringLoader) LoadKeyring() (saltpack.Keyring, error) {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	// Get current username
	username, err := credentials.GetUsername()
	if err != nil {
		return nil, fmt.Errorf("failed to get current username: %w", err)
	}
	
	// Check if we have a cached key that's still valid
	if cached, ok := kl.cache[username]; ok {
		if time.Now().Before(cached.expiresAt) {
			// Cache hit - create keyring with cached key
			keyring := NewSimpleKeyring()
			keyring.AddKey(cached.secretKey)
			return keyring, nil
		}
		// Cache expired - remove it
		delete(kl.cache, username)
	}
	
	// Cache miss or expired - load key from disk
	secretKey, err := loadPrivateKey(kl.configDir, username)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key for user '%s': %w", username, err)
	}
	
	// Validate the key
	if err := ValidateSecretKey(secretKey); err != nil {
		return nil, fmt.Errorf("invalid secret key for user '%s': %w", username, err)
	}
	
	// Get public key and key ID
	publicKey := secretKey.GetPublicKey()
	if publicKey == nil {
		return nil, fmt.Errorf("failed to derive public key from secret key for user '%s'", username)
	}
	
	keyID := publicKey.ToKID()
	
	// Cache the key
	now := time.Now()
	kl.cache[username] = &cachedKey{
		secretKey: secretKey,
		publicKey: publicKey,
		keyID:     keyID,
		loadedAt:  now,
		expiresAt: now.Add(kl.ttl),
	}
	
	// Create keyring with the loaded key
	keyring := NewSimpleKeyring()
	keyring.AddKey(secretKey)
	
	return keyring, nil
}

// LoadKeyringForUser loads a keyring with a specific user's secret key
// This is useful when you need to decrypt as a different user
func (kl *KeyringLoader) LoadKeyringForUser(username string) (saltpack.Keyring, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	// Check if we have a cached key that's still valid
	if cached, ok := kl.cache[username]; ok {
		if time.Now().Before(cached.expiresAt) {
			// Cache hit - create keyring with cached key
			keyring := NewSimpleKeyring()
			keyring.AddKey(cached.secretKey)
			return keyring, nil
		}
		// Cache expired - remove it
		delete(kl.cache, username)
	}
	
	// Cache miss or expired - load key from disk
	secretKey, err := loadPrivateKey(kl.configDir, username)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key for user '%s': %w", username, err)
	}
	
	// Validate the key
	if err := ValidateSecretKey(secretKey); err != nil {
		return nil, fmt.Errorf("invalid secret key for user '%s': %w", username, err)
	}
	
	// Get public key and key ID
	publicKey := secretKey.GetPublicKey()
	if publicKey == nil {
		return nil, fmt.Errorf("failed to derive public key from secret key for user '%s'", username)
	}
	
	keyID := publicKey.ToKID()
	
	// Cache the key
	now := time.Now()
	kl.cache[username] = &cachedKey{
		secretKey: secretKey,
		publicKey: publicKey,
		keyID:     keyID,
		loadedAt:  now,
		expiresAt: now.Add(kl.ttl),
	}
	
	// Create keyring with the loaded key
	keyring := NewSimpleKeyring()
	keyring.AddKey(secretKey)
	
	return keyring, nil
}

// GetSecretKey retrieves a secret key for a specific user (with caching)
func (kl *KeyringLoader) GetSecretKey(username string) (saltpack.BoxSecretKey, error) {
	if username == "" {
		var err error
		username, err = credentials.GetUsername()
		if err != nil {
			return nil, fmt.Errorf("failed to get current username: %w", err)
		}
	}
	
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	// Check cache
	if cached, ok := kl.cache[username]; ok {
		if time.Now().Before(cached.expiresAt) {
			return cached.secretKey, nil
		}
		delete(kl.cache, username)
	}
	
	// Load from disk
	secretKey, err := loadPrivateKey(kl.configDir, username)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key for user '%s': %w", username, err)
	}
	
	// Validate
	if err := ValidateSecretKey(secretKey); err != nil {
		return nil, fmt.Errorf("invalid secret key for user '%s': %w", username, err)
	}
	
	// Cache it
	publicKey := secretKey.GetPublicKey()
	keyID := publicKey.ToKID()
	now := time.Now()
	kl.cache[username] = &cachedKey{
		secretKey: secretKey,
		publicKey: publicKey,
		keyID:     keyID,
		loadedAt:  now,
		expiresAt: now.Add(kl.ttl),
	}
	
	return secretKey, nil
}

// InvalidateCache removes cached keys, forcing a reload on next access
func (kl *KeyringLoader) InvalidateCache() {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	kl.cache = make(map[string]*cachedKey)
}

// InvalidateCacheForUser removes a specific user's cached key
func (kl *KeyringLoader) InvalidateCacheForUser(username string) {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	delete(kl.cache, username)
}

// GetCachedUsers returns a list of usernames with cached keys
func (kl *KeyringLoader) GetCachedUsers() []string {
	kl.mu.RLock()
	defer kl.mu.RUnlock()
	
	users := make([]string, 0, len(kl.cache))
	for username := range kl.cache {
		users = append(users, username)
	}
	return users
}

// GetCacheStats returns statistics about the cache
func (kl *KeyringLoader) GetCacheStats() CacheStats {
	kl.mu.RLock()
	defer kl.mu.RUnlock()
	
	now := time.Now()
	var validCount, expiredCount int
	
	for _, cached := range kl.cache {
		if now.Before(cached.expiresAt) {
			validCount++
		} else {
			expiredCount++
		}
	}
	
	return CacheStats{
		TotalCached: len(kl.cache),
		ValidCount:  validCount,
		ExpiredCount: expiredCount,
		TTL:         kl.ttl,
	}
}

// CacheStats contains statistics about the keyring cache
type CacheStats struct {
	TotalCached  int
	ValidCount   int
	ExpiredCount int
	TTL          time.Duration
}

// CleanupExpiredKeys removes expired keys from the cache
func (kl *KeyringLoader) CleanupExpiredKeys() int {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	now := time.Now()
	removed := 0
	
	for username, cached := range kl.cache {
		if now.After(cached.expiresAt) {
			delete(kl.cache, username)
			removed++
		}
	}
	
	return removed
}

// SetTTL updates the TTL for future cache entries
// Existing cache entries keep their original expiration time
func (kl *KeyringLoader) SetTTL(ttl time.Duration) {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	
	kl.ttl = ttl
}

// GetTTL returns the current TTL setting
func (kl *KeyringLoader) GetTTL() time.Duration {
	kl.mu.RLock()
	defer kl.mu.RUnlock()
	
	return kl.ttl
}

// LoadDefaultKeyring is a convenience function that creates a KeyringLoader
// with default settings and loads the keyring for the current user
func LoadDefaultKeyring() (saltpack.Keyring, error) {
	loader, err := NewKeyringLoader(nil)
	if err != nil {
		return nil, err
	}
	
	return loader.LoadKeyring()
}

// LoadKeyringForUsername is a convenience function that creates a KeyringLoader
// and loads the keyring for a specific user
func LoadKeyringForUsername(username string) (saltpack.Keyring, error) {
	loader, err := NewKeyringLoader(nil)
	if err != nil {
		return nil, err
	}
	
	return loader.LoadKeyringForUser(username)
}
