# Keyring Loading Implementation

This document describes the keyring loading functionality for the Keybase encryption provider.

## Overview

The keyring loading system reads private keys from the local Keybase configuration directory (`~/.config/keybase/` on Linux/macOS) and provides them to the Saltpack decryption layer. It implements the `saltpack.Keyring` interface and includes TTL-based in-memory caching to avoid repeated disk access.

## Architecture

### Components

1. **KeyringLoader**: Main component that loads and caches keys
2. **SimpleKeyring**: Implementation of `saltpack.Keyring` interface
3. **SenderKey Loading**: Uses existing `LoadSenderKey` function from `sender.go`

### Directory Structure

```
~/.config/keybase/
├── config.json              # Contains current logged-in username
├── device_eks/              # Modern key storage location
│   ├── alice.eks           # Encrypted key storage for alice
│   └── bob.eks             # Encrypted key storage for bob
├── secretkeys/             # Legacy key storage location (fallback)
│   ├── alice
│   └── bob
└── <username>/             # Alternative location
    └── device_keys
```

## KeyringLoader API

### Creating a KeyringLoader

```go
import "github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"

// Use default settings (1-hour TTL, auto-detect config directory)
loader, err := crypto.NewKeyringLoader(nil)

// Or customize settings
loader, err := crypto.NewKeyringLoader(&crypto.KeyringLoaderConfig{
    TTL:       30 * time.Minute,  // Custom cache TTL
    ConfigDir: "/custom/path",     // Custom Keybase config directory
})
```

### Loading Keyrings

#### Load Keyring for Current User

```go
// Loads the keyring for the currently logged-in Keybase user
keyring, err := loader.LoadKeyring()
if err != nil {
    log.Fatalf("Failed to load keyring: %v", err)
}

// Use the keyring for decryption
decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
    Keyring: keyring,
})
```

#### Load Keyring for Specific User

```go
// Loads the keyring for a specific user
keyring, err := loader.LoadKeyringForUser("alice")
if err != nil {
    log.Fatalf("Failed to load keyring for alice: %v", err)
}
```

#### Get Secret Key Directly

```go
// Get just the secret key without creating a keyring
secretKey, err := loader.GetSecretKey("bob")
if err != nil {
    log.Fatalf("Failed to get secret key: %v", err)
}
```

### Convenience Functions

For simple use cases, convenience functions are available:

```go
// Load keyring for current user with default settings
keyring, err := crypto.LoadDefaultKeyring()

// Load keyring for specific user with default settings
keyring, err := crypto.LoadKeyringForUsername("alice")
```

## Caching Behavior

### Cache TTL

- **Default TTL**: 1 hour
- **Configurable**: Set custom TTL when creating KeyringLoader
- **Per-user caching**: Each user's key is cached separately
- **Automatic expiration**: Expired keys are automatically reloaded on next access

### Cache Operations

#### Invalidate Entire Cache

```go
// Clear all cached keys
loader.InvalidateCache()
```

#### Invalidate Specific User

```go
// Clear cache for specific user
loader.InvalidateCacheForUser("alice")
```

#### Manual Cleanup

```go
// Remove expired keys from cache
removed := loader.CleanupExpiredKeys()
fmt.Printf("Removed %d expired keys\n", removed)
```

#### Cache Statistics

```go
stats := loader.GetCacheStats()
fmt.Printf("Total cached: %d\n", stats.TotalCached)
fmt.Printf("Valid: %d\n", stats.ValidCount)
fmt.Printf("Expired: %d\n", stats.ExpiredCount)
fmt.Printf("TTL: %v\n", stats.TTL)
```

#### Dynamic TTL Updates

```go
// Update TTL for future cache entries
loader.SetTTL(2 * time.Hour)

// Get current TTL
ttl := loader.GetTTL()
```

#### List Cached Users

```go
users := loader.GetCachedUsers()
fmt.Printf("Cached users: %v\n", users)
```

## Key Storage Format

The KeyringLoader supports multiple key storage formats:

### JSON Format (Primary)

