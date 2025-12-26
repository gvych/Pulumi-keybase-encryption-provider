# Keybase Crypto Package

The `crypto` package provides Saltpack encryption and decryption functionality for the Pulumi Keybase encryption provider. It implements modern public-key encryption with support for multiple recipients.

## Armoring Strategy

**This package uses ASCII-armored Base62 encoding for Pulumi state files.**

### Why ASCII Armoring?

✅ **Git-friendly**: Line-based diffs instead of "binary files differ"  
✅ **Debuggable**: Clear BEGIN/END markers for visual inspection  
✅ **Cross-platform**: No encoding issues or line ending conflicts  
✅ **Industry standard**: Aligns with PGP, SSH, TLS conventions  

**Trade-offs**: 33% size overhead (~300 bytes per 1 KB secret), acceptable for small secrets in state files.

📖 **See [ARMORING_STRATEGY.md](../../ARMORING_STRATEGY.md) for complete decision rationale, benchmarks, and format comparison.**

## Overview

This package wraps the `github.com/keybase/saltpack` library to provide:

- **Multiple recipient encryption**: Encrypt messages for 1 to N recipients
- **Binary and ASCII-armored formats**: Choose between compact binary or text-safe armored output
- **Streaming encryption/decryption**: Efficient handling of large messages
- **Simple keyring management**: Easy key storage and lookup
- **Context support**: Cancellation and timeout support for long operations

## Quick Start

### Basic Encryption and Decryption

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
    // Generate key pairs
    sender, _ := crypto.GenerateKeyPair()
    recipient, _ := crypto.GenerateKeyPair()
    
    // Create encryptor
    encryptor, _ := crypto.NewEncryptor(&crypto.EncryptorConfig{
        SenderKey: sender.SecretKey,
    })
    
    // Encrypt message
    plaintext := []byte("Hello, World!")
    ciphertext, _ := encryptor.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
    
    // Create keyring and decryptor
    keyring := crypto.NewSimpleKeyring()
    keyring.AddKeyPair(recipient)
    keyring.AddPublicKey(sender.PublicKey) // For sender verification
    
    decryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
        Keyring: keyring,
    })
    
    // Decrypt message
    decrypted, _, _ := decryptor.Decrypt(ciphertext)
    fmt.Printf("Decrypted: %s\n", string(decrypted))
}
```

### Multiple Recipients

```go
// Encrypt for multiple recipients
recipients := []saltpack.BoxPublicKey{
    alice.PublicKey,
    bob.PublicKey,
    charlie.PublicKey,
}

ciphertext, err := encryptor.Encrypt(plaintext, recipients)
```

Each recipient can independently decrypt the message with their private key.

### ASCII-Armored Encryption (Recommended for Pulumi)

For storage in text files (like Pulumi state files):

```go
// Encrypt with ASCII armoring (Base62 encoding)
// Produces human-readable, git-friendly output
armoredCiphertext, err := encryptor.EncryptArmored(plaintext, recipients)

// Decrypt armored ciphertext
plaintext, info, err := decryptor.DecryptArmored(armoredCiphertext)
```

**Output format:**
```
BEGIN KEYBASE SALTPACK ENCRYPTED MESSAGE.
kiPgBwdlv5J3sZ7 qNhGGXwhVyE8XTp MPWDxEu0C4OKjmc
rCjQZBxShqhN7g7 o9Vc5xOQJgBPWvj XKZRyiuRn6vFZJC
END KEYBASE SALTPACK ENCRYPTED MESSAGE.
```

See [ARMORING_STRATEGY.md](../../ARMORING_STRATEGY.md) for rationale and format details.

### Streaming for Large Files

For messages larger than 10 MiB, use streaming:

```go
// Stream encrypt
plaintextReader := bytes.NewReader(largeMessage)
var ciphertextBuf bytes.Buffer

err := encryptor.EncryptStream(plaintextReader, &ciphertextBuf, recipients)

// Stream decrypt
ciphertextReader := bytes.NewReader(ciphertextBuf.Bytes())
var decryptedBuf bytes.Buffer

