# Sender Key Example

This example demonstrates how to use Keybase sender keys for authenticated encryption in the Pulumi Keybase encryption provider.

## What This Example Shows

1. **Credential Discovery**: Detecting Keybase installation and login status
2. **Sender Key Loading**: Loading the current user's private key from Keybase
3. **Key Validation**: Validating that the loaded key is properly formatted
4. **Authenticated Encryption**: Encrypting messages with sender authentication
5. **Sender Verification**: Verifying the sender's identity during decryption

## Prerequisites

### With Keybase Installed

- Keybase CLI installed (`keybase` command in PATH)
- Keybase user logged in (`keybase login`)
- Keybase encryption keys set up (`keybase pgp gen`)

### Without Keybase (Fallback Mode)

The example will automatically fall back to test keys if Keybase is not available, demonstrating the functionality without a real Keybase installation.

## Running the Example

```bash
# Navigate to the example directory
cd examples/sender_key

# Run the example
go run main.go
```

## Expected Output

### With Keybase Installed

```
=== Keybase Sender Key Example ===

Step 1: Verifying Keybase installation...

Step 2: Discovering Keybase credentials...
  ✓ Keybase CLI: /usr/bin/keybase
  ✓ Config directory: /home/user/.config/keybase
  ✓ Logged in as: alice

Step 3: Loading sender key from Keybase...
  ✓ Loaded key for user: alice
  ✓ Key ID: 0123456789abcdef...

Step 4: Validating sender key...
  ✓ Sender key is valid

Step 5: Generating recipient key...
  ✓ Recipient key generated

Step 6: Creating encryptor with sender key...
  ✓ Encryptor created

Step 7: Encrypting message...
  ✓ Message encrypted (245 bytes)
  Ciphertext preview: BEGIN KEYBASE SALTPACK ENCRYPTED MESSAGE. KiOXNFMXYMsN...

Step 8: Setting up recipient keyring...
  ✓ Keyring configured

Step 9: Creating decryptor...
  ✓ Decryptor created

Step 10: Decrypting and verifying message...
  ✓ Message decrypted
  ✓ Sender authenticated: true
  Decrypted message: "This is a secret message authenticated by my Keybase identity!"

=== Success! ===
The message was encrypted with your Keybase identity and
successfully decrypted and verified by the recipient.
```

### Without Keybase (Test Mode)

```
=== Keybase Sender Key Example ===

Step 1: Verifying Keybase installation...
Warning: Keybase not available (Keybase CLI not found)
Falling back to test keys for demonstration...

=== Using Test Keys for Demonstration ===

Creating test sender key...
  ✓ Test sender key created for user: alice

Validating test sender key...
  ✓ Test sender key is valid

Generating recipient key...
  ✓ Recipient key generated

Creating encryptor with test sender key...
  ✓ Encryptor created

Encrypting message...
  ✓ Message encrypted (245 bytes)

Setting up recipient keyring...
  ✓ Keyring configured

Creating decryptor...
  ✓ Decryptor created

Decrypting and verifying message...
  ✓ Message decrypted
  ✓ Sender authenticated: true
  Decrypted message: "This is a test message with sender authentication!"

Testing key save and load...
  ✓ Key saved to: /tmp/keybase-test-123456
  ✓ Key loaded for user: alice

=== Test Demonstration Complete ===
In a real scenario, sender keys would be loaded from
your Keybase installation at ~/.config/keybase/
```

## Key Concepts

### Sender Authentication

When you encrypt with a sender key:
- The message is signed with your private key
- Recipients can verify the message came from you
- This provides both confidentiality and authenticity

### Anonymous Encryption

If you don't want to authenticate messages, pass `nil` as the sender key:

```go
encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
    SenderKey: nil, // Anonymous sender
})
```

### Key Storage

Keybase stores your encryption keys in:
- **Linux/macOS**: `~/.config/keybase/device_eks/<username>.eks`
- **Windows**: `%LOCALAPPDATA%\Keybase\device_eks\<username>.eks`

### Security Considerations

1. **Protect Your Keys**: Sender keys are stored with 0600 permissions (owner read/write only)
2. **Validate Before Use**: Always validate sender keys with `ValidateSenderKey()`
3. **Use Real Keys in Production**: Test keys should only be used for development
4. **Verify Sender Identity**: Check `info.SenderIsAnon` to ensure sender is authenticated

## Related Examples

- [Basic Example](../basic/) - Basic encryption without sender authentication
- [Crypto Example](../crypto/) - Advanced cryptographic operations
- [API Example](../api/) - Fetching public keys from Keybase API

## See Also

- [Sender Key Documentation](../../keybase/crypto/README.md#sender-key-management)
- [Credential Discovery](../../keybase/credentials/README.md)
- [Saltpack Specification](https://saltpack.org/)
