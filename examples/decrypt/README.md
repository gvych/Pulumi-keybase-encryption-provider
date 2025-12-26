# Decrypt Method Example

This example demonstrates the Keybase Decrypt method implementation from Linear issue PUL-23.

## Features Demonstrated

1. **Single Recipient Decryption**: Encrypt and decrypt a message for one recipient
2. **Multiple Recipients**: Encrypt once, decrypt by any recipient
3. **Streaming Decryption**: Automatic streaming for large files (>10 MiB)
4. **Error Handling**: Proper error handling for invalid inputs

## Running the Example

```bash
cd examples/decrypt
go run main.go
```

## Expected Output

```
=== Keybase Decrypt Method Example ===

Example 1: Single Recipient
  Original: Hello, Alice! This is a secret message.
  Encrypted: 234 bytes
  Decrypted: Hello, Alice! This is a secret message.
  ✅ Success! Plaintext matches: true

Example 2: Multiple Recipients
  Original: Team message: Project Alpha is green!
  Recipients: alice, bob, charlie
  Encrypted: 298 bytes
  ✅ alice decrypted successfully
  ✅ bob decrypted successfully
  ✅ charlie decrypted successfully

Example 3: Streaming Large Files (>10 MiB)
  Message size: 11534336 bytes (11.00 MiB)
  Streaming threshold: 10 MiB
  Will use: Streaming decryption
  Encrypted: 16533798 bytes (15.77 MiB)
  Decrypted: 11534336 bytes (11.00 MiB)
  ✅ Success! Streaming decryption worked correctly: true

Example 4: Error Handling
  Test 1: Empty ciphertext
    ✅ Correctly rejected: ciphertext cannot be empty
  Test 2: Invalid ciphertext
    ✅ Correctly rejected: decryption failed: ...
  Test 3: Corrupted armored ciphertext
    ✅ Correctly rejected: armored decryption failed: ...
  Test 4: Decryption with wrong key
    ✅ Correctly rejected: no decryption key found
```

## Key Concepts

### Automatic Key Matching

When a message is encrypted for multiple recipients, Saltpack stores an encrypted copy of the session key for each recipient. During decryption:

1. The keyring is queried for available secret keys
2. Saltpack tries each encrypted session key
3. When a match is found, the session key is decrypted
4. The message payload is decrypted with the session key

**Result:** Any recipient can decrypt, but only needs their own secret key.

### Streaming vs In-Memory

The Keeper automatically chooses the best decryption method:

- **< 10 MiB:** In-memory decryption (faster)
- **≥ 10 MiB:** Streaming decryption (memory-efficient)

This is transparent to the caller - just use `Decrypt()` and it handles the rest.

### Error Handling

All errors are properly classified with Go Cloud error codes:

- `InvalidArgument`: Empty, invalid, or corrupted ciphertext
- `NotFound`: No matching decryption key in keyring
- `DeadlineExceeded`: Context timeout or cancellation

## Integration with Pulumi

This implementation satisfies the `driver.Keeper` interface required by Pulumi:

```go
type Keeper interface {
    Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
    // ... other methods
}
```

Pulumi can use this to decrypt secrets from state files encrypted with Keybase.

## Security Notes

1. **No Recipient Enumeration:** Attackers can't determine who can decrypt
2. **Authenticated Encryption:** All ciphertext is authenticated with Poly1305
3. **Modern Cryptography:** ChaCha20-Poly1305 and Curve25519
4. **Automatic Key Selection:** Prevents key confusion attacks

## Related Documentation

- [DECRYPT_METHOD_IMPLEMENTATION.md](../../DECRYPT_METHOD_IMPLEMENTATION.md) - Complete implementation details
- [SALTPACK_INTEGRATION_SUMMARY.md](../../SALTPACK_INTEGRATION_SUMMARY.md) - Saltpack integration
- [KEYRING_LOADING_IMPLEMENTATION.md](../../KEYRING_LOADING_IMPLEMENTATION.md) - Keyring details
