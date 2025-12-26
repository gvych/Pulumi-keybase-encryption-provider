# Examples

This directory contains example programs demonstrating how to use the Keybase encryption provider.

## Running the Examples

### API Client Example

Demonstrates the Keybase API client wrapper:

```bash
cd examples/api
go run main.go
```

This example shows:
- Creating API client with default and custom configuration
- Username validation
- Fetching public keys from Keybase API
- Batch user lookup (multiple users in single request)
- Error handling with APIError types
- Context usage for cancellation and timeouts
- Best practices for API usage

### Basic Usage Example

Demonstrates core cache operations with mock data:

```bash
cd examples/basic
go run main.go
```

This example shows:
- Creating a cache manager
- Populating cache with mock data
- Retrieving cached entries
- Cache statistics
- Cache invalidation
- Pruning expired entries
- Clearing the cache

### Custom Configuration Example

Demonstrates custom configuration options:

```bash
cd examples/custom
go run main.go
```

This example shows:
- Custom cache file path
- Custom TTL settings
- Custom API client configuration
- Viewing cache file contents
- Cache statistics

### Caching Example

Comprehensive demonstration of the caching layer:

```bash
cd examples/caching
go run main.go
```

This example shows:
- Direct cache usage with Set/Get operations
- Cache manager integration with API client
- TTL-based expiration
- Cache statistics and monitoring
- Entry pruning and invalidation
- Batch operations and performance optimization
- Cache persistence across application restarts

## Using with Real Keybase API

To test with the real Keybase API, uncomment the relevant sections in the examples:

```go
// In basic/main.go
ctx := context.Background()
key, err := manager.GetPublicKey(ctx, "alice")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Username: %s\n", key.Username)
fmt.Printf("Key ID: %s\n", key.KeyID)
```

**Note:** Replace "alice" with an actual Keybase username to test with real data.

## Example Output

### Basic Usage

```
=== Keybase Public Key Cache Example ===

Example 1: Fetching single user's public key
Note: This requires a valid Keybase user. Uncomment to test with real API.

Example 2: Cache operations
  Populating cache with mock data...
  Retrieving cached entries...
    Found cached entry for alice:
      Key ID: alice_kid_123
      Fetched at: 2025-12-26T17:54:04Z
      Expires at: 2025-12-27T17:54:04Z

Example 3: Cache statistics
  Total entries: 3
  Valid entries: 3
  Expired entries: 0

Example 4: Cache invalidation
  Invalidating alice's cache entry...
  Entries after invalidation: 2

Example 5: Batch operations
  Note: This would fetch multiple users in a single API call.
  Uncomment to test with real API:

Example 6: Pruning expired entries
  Expired entries pruned successfully
  Valid entries remaining: 2

Example 7: Clearing cache
  Cache cleared successfully
  Total entries: 0

=== Example Complete ===
```

### Custom Configuration

```
=== Custom Configuration Example ===

Using temporary cache directory: /tmp/keybase-cache-example-XXX

Custom Configuration:
  Cache Path: /tmp/keybase-cache-example-XXX/custom_cache.json
  Cache TTL: 6h0m0s
  API Timeout: 15s
  Max Retries: 5
  Retry Delay: 2s

Cache manager created successfully with custom configuration

Populating cache with test data...
  Cached: alice
  Cached: bob
  Cached: charlie

Cache file created at: /tmp/keybase-cache-example-XXX/custom_cache.json

Cache file contents:
{
  "entries": {
    "alice": {
      "username": "alice",
      "public_key": "mock_public_key_for_alice",
      "key_id": "mock_kid_for_alice",
      "fetched_at": "2025-12-26T17:53:58.360174646Z",
      "expires_at": "2025-12-26T23:53:58.360174646Z"
    },
    ...
  }
}

Cache Statistics:
  Total Entries: 3
  Valid Entries: 3
  Expired Entries: 0

=== Example Complete ===
```

## Additional Examples

### Fetching Multiple Users

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func main() {
	manager, err := cache.NewManager(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Fetch multiple users in a single API call
	ctx := context.Background()
	keys, err := manager.GetPublicKeys(ctx, []string{"alice", "bob", "charlie"})
	if err != nil {
		log.Fatal(err)
	}

	for _, key := range keys {
		fmt.Printf("User: %s, Key ID: %s\n", key.Username, key.KeyID)
	}
}
```

### Force Refresh

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func main() {
	manager, err := cache.NewManager(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Force refresh from API (bypasses cache)
	key, err := manager.RefreshUser(ctx, "alice")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Refreshed key for %s: %s\n", key.Username, key.KeyID)
}
```

### Cache Statistics Monitoring

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func main() {
	manager, err := cache.NewManager(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Close()

	// Monitor cache statistics
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats := manager.Stats()
		fmt.Printf("Cache: Total=%d, Valid=%d, Expired=%d\n",
			stats.TotalEntries,
			stats.ValidEntries,
			stats.ExpiredEntries,
		)

		// Prune expired entries
		if stats.ExpiredEntries > 0 {
			if err := manager.PruneExpired(); err != nil {
				log.Printf("Failed to prune: %v", err)
			}
		}
	}
}
```

## See Also

- [Main README](../README.md) - Project overview
- [Cache Package](../keybase/cache/README.md) - Cache implementation details
- [API Package](../keybase/api/README.md) - API client documentation
