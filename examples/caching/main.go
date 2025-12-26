package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func main() {
	fmt.Println("=== Keybase Public Key Caching Demo ===")

	// Example 1: Using the cache directly
	fmt.Println("Example 1: Direct cache usage")
	demonstrateDirectCache()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// Example 2: Using the cache manager with API integration
	fmt.Println("Example 2: Cache Manager with API integration")
	demonstrateCacheManager()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// Example 3: Cache statistics and management
	fmt.Println("Example 3: Cache statistics and management")
	demonstrateCacheManagement()
}

func demonstrateDirectCache() {
	// Create a cache with custom configuration
	config := &cache.CacheConfig{
		FilePath: "/tmp/keybase_demo_cache.json",
		TTL:      5 * time.Minute, // Short TTL for demo
	}

	c, err := cache.NewCache(config)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	// Store a public key
	username := "alice"
	publicKey := "-----BEGIN PGP PUBLIC KEY BLOCK-----\n...\n-----END PGP PUBLIC KEY BLOCK-----"
	keyID := "0120abc123..."

	fmt.Printf("Storing public key for user: %s\n", username)
	if err := c.Set(username, publicKey, keyID); err != nil {
		log.Fatalf("Failed to set cache entry: %v", err)
	}

	// Retrieve the public key
	fmt.Printf("Retrieving public key for user: %s\n", username)
	entry := c.Get(username)
	if entry != nil {
		fmt.Printf("  Username: %s\n", entry.Username)
		fmt.Printf("  Key ID: %s\n", entry.KeyID)
		fmt.Printf("  Fetched at: %s\n", entry.FetchedAt.Format(time.RFC3339))
		fmt.Printf("  Expires at: %s\n", entry.ExpiresAt.Format(time.RFC3339))
		fmt.Printf("  Is expired: %v\n", entry.IsExpired())
	} else {
		fmt.Println("  Entry not found or expired")
	}
}

func demonstrateCacheManager() {
	// Create a cache manager with default configuration
	// This integrates the cache with the Keybase API client
	config := &cache.ManagerConfig{
		CacheConfig: &cache.CacheConfig{
			FilePath: "/tmp/keybase_manager_cache.json",
			TTL:      24 * time.Hour,
		},
	}

	manager, err := cache.NewManager(config)
	if err != nil {
		log.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Note: This would normally make an API call to fetch the public key
	// For this demo, we'll just show the API structure
	fmt.Println("Fetching public key (would query API if not cached):")
	fmt.Println("  manager.GetPublicKey(ctx, \"alice\")")
	fmt.Println("  -> Checks cache first")
	fmt.Println("  -> Falls back to API if not cached or expired")
	fmt.Println("  -> Automatically caches the result")

	// Get multiple keys efficiently (batch API call)
	fmt.Println("\nFetching multiple keys:")
	fmt.Println("  manager.GetPublicKeys(ctx, []string{\"alice\", \"bob\", \"charlie\"})")
	fmt.Println("  -> Uses cache for available keys")
	fmt.Println("  -> Fetches missing keys in single batch API call")
	fmt.Println("  -> Caches all fetched keys")

	// Force refresh a user's key
	fmt.Println("\nForcing refresh of a user's key:")
	fmt.Println("  manager.RefreshUser(ctx, \"alice\")")
	fmt.Println("  -> Invalidates cache entry")
	fmt.Println("  -> Fetches fresh key from API")
	fmt.Println("  -> Updates cache with new key")
}

func demonstrateCacheManagement() {
	config := &cache.CacheConfig{
		FilePath: "/tmp/keybase_mgmt_cache.json",
		TTL:      100 * time.Millisecond, // Very short TTL for demo
	}

	c, err := cache.NewCache(config)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	// Add several entries
	users := []string{"alice", "bob", "charlie", "dave"}
	for _, user := range users {
		if err := c.Set(user, "key_"+user, "id_"+user); err != nil {
			log.Printf("Failed to set %s: %v", user, err)
		}
	}

	// Check statistics
	stats := c.Stats()
	fmt.Printf("Initial cache statistics:\n")
	fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Valid entries: %d\n", stats.ValidEntries)
	fmt.Printf("  Expired entries: %d\n", stats.ExpiredEntries)

	// Wait for entries to expire
	fmt.Println("\nWaiting for entries to expire...")
	time.Sleep(150 * time.Millisecond)

	// Check statistics again
	stats = c.Stats()
	fmt.Printf("\nAfter expiration:\n")
	fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Valid entries: %d\n", stats.ValidEntries)
	fmt.Printf("  Expired entries: %d\n", stats.ExpiredEntries)

	// Prune expired entries
	fmt.Println("\nPruning expired entries...")
	if err := c.PruneExpired(); err != nil {
		log.Printf("Failed to prune: %v", err)
	}

	// Check statistics after pruning
	stats = c.Stats()
	fmt.Printf("\nAfter pruning:\n")
	fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Valid entries: %d\n", stats.ValidEntries)
	fmt.Printf("  Expired entries: %d\n", stats.ExpiredEntries)

	// Invalidate specific user
	fmt.Println("\nInvalidating specific user (alice)...")
	if err := c.Delete("alice"); err != nil {
		log.Printf("Failed to delete: %v", err)
	}

	// Clear all entries
	fmt.Println("Clearing all cache entries...")
	if err := c.Clear(); err != nil {
		log.Printf("Failed to clear: %v", err)
	}

	stats = c.Stats()
	fmt.Printf("\nAfter clearing:\n")
	fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
}
