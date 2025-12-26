package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func main() {
	fmt.Println("=== Custom Configuration Example ===\n")

	// Get temporary directory for this example
	tmpDir, err := os.MkdirTemp("", "keybase-cache-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Using temporary cache directory: %s\n\n", tmpDir)

	// Create custom configuration
	config := &cache.ManagerConfig{
		CacheConfig: &cache.CacheConfig{
			FilePath: filepath.Join(tmpDir, "custom_cache.json"),
			TTL:      6 * time.Hour, // 6 hour cache instead of default 24h
		},
		APIConfig: &api.ClientConfig{
			BaseURL:    api.DefaultAPIEndpoint,
			Timeout:    15 * time.Second,  // Longer timeout
			MaxRetries: 5,                 // More retries
			RetryDelay: 2 * time.Second,   // Longer initial delay
		},
	}

	fmt.Println("Custom Configuration:")
	fmt.Printf("  Cache Path: %s\n", config.CacheConfig.FilePath)
	fmt.Printf("  Cache TTL: %v\n", config.CacheConfig.TTL)
	fmt.Printf("  API Timeout: %v\n", config.APIConfig.Timeout)
	fmt.Printf("  Max Retries: %d\n", config.APIConfig.MaxRetries)
	fmt.Printf("  Retry Delay: %v\n\n", config.APIConfig.RetryDelay)

	// Create manager with custom config
	manager, err := cache.NewManager(config)
	if err != nil {
		log.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	fmt.Println("Cache manager created successfully with custom configuration")

	// Add some test data
	testUsers := []string{"alice", "bob", "charlie"}
	fmt.Println("\nPopulating cache with test data...")
	
	for _, username := range testUsers {
		err := manager.Cache().Set(
			username,
			fmt.Sprintf("mock_public_key_for_%s", username),
			fmt.Sprintf("mock_kid_for_%s", username),
		)
		if err != nil {
			log.Printf("Failed to cache %s: %v", username, err)
			continue
		}
		fmt.Printf("  Cached: %s\n", username)
	}

	// Verify cache file was created
	if _, err := os.Stat(config.CacheConfig.FilePath); err == nil {
		fmt.Printf("\nCache file created at: %s\n", config.CacheConfig.FilePath)
		
		// Show file contents
		data, err := os.ReadFile(config.CacheConfig.FilePath)
		if err == nil {
			fmt.Println("\nCache file contents:")
			fmt.Println(string(data))
		}
	}

	// Show cache statistics
	stats := manager.Stats()
	fmt.Printf("\nCache Statistics:\n")
	fmt.Printf("  Total Entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Valid Entries: %d\n", stats.ValidEntries)
	fmt.Printf("  Expired Entries: %d\n", stats.ExpiredEntries)

	fmt.Println("\n=== Example Complete ===")
}