info, err := decryptor.DecryptStream(ciphertextReader, &decryptedBuf)
```

## API Reference

### Encryptor

#### `NewEncryptor(config *EncryptorConfig) (*Encryptor, error)`

Creates a new encryptor.

**Configuration:**
- `Version`: Saltpack version (defaults to Version2)
- `SenderKey`: Sender's secret key (optional, nil for anonymous sender)

#### `Encrypt(plaintext []byte, receivers []saltpack.BoxPublicKey) ([]byte, error)`

Encrypts plaintext for multiple recipients. Returns binary ciphertext.

#### `EncryptArmored(plaintext []byte, receivers []saltpack.BoxPublicKey) (string, error)`

Encrypts and returns ASCII-armored ciphertext (Base62 encoding).

#### `EncryptStream(plaintext io.Reader, ciphertext io.Writer, receivers []saltpack.BoxPublicKey) error`

Streams plaintext through encryption. Efficient for large messages.

#### `EncryptStreamArmored(plaintext io.Reader, ciphertext io.Writer, receivers []saltpack.BoxPublicKey) error`

Streams encryption with ASCII armoring.

#### `EncryptWithContext(ctx context.Context, plaintext []byte, receivers []saltpack.BoxPublicKey) ([]byte, error)`

Encrypts with context support for cancellation.

### Decryptor

#### `NewDecryptor(config *DecryptorConfig) (*Decryptor, error)`

Creates a new decryptor.

**Configuration:**
- `Keyring`: Required keyring containing secret keys for decryption

#### `Decrypt(ciphertext []byte) ([]byte, *saltpack.MessageKeyInfo, error)`

Decrypts binary ciphertext. Returns plaintext and message info.

#### `DecryptArmored(armoredCiphertext string) ([]byte, *saltpack.MessageKeyInfo, error)`

Decrypts ASCII-armored ciphertext.

#### `DecryptStream(ciphertext io.Reader, plaintext io.Writer) (*saltpack.MessageKeyInfo, error)`

Streams ciphertext through decryption.

#### `DecryptStreamArmored(armoredCiphertext io.Reader, plaintext io.Writer) (*saltpack.MessageKeyInfo, error)`

Streams armored ciphertext through decryption.

#### `DecryptWithContext(ctx context.Context, ciphertext []byte) ([]byte, *saltpack.MessageKeyInfo, error)`

Decrypts with context support.

### Key Management

#### `GenerateKeyPair() (*KeyPair, error)`

Generates a new random Curve25519 key pair.

#### `CreatePublicKey(keyBytes []byte) (saltpack.BoxPublicKey, error)`

Creates a public key from 32-byte array.

#### `CreatePublicKeyFromHex(hexKey string) (saltpack.BoxPublicKey, error)`

Creates a public key from hex-encoded string.

#### `CreateSecretKey(keyBytes []byte) (saltpack.BoxSecretKey, error)`

Creates a secret key from 32-byte array.

#### `CreateSecretKeyFromHex(hexKey string) (saltpack.BoxSecretKey, error)`

Creates a secret key from hex-encoded string.

#### `ValidatePublicKey(key saltpack.BoxPublicKey) error`

Validates a public key (checks for null, all-zero keys).

#### `ValidateSecretKey(key saltpack.BoxSecretKey) error`

Validates a secret key.

#### `KeysEqual(k1, k2 saltpack.BoxPublicKey) bool`

Checks if two public keys are equal.

### SimpleKeyring

A basic keyring implementation for storing and looking up keys.

#### `NewSimpleKeyring() *SimpleKeyring`

Creates a new empty keyring.

#### `AddKey(secretKey saltpack.BoxSecretKey)`

Adds a secret key to the keyring.

#### `AddPublicKey(publicKey saltpack.BoxPublicKey)`

Adds a public key (for sender verification).

#### `AddKeyPair(keyPair *KeyPair)`

Adds a key pair to the keyring.

#### `LookupBoxSecretKey(kids [][]byte) (int, saltpack.BoxSecretKey)`

Looks up a secret key by key ID. Returns index and key, or -1 if not found.

#### `LookupBoxPublicKey(kid []byte) saltpack.BoxPublicKey`

Looks up a public key by key ID.

#### `GetAllBoxSecretKeys() []saltpack.BoxSecretKey`

Returns all secret keys in the keyring.

#### `ImportBoxEphemeralKey(kid []byte) saltpack.BoxPublicKey`

Imports an ephemeral public key.

#### `CreateEphemeralKey() (saltpack.BoxSecretKey, error)`

Creates a new ephemeral key pair.

## Saltpack Format

Saltpack is a modern encryption format designed as a simpler, more secure alternative to PGP:

### Key Features

1. **Modern Cryptography**: Uses NaCl's Box construction (Curve25519-XSalsa20-Poly1305)
2. **Multiple Recipients**: Native support for encrypting to N recipients
3. **Authenticated Encryption**: Always authenticated, never outputs unauthenticated data
4. **Recipient Privacy**: Recipient list is encrypted (prevents enumeration attacks)
5. **Streaming Support**: Efficient for large files
6. **Two Encodings**: Binary (compact) and Base62-armored (text-safe)

### How It Works

1. **Encryption Process**:
   - Generate a random symmetric session key
   - Encrypt plaintext with ChaCha20-Poly1305 using session key
   - For each recipient:
     - Encrypt session key with recipient's public key using NaCl Box
     - Store encrypted session key in message header
   - Return ciphertext with header + encrypted payload

2. **Decryption Process**:
   - Read message header
   - Try to decrypt session key using recipient's secret key
   - If successful, decrypt payload with session key
   - Return plaintext

3. **Sender Authentication**:
   - Sender's secret key signs the message
   - Recipient can verify sender (if sender's public key is known)
   - Supports anonymous senders (sender key = nil)

## Security Considerations

### Best Practices

1. **Always verify sender**: Add sender's public key to keyring for verification
2. **Protect secret keys**: Store in secure locations with appropriate permissions
3. **Use unique keys**: Generate separate key pairs for different purposes
4. **Validate inputs**: Use `ValidatePublicKey()` and `ValidateSecretKey()`
5. **Clear sensitive data**: Zero out keys in memory when done

### Security Properties

- **Confidentiality**: Only intended recipients can decrypt
- **Authenticity**: Recipients can verify sender (if not anonymous)
- **Integrity**: Any tampering is detected
- **Forward Secrecy**: Not provided (use ephemeral keys if needed)
- **Deniability**: Messages can be forged by recipients (repudiable authentication)

### Known Limitations

1. **No forward secrecy**: Same keys encrypt all messages
2. **No key rotation**: Messages encrypted with old keys require re-encryption
3. **Recipient enumeration**: Header size reveals approximate recipient count
4. **PGP compatibility**: Not compatible with PGP/GPG

## Performance

### Benchmarks

Typical performance on modern hardware:

- **Encryption**: ~500 MB/s for large messages
- **Decryption**: ~600 MB/s for large messages
- **Key generation**: ~50,000 keys/second
- **Small messages**: <1ms overhead per message

### Optimization Tips

1. **Use streaming for large files**: Reduces memory usage
2. **Batch operations**: Encrypt multiple small messages as one large message
3. **Precompute shared keys**: Use `Precompute()` for repeated encryption to same recipient
4. **Cache public keys**: Avoid repeated API calls to fetch keys

## Examples

See the [examples/saltpack](../../examples/saltpack) directory for complete working examples:

- Basic encryption/decryption
- Multiple recipients
- ASCII armoring
- Streaming for large files
- Error handling
- Key management

## Testing

Run the test suite:

```bash
# Run all tests
go test -v ./keybase/crypto/...

# Run with coverage
go test -v -cover ./keybase/crypto/...

# Run benchmarks
go test -bench=. ./keybase/crypto/...
```

## Related Documentation

- [Saltpack Specification](https://saltpack.org/)
- [NaCl: Networking and Cryptography library](https://nacl.cr.yp.to/)
- [Keybase API Documentation](https://keybase.io/docs/api)
- [Go CDK Secrets](https://gocloud.dev/howto/secrets/)

## Contributing

When contributing to this package:

1. Maintain >90% test coverage
2. Add tests for all new features
3. Update documentation
4. Follow Go standard formatting
5. Never log plaintext or keys
6. Validate all inputs

## License

This package is part of the Pulumi Keybase encryption provider and follows Pulumi's licensing terms.