```json
{
  "encryption_key": "a1b2c3d4e5f6...",
  "username": "alice"
}
```

Alternative field names supported:
- `encryption_key`
- `box_key`
- `nacl_key`
- `key`
- `key_hex`

### Raw Hex Format (Fallback)

Keys can also be stored as plain hex strings:

```
a1b2c3d4e5f6789012345678901234567890123456789012345678901234
```

### Key Prefixes

The loader automatically strips common prefixes:
- `0x`
- `0X`

## Error Handling

### Common Errors

#### Keybase Not Installed

```go
keyring, err := loader.LoadKeyring()
// Error: "Keybase CLI not found: keybase command not found in PATH"
```

**Solution**: Install Keybase from https://keybase.io/download

#### No User Logged In

```go
keyring, err := loader.LoadKeyring()
// Error: "no Keybase user logged in"
```

**Solution**: Run `keybase login` to authenticate

#### Key Not Found

```go
keyring, err := loader.LoadKeyringForUser("nonexistent")
// Error: "sender key not found for user 'nonexistent'"
```

**Solution**: Ensure the user has encryption keys set up. Run `keybase pgp gen` if needed.

#### Invalid Key Format

```go
keyring, err := loader.LoadKeyring()
// Error: "invalid secret key for user 'alice': secret key must be 32 bytes"
```

**Solution**: Check that the key file is not corrupted and contains valid NaCl keys.

## Integration with Keeper

The Keeper automatically loads the local user's secret key for decryption:

```go
keeper, err := keybase.NewKeeper(&keybase.KeeperConfig{
    Config: config,
})
// Keeper automatically loads local secret key using LoadSenderKey
```

The `loadLocalSecretKey` function in `keeper.go`:

```go
func loadLocalSecretKey(keyring *crypto.SimpleKeyring) error {
    // Verify Keybase is available
    if err := credentials.VerifyKeybaseAvailable(); err != nil {
        return fmt.Errorf("keybase not available: %w", err)
    }
    
    // Load the sender key
    senderKey, err := crypto.LoadSenderKey(nil)
    if err != nil {
        return fmt.Errorf("failed to load sender key: %w", err)
    }
    
    // Add to keyring
    keyring.AddKey(senderKey.SecretKey)
    return nil
}
```

## Thread Safety

The KeyringLoader is thread-safe and can be used concurrently:

```go
loader, _ := crypto.NewKeyringLoader(nil)

// Safe to call from multiple goroutines
go func() { loader.LoadKeyringForUser("alice") }()
go func() { loader.LoadKeyringForUser("bob") }()
go func() { loader.GetCacheStats() }()
```

Internal synchronization uses `sync.RWMutex`:
- **Read operations** (GetCacheStats, GetTTL): Use read lock (concurrent)
- **Write operations** (LoadKeyring, InvalidateCache): Use write lock (exclusive)

## Performance Characteristics

### Cache Hit Performance

- **First load**: ~10-50ms (disk I/O + key parsing + validation)
- **Cached load**: <1μs (memory lookup only)
- **Memory usage**: ~1 KB per cached key

### Cache Miss Scenarios

1. **Key never loaded**: Full disk load + cache
2. **Cache expired**: Full disk load + cache update
3. **Cache invalidated**: Full disk load + cache

### Optimization Tips

1. **Use longer TTL** for stable environments
2. **Call LoadKeyring once** and reuse the keyring
3. **Pre-load keys** at startup if you know the users
4. **Monitor cache stats** to tune TTL

