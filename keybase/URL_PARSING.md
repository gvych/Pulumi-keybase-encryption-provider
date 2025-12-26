# URL Scheme Parsing

This document describes the URL scheme parsing functionality for the Keybase encryption provider.

## Overview

The Keybase provider uses a custom URL scheme to configure encryption recipients and options. The URL parsing functionality extracts and validates these configuration parameters.

## URL Format

```
keybase://user1,user2,user3?format=saltpack&cache_ttl=86400&verify_proofs=true
```

### Components

| Component | Description | Required | Default |
|-----------|-------------|----------|---------|
| `keybase://` | Scheme identifier | Yes | - |
| `user1,user2,user3` | Comma-separated recipient usernames | Yes | - |
| `format` | Encryption format: `saltpack` or `pgp` | No | `saltpack` |
| `cache_ttl` | Public key cache TTL in seconds | No | `86400` (24 hours) |
| `verify_proofs` | Require identity proof verification | No | `false` |

## Username Validation

Usernames must:
- Contain only alphanumeric characters (a-z, A-Z, 0-9) and underscores (_)
- Not be empty
- Not contain spaces, hyphens, or special characters

### Valid Usernames
- `alice`
- `bob_123`
- `charlie_test`
- `user123`

### Invalid Usernames
- `alice-bob` (contains hyphen)
- `alice@example.com` (contains @ and .)
- `alice bob` (contains space)
- `` (empty)

## Format Parameter

The `format` parameter specifies the encryption format to use.

### Supported Formats

| Format | Description |
|--------|-------------|
| `saltpack` | Modern Saltpack encryption format (default, recommended) |
| `pgp` | Legacy PGP encryption format |

The format parameter is case-insensitive (`SALTPACK`, `saltpack`, `SaltPack` are all valid).

## Cache TTL Parameter

The `cache_ttl` parameter specifies how long public keys should be cached, in seconds.

- Must be a non-negative integer
- Value of `0` disables caching (always fetch from API)
- Default is `86400` seconds (24 hours)

### Examples
- `cache_ttl=3600` - 1 hour
- `cache_ttl=43200` - 12 hours
- `cache_ttl=86400` - 24 hours (default)
- `cache_ttl=0` - No caching

## Verify Proofs Parameter

The `verify_proofs` parameter enables identity proof verification.

- Must be a boolean value: `true` or `false`
- Default is `false`

When enabled, the provider will verify Keybase identity proofs before accepting public keys.

## Usage

### Basic Parsing

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/pulumi/pulumi-keybase-encryption/keybase"
)

func main() {
	// Parse a URL
	config, err := keybase.ParseURL("keybase://alice,bob?format=saltpack")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Recipients: %v\n", config.Recipients)
	fmt.Printf("Format: %s\n", config.Format)
	fmt.Printf("Cache TTL: %s\n", config.CacheTTL)
	fmt.Printf("Verify Proofs: %t\n", config.VerifyProofs)
}
```

### Creating URLs from Config

```go
config := &keybase.Config{
	Recipients:   []string{"alice", "bob", "charlie"},
	Format:       keybase.FormatPGP,
	CacheTTL:     12 * time.Hour,
	VerifyProofs: true,
}

url := config.ToURL()
fmt.Println(url)
// Output: keybase://alice,bob,charlie?cache_ttl=43200&format=pgp&verify_proofs=true
```

### Round-Trip Conversion

```go
// Original config
original := &keybase.Config{
	Recipients:   []string{"alice", "bob"},
	Format:       keybase.FormatSaltpack,
	CacheTTL:     24 * time.Hour,
	VerifyProofs: false,
}

// Convert to URL
url := original.ToURL()

// Parse back to config
parsed, err := keybase.ParseURL(url)
if err != nil {
	log.Fatal(err)
}

// parsed should equal original
```

## Error Handling

The parser provides detailed error messages for various failure scenarios:

### Empty URL
```go
_, err := keybase.ParseURL("")
// Error: URL cannot be empty
```

### Invalid Scheme
```go
_, err := keybase.ParseURL("https://alice,bob")
// Error: invalid URL scheme: expected 'keybase', got 'https'
```

### No Recipients
```go
_, err := keybase.ParseURL("keybase://")
// Error: no recipients specified in URL
```

### Invalid Username
```go
_, err := keybase.ParseURL("keybase://alice@example.com")
// Error: invalid recipient username 'alice@example.com': username contains invalid character: @
```

### Invalid Format
```go
_, err := keybase.ParseURL("keybase://alice?format=aes")
// Error: invalid format parameter: unsupported format 'aes': must be 'saltpack' or 'pgp'
```

### Invalid Cache TTL
```go
_, err := keybase.ParseURL("keybase://alice?cache_ttl=invalid")
// Error: invalid cache_ttl parameter: strconv.ParseInt: parsing "invalid": invalid syntax

