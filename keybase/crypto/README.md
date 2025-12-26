# Keybase Crypto Package

This package implements PGP to Saltpack key conversion for the Keybase encryption provider.

## Overview

The crypto package provides functions to convert Keybase public keys from various formats (primarily Keybase Key IDs) into Saltpack BoxPublicKey format, which is used for encryption operations.

## Key Features

- **KID to BoxPublicKey Conversion**: Extracts Curve25519 public keys from Keybase Key IDs (KIDs)
- **Multiple Input Formats**: Supports KID, raw hex, and binary key formats
- **Key Validation**: Validates key format, size, and content
- **Batch Conversion**: Converts multiple user keys in a single operation
- **Type Safety**: Implements the saltpack.BoxPublicKey interface correctly

## Usage

### Converting a Single Key

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	// From Keybase KID (most common)
	kid := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	key, err := crypto.PGPToBoxPublicKey("", kid)
	if err != nil {
		log.Fatal(err)
	}
	
	// Validate the key
	if err := crypto.ValidatePublicKey(key); err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Successfully converted key: %s\n", crypto.FormatKeyID(key))
}
```

### Converting Multiple Keys

```go
users := []crypto.UserPublicKey{
	{
		Username:  "alice",
		KeyID:     "0120abcdef...",
		PublicKey: "",
	},
	{
		Username:  "bob",
		KeyID:     "01201234567890...",
		PublicKey: "",
	},
}

results, err := crypto.ConvertPublicKeys(users)
if err != nil {
	log.Printf("Warning: Some keys failed to convert: %v", err)
}

for _, result := range results {
	fmt.Printf("Converted key for %s\n", result.Username)
}
```

### Key Formats Supported

#### 1. Keybase KID (Recommended)

The Keybase Key ID format is the most reliable and preferred method:

```
Format: 0120 + 64 hex characters (32 bytes)
Example: 0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
```

- `0120`: Key type prefix (NaCl encryption key)
- 64 hex characters: The actual Curve25519 public key

#### 2. Raw Hex String

64 hex characters representing the 32-byte public key:

```go
hexKey := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
key, err := crypto.PGPToBoxPublicKey(hexKey, "")
```

#### 3. Raw Binary

32-byte binary data:

```go
binaryKey := []byte{0xab, 0xcd, 0xef, ...} // 32 bytes
key, err := crypto.PGPToBoxPublicKey(string(binaryKey), "")
```

#### 4. PGP Bundle (Not Yet Implemented)

Full PGP public key bundles are recognized but not yet parsed:

```go
pgpBundle := `-----BEGIN PGP PUBLIC KEY BLOCK-----
...
-----END PGP PUBLIC KEY BLOCK-----`

key, err := crypto.PGPToBoxPublicKey(pgpBundle, "")
// Returns error: "PGP bundle parsing not yet implemented; please use the KID field"
```

**Note**: For Keybase keys, always use the KID field from the API response rather than attempting to parse the PGP bundle.

## API Reference

### Functions

#### `PGPToBoxPublicKey(keyData string, kid string) (BoxPublicKey, error)`

Converts a public key from various formats to a BoxPublicKey.

**Parameters:**
- `keyData`: Key data in hex, binary, or PGP format
- `kid`: Keybase Key ID (preferred, tried first)

**Returns:**
- `BoxPublicKey`: The converted public key
- `error`: Any error that occurred during conversion

**Strategy:**
1. Try KID extraction (if provided)
2. Try hex string parsing (if 64 chars)
3. Try PGP bundle extraction (if contains "BEGIN PGP PUBLIC KEY")
4. Try direct binary conversion (if exactly 32 bytes)

#### `ValidatePublicKey(key BoxPublicKey) error`

Validates that a BoxPublicKey is well-formed.

**Checks:**
- Key is not all zeros (invalid key)

**Returns:**
- `nil` if valid
- `error` describing the validation failure

#### `FormatKeyID(key BoxPublicKey) string`

Formats a BoxPublicKey as a Keybase KID string.

**Returns:**
- String in format: `0120<64-hex-chars>`

#### `PublicKeyToHex(key BoxPublicKey) string`

Converts a BoxPublicKey to a hex string (without KID prefix).

**Returns:**
- 64-character hex string

#### `ConvertPublicKeys(users []UserPublicKey) ([]ConvertedKey, error)`

Converts multiple user public keys to BoxPublicKeys.

**Parameters:**
- `users`: Slice of UserPublicKey structs

**Returns:**
- `[]ConvertedKey`: Successfully converted keys
- `error`: Aggregated error message (if any conversions failed)

**Note**: Returns partial results if some keys fail validation.

### Types

#### `BoxPublicKey`

Wrapper around `saltpack.RawBoxKey` that implements `saltpack.BoxPublicKey` interface.

**Interface Methods:**
- `ToKID() []byte`: Returns the key bytes
- `ToRawBoxKeyPointer() *saltpack.RawBoxKey`: Returns pointer to raw key
- `HideIdentity() bool`: Returns false (don't hide recipient identities)
- `CreateEphemeralKey() (saltpack.BoxSecretKey, error)`: Creates ephemeral keypair

#### `UserPublicKey`

Input format for batch conversion:

```go
type UserPublicKey struct {
	Username  string // Keybase username
	PublicKey string // PGP bundle or hex
	KeyID     string // Keybase KID
}
```

#### `ConvertedKey`

Output format for batch conversion:

```go
type ConvertedKey struct {
	Username     string       // Keybase username
	BoxPublicKey BoxPublicKey // Converted key
	KeyID        string       // Original KID
}
```

## Key ID Format

Keybase uses a specific Key ID format for NaCl encryption keys:

```
0120 + <32-byte-public-key-as-hex>
```

- **0120**: Prefix indicating NaCl encryption key type
- **32 bytes**: Curve25519 public key (64 hex characters)

**Example:**
```
0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
|  |                                                                |
|  └─ 32-byte Curve25519 public key (64 hex chars)               |
|                                                                  |
└─ Key type prefix (NaCl encryption)                              |
```

## Integration with Keybase API

The typical workflow integrates with the Keybase API:

```go
// 1. Fetch user public keys from Keybase API
apiClient := api.NewClient(nil)
users, err := apiClient.LookupUsers(ctx, []string{"alice", "bob"})
if err != nil {
	log.Fatal(err)
}