## Example: Complete Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
    // Create keyring loader with 30-minute cache
    loader, err := crypto.NewKeyringLoader(&crypto.KeyringLoaderConfig{
        TTL: 30 * time.Minute,
    })
    if err != nil {
        log.Fatalf("Failed to create keyring loader: %v", err)
    }

    // Load keyring for current user
    keyring, err := loader.LoadKeyring()
    if err != nil {
        log.Fatalf("Failed to load keyring: %v", err)
    }

    // Create encryptor and decryptor
    senderKey, _ := crypto.LoadSenderKey(nil)
    
    encryptor, _ := crypto.NewEncryptor(&crypto.EncryptorConfig{
        SenderKey: senderKey.SecretKey,
    })
    
    decryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
        Keyring: keyring,
    })

    // Encrypt and decrypt
    plaintext := []byte("Hello, World!")
    receivers := []saltpack.BoxPublicKey{senderKey.PublicKey}
    
    ciphertext, _ := encryptor.EncryptArmored(plaintext, receivers)
    decrypted, _, _ := decryptor.DecryptArmored(ciphertext)
    
    fmt.Printf("Original:  %s\n", plaintext)
    fmt.Printf("Decrypted: %s\n", decrypted)
    
    // Check cache stats
    stats := loader.GetCacheStats()
    fmt.Printf("Cache: %d valid, %d expired\n", stats.ValidCount, stats.ExpiredCount)
}
```

## Testing

Comprehensive tests are available in `keyring_test.go`:

```bash
# Run keyring loader tests
go test -v -run TestKeyringLoader

# Run all crypto tests including keyring
go test -v ./keybase/crypto
```

### Test Coverage

- ✅ Loading keyring for current user
- ✅ Loading keyring for specific user
- ✅ TTL-based cache expiration
- ✅ Cache invalidation (full and per-user)
- ✅ Cache statistics
- ✅ Concurrent access
- ✅ Multiple key formats (JSON, hex)
- ✅ Error handling (missing keys, invalid format)
- ✅ Dynamic TTL updates
- ✅ Expired key cleanup

## Security Considerations

### Key Storage Security

- Keys are stored in `~/.config/keybase/` with restrictive permissions (0600)
- Only the current user can read key files
- Keys are encrypted at rest by the Keybase client

### In-Memory Caching

- Cached keys are stored in process memory
- Keys are not written to disk by KeyringLoader
- Cache is cleared when process exits
- No key material appears in logs

### Key Rotation

When rotating keys:

```go
// Invalidate cache to force reload of new keys
loader.InvalidateCacheForUser("alice")

// Next load will fetch the new key
keyring, _ := loader.LoadKeyringForUser("alice")
```

## Platform Support

### Linux and macOS

- Config directory: `~/.config/keybase/`
- Full support for all features

### Windows

- Config directory: `%LOCALAPPDATA%\Keybase`
- Full support for all features

### Cross-platform Paths

The KeyringLoader automatically detects the correct config directory based on the operating system using the `credentials` package.

## Troubleshooting

### Debug Mode

To debug keyring loading issues:

```go
// Load keyring and capture detailed error
keyring, err := loader.LoadKeyring()
if err != nil {
    log.Printf("Detailed error: %+v\n", err)
}

// Check cache state
stats := loader.GetCacheStats()
log.Printf("Cache stats: %+v\n", stats)

// List cached users
users := loader.GetCachedUsers()
log.Printf("Cached users: %v\n", users)
```

### Common Issues

**Issue**: "config.json not found"
- **Cause**: Keybase not configured or no user logged in
- **Fix**: Run `keybase login`

**Issue**: "key file not found"
- **Cause**: User doesn't have encryption keys
- **Fix**: Run `keybase pgp gen` to generate keys

**Issue**: Cache not working
- **Cause**: TTL too short or cache being invalidated
- **Fix**: Increase TTL or check for unnecessary invalidation calls

## Future Enhancements

Potential improvements for future versions:

1. **Automatic key rotation detection**: Monitor key file changes
2. **Multiple key support**: Load all device keys, not just primary
3. **Key backup/restore**: Export/import cached keys
4. **Metrics collection**: Track cache hit rates and load times
5. **Smart cache eviction**: LRU or LFU instead of TTL-only

## Related Documentation

- [Sender Key Implementation](./sender.go) - Private key loading
- [Simple Keyring](./keys.go) - Keyring interface implementation
- [Credentials Discovery](../credentials/README.md) - Keybase installation detection
- [Crypto Package](./README.md) - Overall encryption/decryption documentation
