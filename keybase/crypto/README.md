# Keybase Crypto Package

This package provides cryptographic functionality for the Keybase encryption provider, including ephemeral key generation for Saltpack encryption.

## Features

- **Ephemeral Key Generation**: Secure generation of temporary NaCl box key pairs
- **Entropy Error Handling**: Proper detection and handling of insufficient entropy conditions
- **Batch Key Generation**: Efficient generation of multiple key pairs
- **Secure Memory Zeroing**: Safe cleanup of secret keys from memory

## Usage

### Basic Key Generation

```go
package main

import (
	"fmt"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	// Create a new ephemeral key creator
	creator := crypto.NewEphemeralKeyCreator()
	
	// Generate a single key pair
	pair, err := creator.GenerateKey()
	if err != nil {
		panic(err)
	}
	
	// Use the keys...
	fmt.Printf("Public key: %x\n", pair.PublicKey.Bytes())
	
	// Clean up secret key when done
	defer pair.Zero()
}
```

### Batch Key Generation

```go
// Generate multiple key pairs at once
creator := crypto.NewEphemeralKeyCreator()

pairs, err := creator.GenerateKeys(10)
if err != nil {
	panic(err)
}

// Use the keys...
for i, pair := range pairs {
	fmt.Printf("Key pair %d: %x\n", i, pair.PublicKey.Bytes())
	defer pair.Zero()
}
```

### Custom Randomness Source (for testing)

```go
import (
	"crypto/rand"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

// Create a creator with custom randomness source
creator := crypto.NewEphemeralKeyCreatorWithReader(rand.Reader)

pair, err := creator.GenerateKey()
if err != nil {
	panic(err)
}
defer pair.Zero()
```

## Types

### EphemeralKeyCreator

The main type for generating ephemeral keys. Uses `crypto/rand` as the default randomness source.

**Methods:**
- `GenerateKey() (*EphemeralKeyPair, error)` - Generate a single key pair
- `GenerateKeys(count int) ([]*EphemeralKeyPair, error)` - Generate multiple key pairs

### EphemeralKeyPair

Represents a generated ephemeral key pair with public and secret components.

**Methods:**
- `Zero()` - Securely zero out both keys in memory

### BoxPublicKey / BoxSecretKey

32-byte NaCl box keys compatible with the NaCl/libsodium cryptography library.

**Methods:**
- `Bytes() []byte` - Get the key as a byte slice
- `Zero()` (SecretKey only) - Securely zero out the key in memory

## Error Handling

The package defines two main error types:

- **`ErrInsufficientEntropy`**: Returned when the system doesn't have enough entropy to generate secure keys
- **`ErrKeyGenerationFailed`**: Returned for general key generation failures

Both errors can be checked using `errors.Is()`:

```go
pair, err := creator.GenerateKey()
if err != nil {
	if errors.Is(err, crypto.ErrInsufficientEntropy) {
		// Handle entropy-specific error
	} else if errors.Is(err, crypto.ErrKeyGenerationFailed) {
		// Handle general generation error
	}
}
```

## Security Considerations

1. **Key Cleanup**: Always call `Zero()` on key pairs when done to clear secret keys from memory
2. **Randomness**: The package uses `crypto/rand.Reader` for secure randomness
3. **Entropy**: The implementation detects and reports insufficient entropy conditions
4. **Memory Safety**: Secret keys should be zeroed after use to prevent memory disclosure

## Performance

The package is optimized for performance:

- Single key generation: ~50-100 microseconds
- Batch generation: Efficient for generating multiple keys
- No unnecessary allocations or copies

See benchmarks in `ephemeral_test.go` for detailed performance metrics.

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./keybase/crypto/...

# Run with coverage
go test ./keybase/crypto/... -cover

# Run benchmarks
go test ./keybase/crypto/... -bench=.
```

Test coverage: >96%

## Implementation Notes

- Uses `golang.org/x/crypto/nacl/box` for NaCl box key generation
- Compatible with Saltpack encryption format
- Thread-safe (each EphemeralKeyCreator instance is independent)
- No external state or global variables

## References

- [NaCl: Networking and Cryptography library](https://nacl.cr.yp.to/)
- [Saltpack: a modern crypto messaging format](https://saltpack.org/)
- [Go crypto/rand package](https://pkg.go.dev/crypto/rand)
