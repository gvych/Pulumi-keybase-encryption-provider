# Keyring Loading Example

This example demonstrates the keyring loading functionality with TTL-based caching for the Keybase encryption provider.

## What This Example Shows

1. **Keyring Loading**: Automatically loads secret keys from `~/.config/keybase/`
2. **TTL-Based Caching**: Caches loaded keys in memory with configurable expiration
3. **Cache Management**: Shows how to manage cache (statistics, invalidation, cleanup)
4. **Encryption/Decryption**: Demonstrates using the loaded keyring for encryption and decryption
5. **Fallback Mode**: Gracefully handles missing Keybase installation with demo keys

## Prerequisites

### Option 1: With Real Keybase (Recommended)

1. Install Keybase:
   - **macOS**: `brew install keybase`
   - **Linux**: Download from https://keybase.io/download
   - **Windows**: Download from https://keybase.io/download

2. Configure Keybase:
   ```bash
   keybase login
   ```

3. Generate encryption keys (if you don't have them):
   ```bash
   keybase pgp gen
   ```

### Option 2: Demo Mode (No Installation Required)

If Keybase is not installed, the example automatically runs in demo mode with generated keys.

## Running the Example

```bash
# From the examples/keyring directory
go run main.go

# Or build and run
go build -o keyring main.go
./keyring
```

## Expected Output

### With Keybase Installed

```
=== Keybase Keyring Loading Example ===

Step 1: Checking Keybase availability...
  ✓ Keybase is available

Step 2: Getting current username...
  ✓ Current user: yourusername

Step 3: Creating keyring loader...
  ✓ Keyring loader created (TTL: 30 minutes)

Step 4: Loading keyring for current user...
  ✓ Keyring loaded successfully

Step 5: Cache statistics...
  Total cached keys: 1
  Valid keys: 1
  Expired keys: 0
  Cache TTL: 30m0s

Step 6: Loading sender key...
  ✓ Sender key loaded for: yourusername

Step 7: Testing encryption and decryption...
  ✓ Message encrypted
  ✓ Message decrypted
  Original:  Hello from Keybase keyring example!
  Decrypted: Hello from Keybase keyring example!
  Sender verified: <key-id>

Step 8: Demonstrating cache reuse...
  ✓ Keyring loaded from cache in 42.125µs
  ✓ Cache working correctly

Step 9: Cache management...
  Cached users: [yourusername]
  Removed 0 expired keys
  ✓ Invalidated cache for user: yourusername
  Cached keys after invalidation: 0

=== Example completed successfully! ===
```

### Without Keybase (Demo Mode)

```
=== Keybase Keyring Loading Example ===

Step 1: Checking Keybase availability...
  ⚠️  Keybase not available: Keybase CLI not found

This example requires Keybase to be installed and configured.
Install from: https://keybase.io/download

Falling back to demo mode with generated keys...

=== Running in Demo Mode ===

Step 1: Generating test keys...
  ✓ Test keys generated

Step 2: Creating keyring...
  ✓ Keyring created

Step 3: Setting up encryption...
  ✓ Encryptor and decryptor ready

Step 4: Testing encryption/decryption...
  ✓ Message encrypted
  ✓ Message decrypted
  Original:  Demo message with generated keys
  Decrypted: Demo message with generated keys

=== Demo completed successfully! ===
```

## Key Concepts Demonstrated

### KeyringLoader Configuration

```go
loader, err := crypto.NewKeyringLoader(&crypto.KeyringLoaderConfig{
    TTL: 30 * time.Minute,  // Cache keys for 30 minutes
})
```

**Configuration Options:**
- `TTL`: Time-to-live for cached keys (default: 1 hour)
- `ConfigDir`: Custom Keybase config directory (default: auto-detected)

### Loading Keyrings

```go
// Load for current user
keyring, err := loader.LoadKeyring()

// Load for specific user
keyring, err := loader.LoadKeyringForUser("alice")

// Get secret key directly
secretKey, err := loader.GetSecretKey("bob")
```

### Cache Management

```go
// Get statistics
stats := loader.GetCacheStats()

// Invalidate specific user
loader.InvalidateCacheForUser("alice")

// Invalidate all users
loader.InvalidateCache()

// Cleanup expired keys
removed := loader.CleanupExpiredKeys()
```

### Cache Performance

**First Load (Cold Cache)**:
- Reads from disk: ~10-50ms
- Parses and validates key: ~1-5ms
- Total: ~11-55ms

**Cached Load (Warm Cache)**:
- Memory lookup only: <1μs
- Speed improvement: ~10,000x faster

## Code Structure

```
main.go
├── main()                    # Entry point
│   ├── Check Keybase availability
│   ├── Create KeyringLoader
│   ├── Load keyring
│   ├── Test encryption/decryption
│   ├── Demonstrate cache reuse
│   └── Show cache management
└── runDemoMode()            # Fallback with generated keys
    ├── Generate test keys
    ├── Create simple keyring
    └── Test encryption/decryption
```

## Troubleshooting

### Error: "Keybase CLI not found"

**Cause**: Keybase is not installed or not in PATH

**Solution**: Install Keybase from https://keybase.io/download

### Error: "no Keybase user logged in"

**Cause**: No user is authenticated with Keybase

**Solution**: Run `keybase login` to authenticate

### Error: "sender key not found"

**Cause**: User doesn't have encryption keys set up

**Solution**: Run `keybase pgp gen` to generate keys

### Error: "invalid secret key"

**Cause**: Key file is corrupted or in wrong format

**Solution**: 
1. Backup existing keys
2. Run `keybase pgp gen` to generate new keys
3. Update references to use new keys

## Performance Tips

### 1. Use Longer TTL for Stable Environments

```go
// Production environment with infrequent key rotation
loader, _ := crypto.NewKeyringLoader(&crypto.KeyringLoaderConfig{
    TTL: 24 * time.Hour,  // Cache for 24 hours
})
```

### 2. Reuse the Keyring

```go
// Load once, use multiple times
keyring, _ := loader.LoadKeyring()

// Create multiple decryptors with same keyring
decryptor1, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{Keyring: keyring})
decryptor2, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{Keyring: keyring})
```

### 3. Pre-load Keys at Startup

```go
// Pre-load keys for known users at startup
for _, username := range knownUsers {
    loader.LoadKeyringForUser(username)
}
```

### 4. Monitor Cache Performance

```go
// Periodically log cache statistics
stats := loader.GetCacheStats()
log.Printf("Cache hit rate: %.2f%%", 
    float64(stats.ValidCount) / float64(stats.TotalCached) * 100)
```

## Related Documentation

- [Keyring Loading Documentation](../../keybase/crypto/KEYRING_LOADING.md)
- [Crypto Package Documentation](../../keybase/crypto/README.md)
- [Sender Key Implementation](../../keybase/crypto/sender.go)
- [Credentials Discovery](../../keybase/credentials/README.md)

## Additional Examples

See also:
- [Basic Example](../basic/main.go) - Simple encryption/decryption
- [Sender Key Example](../sender_key/main.go) - Using Keybase sender keys
- [Crypto Example](../crypto/main.go) - Comprehensive crypto features
- [Caching Example](../caching/main.go) - Public key caching

## License

See the [LICENSE](../../LICENSE) file in the root directory.