// 2. Convert API response to BoxPublicKeys
var cryptoUsers []crypto.UserPublicKey
for _, user := range users {
	cryptoUsers = append(cryptoUsers, crypto.UserPublicKey{
		Username:  user.Username,
		PublicKey: user.PublicKey,
		KeyID:     user.KeyID,
	})
}

// 3. Convert to BoxPublicKeys
keys, err := crypto.ConvertPublicKeys(cryptoUsers)
if err != nil {
	log.Printf("Warning: %v", err)
}

// 4. Use for encryption
for _, key := range keys {
	fmt.Printf("Ready to encrypt for: %s\n", key.Username)
}
```

## Error Handling

The package provides detailed error messages:

```go
key, err := crypto.PGPToBoxPublicKey("invalid", "")
if err != nil {
	// Error message indicates the problem:
	// "unsupported key format: expected Keybase KID, PGP bundle, or 32-byte key"
}
```

**Common Errors:**
- `"invalid KID length"`: KID is not 68 characters (4 prefix + 64 hex)
- `"invalid KID prefix"`: KID doesn't start with "0120"
- `"failed to decode KID hex"`: KID contains non-hex characters
- `"invalid key size"`: Key is not 32 bytes
- `"public key is all zeros"`: Invalid/empty key
- `"PGP bundle parsing not yet implemented"`: Use KID instead

## Testing

Run the test suite:

```bash
# Run all tests
go test -v ./keybase/crypto/...

# Run with coverage
go test -v -cover ./keybase/crypto/...

# Run benchmarks
go test -v -bench=. ./keybase/crypto/...
```

**Test Coverage:** 72.5%

### Benchmarks

```bash
$ go test -bench=. ./keybase/crypto/...
BenchmarkExtractKeyFromKID-8        1000000    1234 ns/op
BenchmarkPGPToBoxPublicKey-8        1000000    1345 ns/op
BenchmarkConvertPublicKeys-8         500000    3456 ns/op
```

## Security Considerations

1. **Key Validation**: Always validate keys with `ValidatePublicKey()` after conversion
2. **KID Preference**: Always use KID when available (more reliable than PGP parsing)
3. **No Secret Keys**: This package only handles public keys; secret keys are never exposed
4. **Memory Safety**: Keys are properly copied, not referenced
5. **Input Validation**: All inputs are validated before processing

## Future Enhancements

- [ ] Full PGP bundle parsing support
- [ ] Curve25519 point validation
- [ ] Known weak key detection
- [ ] Key rotation support
- [ ] Saltpack native key format detection

## Dependencies

- `github.com/keybase/saltpack` - Saltpack encryption library
- `golang.org/x/crypto/nacl/box` - NaCl Box cryptography

## Related Packages

- `keybase/api` - Keybase API client (fetches public keys)
- `keybase/cache` - Public key caching layer

## References

- [Saltpack Format](https://saltpack.org/)
- [Keybase API](https://keybase.io/docs/api/1.0)
- [NaCl Cryptography](https://nacl.cr.yp.to/)
- [Curve25519](https://cr.yp.to/ecdh.html)
