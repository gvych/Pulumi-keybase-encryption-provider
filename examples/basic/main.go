package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func main() {
	fmt.Println("=== Keybase Public Key Cache Example ===\n")

	// Create cache manager with default configuration
	manager, err := cache.NewManager(nil)
	if err != nil {
		log.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Example 1: Fetch a single user's public key
	fmt.Println("Example 1: Fetching single user's public key")
	fmt.Println("Note: This requires a valid Keybase user. Uncomment to test with real API.")
	/*
	ctx := context.Background()
	key, err := manager.GetPublicKey(ctx, "alice")
	if err != nil {
		log.Printf("Error fetching key: %v", err)
	} else {
		fmt.Printf("  Username: %s\n", key.Username)
		fmt.Printf("  Key ID: %s\n", key.KeyID)
		fmt.Printf("  Public Key (first 50 chars): %s...\n", key.PublicKey[:50])
	}
	*/

	// Example 2: Demonstrate cache usage with mock data
	fmt.Println("\nExample 2: Cache operations")
	
	// Manually populate cache (simulating API response)
	fmt.Println("  Populating cache with mock data...")
	mockUsers := []struct {
		username  string
		publicKey string
		keyID     string
	}{
		{"alice", "-----BEGIN PGP PUBLIC KEY BLOCK----- alice_key_data...", "alice_kid_123"},
		{"bob", "-----BEGIN PGP PUBLIC KEY BLOCK----- bob_key_data...", "bob_kid_456"},
		{"charlie", "-----BEGIN PGP PUBLIC KEY BLOCK----- charlie_key_data...", "charlie_kid_789"},
	}

	for _, user := range mockUsers {
		if err := manager.Cache().Set(user.username, user.publicKey, user.keyID); err != nil {
			log.Printf("Failed to set cache for %s: %v", user.username, err)
		}
	}

	// Retrieve from cache
	fmt.Println("  Retrieving cached entries...")
	entry := manager.Cache().Get("alice")
	if entry != nil {
		fmt.Printf("    Found cached entry for alice:\n")
		fmt.Printf("      Key ID: %s\n", entry.KeyID)
		fmt.Printf("      Fetched at: %s\n", entry.FetchedAt.Format(time.RFC3339))
		fmt.Printf("      Expires at: %s\n", entry.ExpiresAt.Format(time.RFC3339))
	}

	// Example 3: Cache statistics
	fmt.Println("\nExample 3: Cache statistics")
	stats := manager.Stats()
	fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Valid entries: %d\n", stats.ValidEntries)
	fmt.Printf("  Expired entries: %d\n", stats.ExpiredEntries)

	// Example 4: Cache invalidation
	fmt.Println("\nExample 4: Cache invalidation")
	fmt.Println("  Invalidating alice's cache entry...")
	if err := manager.InvalidateUser("alice"); err != nil {
		log.Printf("Failed to invalidate: %v", err)
	}
	
	stats = manager.Stats()
	fmt.Printf("  Entries after invalidation: %d\n", stats.TotalEntries)

	// Example 5: Batch operations
	fmt.Println("\nExample 5: Batch operations")
	fmt.Println("  Note: This would fetch multiple users in a single API call.")
	fmt.Println("  Uncomment to test with real API:")
	/*
	ctx := context.Background()
	keys, err := manager.GetPublicKeys(ctx, []string{"alice", "bob", "charlie"})
	if err != nil {
		log.Printf("Error fetching keys: %v", err)
	} else {
		fmt.Printf("  Fetched %d keys\n", len(keys))
		for _, key := range keys {
			fmt.Printf("    - %s (Key ID: %s)\n", key.Username, key.KeyID)
		}
	}
	*/

	// Example 6: Prune expired entries
	fmt.Println("\nExample 6: Pruning expired entries")
	if err := manager.PruneExpired(); err != nil {
		log.Printf("Failed to prune: %v", err)
	} else {
		fmt.Println("  Expired entries pruned successfully")
		stats = manager.Stats()
		fmt.Printf("  Valid entries remaining: %d\n", stats.ValidEntries)
	}

	// Example 7: Clear cache
	fmt.Println("\nExample 7: Clearing cache")
	if err := manager.InvalidateAll(); err != nil {
		log.Printf("Failed to clear cache: %v", err)
	} else {
		fmt.Println("  Cache cleared successfully")
		stats = manager.Stats()
		fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
	}

	fmt.Println("\n=== Example Complete ===")
}