_, err := keybase.ParseURL("keybase://alice?cache_ttl=-100")
// Error: cache_ttl must be non-negative, got -100
```

### Invalid Verify Proofs
```go
_, err := keybase.ParseURL("keybase://alice?verify_proofs=maybe")
// Error: invalid verify_proofs parameter: strconv.ParseBool: parsing "maybe": invalid syntax
```

## Examples

### Single Recipient (Minimal)
```
keybase://alice
```
- Recipients: `[alice]`
- Format: `saltpack` (default)
- Cache TTL: `24h` (default)
- Verify Proofs: `false` (default)

### Multiple Recipients
```
keybase://alice,bob,charlie
```
- Recipients: `[alice, bob, charlie]`
- Format: `saltpack` (default)

### With Format
```
keybase://alice,bob?format=pgp
```
- Recipients: `[alice, bob]`
- Format: `pgp`

### With Cache TTL
```
keybase://alice?cache_ttl=3600
```
- Recipients: `[alice]`
- Cache TTL: `1h`

### With Identity Verification
```
keybase://alice?verify_proofs=true
```
- Recipients: `[alice]`
- Verify Proofs: `true`

### All Parameters
```
keybase://alice,bob,charlie?format=pgp&cache_ttl=7200&verify_proofs=true
```
- Recipients: `[alice, bob, charlie]`
- Format: `pgp`
- Cache TTL: `2h`
- Verify Proofs: `true`

### Complex Usernames
```
keybase://alice_test,bob_123,charlie_456
```
- Recipients: `[alice_test, bob_123, charlie_456]`
- Underscores are allowed in usernames

## API Reference

### Types

#### `Config`
```go
type Config struct {
	Recipients   []string
	Format       EncryptionFormat
	CacheTTL     time.Duration
	VerifyProofs bool
}
```

#### `EncryptionFormat`
```go
type EncryptionFormat string

const (
	FormatSaltpack EncryptionFormat = "saltpack"
	FormatPGP      EncryptionFormat = "pgp"
)
```

### Functions

#### `ParseURL(rawURL string) (*Config, error)`
Parses a Keybase URL scheme and returns a Config.

**Parameters:**
- `rawURL`: The URL string to parse

**Returns:**
- `*Config`: Parsed configuration
- `error`: Error if parsing fails

#### `ValidateFormat(format EncryptionFormat) error`
Validates that the encryption format is supported.

**Parameters:**
- `format`: The encryption format to validate

**Returns:**
- `error`: Error if format is not supported

#### `DefaultConfig() *Config`
Returns a Config with default values.

**Returns:**
- `*Config`: Default configuration

### Methods

#### `(c *Config) String() string`
Returns a string representation of the Config.

#### `(c *Config) ToURL() string`
Converts a Config back to a URL string.

## Testing

The URL parsing functionality includes comprehensive unit tests covering:

- Basic single recipient
- Multiple recipients
- All query parameters
- Username validation
- Format validation
- Cache TTL validation
- Verify proofs validation
- Error handling
- Round-trip conversion
- Edge cases

Run tests with:
```bash
go test -v ./keybase -run TestParseURL
go test -v ./keybase -run TestValidateFormat
go test -v ./keybase -run TestConfigToURL
go test -v ./keybase -run TestRoundTrip
```

Test coverage: **96.9%**

## Integration with Pulumi

The URL parsing functionality is designed to integrate with Pulumi's stack configuration:

```yaml
config:
  secretsprovider: keybase://alice,bob,charlie?format=saltpack
  encryptedkey: <encrypted-dek>
```

The Pulumi provider will:
1. Parse the URL from stack configuration
2. Extract recipient usernames
3. Configure the cache manager with the specified TTL
4. Use the specified encryption format
5. Optionally verify identity proofs

## Performance

- URL parsing is fast (< 1ms for typical URLs)
- No network calls during parsing
- Validation is done synchronously
- Safe for concurrent use

## Security Considerations

- URL parsing does not validate that users exist (validation happens during encryption)
- Username validation prevents injection attacks
- Format validation ensures only supported formats are used
- Cache TTL validation prevents invalid values
- No secrets are present in URLs (only public usernames and configuration)

## Future Enhancements

Potential future improvements:

1. Support for key fingerprint verification in URL
2. Support for additional encryption formats
3. Support for custom keyring paths
4. Support for offline mode configuration
5. Support for key rotation policies

## Related Documentation

- [Main README](../README.md) - Project overview
- [API Client](api/README.md) - Keybase API integration
- [Cache Manager](cache/README.md) - Public key caching
- [Cursor Rules](../.cursorrules) - Development guidelines
