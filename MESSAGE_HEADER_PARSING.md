# Saltpack Message Header Parsing Implementation

## Overview

This document describes the implementation of Saltpack message header parsing functionality for Phase 3 of the Keybase encryption provider (Linear issue PUL-22).

## Implemented Features

### 1. Message Header Deserialization

The implementation parses Saltpack message headers to extract recipient and sender information from encrypted messages. The core functionality is provided through the `crypto.ParseMessageKeyInfo()` function.

### 2. MessageKeyInfo Extraction

When decrypting a Saltpack message, the `saltpack.Open()` function returns a `MessageKeyInfo` structure containing:

- **ReceiverKey**: The secret key that was used to decrypt the message (BoxSecretKey)
- **SenderKey**: The public key of the sender (nil if anonymous)
- **SenderIsAnon**: Boolean flag indicating if the sender is anonymous
- **ReceiverIsAnon**: Boolean flag indicating if the receiver is anonymous

### 3. Recipient Key Identification

The implementation determines which recipient key was used for decryption by:

1. Extracting the `ReceiverKey` from `MessageKeyInfo`
2. Deriving its public key using `GetPublicKey()`
3. Computing the key identifier (KID) using `ToKID()`
4. Exposing this information through the `MessageInfo` structure

## API Reference

### Core Types

#### `MessageInfo`

```go
type MessageInfo struct {
    // ReceiverKID is the key identifier of the recipient key used for decryption
    ReceiverKID []byte
    
    // ReceiverKIDHex is the hex-encoded receiver key identifier
    ReceiverKIDHex string
    
    // SenderKID is the key identifier of the sender (nil if anonymous)
    SenderKID []byte
    
    // SenderKIDHex is the hex-encoded sender key identifier (empty if anonymous)
    SenderKIDHex string
    
    // IsAnonymousSender indicates if the sender is anonymous
    IsAnonymousSender bool
    
    // ReceiverIndex is the index of the recipient in the receivers list (0-based)
    // Currently set to -1 (unknown) as this information is not directly exposed
    // by the saltpack.MessageKeyInfo structure
    ReceiverIndex int
}
```

### Functions

#### `ParseMessageKeyInfo(info *saltpack.MessageKeyInfo) (*MessageInfo, error)`

Parses a `saltpack.MessageKeyInfo` structure and extracts all relevant information about the message sender and receiver.

**Returns:**
- Structured `MessageInfo` with parsed details
- Error if parsing fails

**Example:**
```go
plaintext, messageKeyInfo, err := decryptor.Decrypt(ciphertext)
if err != nil {
    return err
}

messageInfo, err := crypto.ParseMessageKeyInfo(messageKeyInfo)
if err != nil {
    return err
}

fmt.Printf("Receiver KID: %s\n", messageInfo.ReceiverKIDHex)
if messageInfo.IsAnonymousSender {
    fmt.Println("Sender: Anonymous")
} else {
    fmt.Printf("Sender KID: %s\n", messageInfo.SenderKIDHex)
}
```

#### `GetReceiverKeyID(info *saltpack.MessageKeyInfo) ([]byte, error)`

Convenience function to extract just the receiver key ID.

**Example:**
```go
receiverKID, err := crypto.GetReceiverKeyID(messageKeyInfo)
if err != nil {
    return err
}
fmt.Printf("Receiver: %x\n", receiverKID)
```

#### `GetSenderKeyID(info *saltpack.MessageKeyInfo) []byte`

Extracts the sender key ID. Returns `nil` if the sender is anonymous.

**Example:**
```go
senderKID := crypto.GetSenderKeyID(messageKeyInfo)
if senderKID == nil {
    fmt.Println("Anonymous sender")
} else {
    fmt.Printf("Sender: %x\n", senderKID)
}
```

#### `IsAnonymousSender(info *saltpack.MessageKeyInfo) bool`

Checks if the message was sent anonymously.

**Example:**
```go
if crypto.IsAnonymousSender(messageKeyInfo) {
    fmt.Println("Message is from anonymous sender")
}
```

#### `VerifySender(info *saltpack.MessageKeyInfo, expectedSenderKey saltpack.BoxPublicKey) bool`

Verifies that the message was sent by a specific sender.

**Example:**
```go
if crypto.VerifySender(messageKeyInfo, alicePublicKey) {
    fmt.Println("Message is from Alice")
}
```

#### `VerifyReceiver(info *saltpack.MessageKeyInfo, expectedReceiverKey saltpack.BoxPublicKey) bool`

Verifies that the message was decrypted by a specific receiver.

**Example:**
```go
if crypto.VerifyReceiver(messageKeyInfo, bobPublicKey) {
    fmt.Println("Message was decrypted by Bob")
}
```

#### `FormatKeyID(kid []byte) string`

Formats a key ID as a human-readable hex string. Useful for logging and debugging.

**Example:**
```go
formatted := crypto.FormatKeyID(receiverKID)
fmt.Printf("Receiver: %s\n", formatted)
```

## Keeper Integration

The `Keeper` type now provides a `DecryptWithInfo()` method that returns message header information along with the plaintext:

```go
func (k *Keeper) DecryptWithInfo(ctx context.Context, ciphertext []byte) ([]byte, *crypto.MessageInfo, error)
```

**Example usage:**
```go
keeper, err := keybase.NewKeeperFromURL("keybase://alice,bob")
if err != nil {
    return err
}
defer keeper.Close()

plaintext, messageInfo, err := keeper.DecryptWithInfo(ctx, ciphertext)
if err != nil {
    return err
}

fmt.Printf("Plaintext: %s\n", plaintext)
fmt.Printf("Decrypted by: %s\n", messageInfo.ReceiverKIDHex)

if messageInfo.IsAnonymousSender {
    fmt.Println("From: Anonymous")
} else {
    fmt.Printf("From: %s\n", messageInfo.SenderKIDHex)
}
```

## Multiple Recipients Support

The implementation correctly identifies which recipient was used for decryption when a message is encrypted for multiple recipients:

```go
// Encrypt message for Alice, Bob, and Charlie
ciphertext, err := encryptor.EncryptArmored(plaintext, []saltpack.BoxPublicKey{
    alicePublicKey,
    bobPublicKey,
    charliePublicKey,
})

// Bob decrypts the message
plaintext, messageInfo, err := bobDecryptor.DecryptWithInfo(ctx, ciphertext)

// Verify that Bob was the one who decrypted
if crypto.VerifyReceiver(messageInfo, bobPublicKey) {
    fmt.Println("Confirmed: Bob decrypted this message")
}

// This will return false for Alice and Charlie
crypto.VerifyReceiver(messageInfo, alicePublicKey)  // false
crypto.VerifyReceiver(messageInfo, charliePublicKey) // false
```

## Implementation Details

### Header Parsing Process

1. **Decryption**: When `saltpack.Open()` or `saltpack.Dearmor62DecryptOpen()` is called, the Saltpack library:
   - Parses the message header
   - Finds the matching secret key from the keyring
   - Decrypts the session key
   - Decrypts the payload
   - Returns `MessageKeyInfo` with details about the operation

2. **Information Extraction**: The `ParseMessageKeyInfo()` function:
   - Extracts the receiver's secret key that was used
   - Derives the corresponding public key
   - Computes the key identifier (KID)
   - Extracts sender information (if not anonymous)
   - Packages everything into a `MessageInfo` structure

3. **Verification**: Helper functions allow verification of:
   - Sender identity (for non-anonymous messages)
   - Receiver identity (which key was used)
   - Anonymous sender detection

### Saltpack Message Structure

Saltpack messages contain:
- **Format version**: Indicates the Saltpack version
- **Encryption type**: Always "encryption" for encrypted messages
- **Ephemeral public key**: Used for key agreement
- **Sender public key**: Present if sender is not anonymous
- **Recipients**: Array of encrypted session keys (one per recipient)
- **Payload**: Encrypted message data

The `MessageKeyInfo` provides information about which recipient slot was successfully used for decryption.

## Testing

Comprehensive test coverage includes:

- **Parsing tests**: Valid and invalid `MessageKeyInfo` parsing
- **Anonymous sender tests**: Handling of anonymous vs. identified senders
- **Multiple recipients tests**: Verification of correct recipient identification
- **Verification tests**: Sender and receiver verification functions
- **Integration tests**: Full encrypt/decrypt cycle with message info extraction

Run tests:
```bash
go test -v ./keybase/crypto/... -run Message
```

## Security Considerations

1. **Key ID Privacy**: The receiver key ID reveals which key was used for decryption. This is by design and necessary for the functionality.

2. **Sender Anonymity**: When messages are sent anonymously (`SenderKey: nil`), the sender's identity is not revealed in the message header.

3. **Recipient Enumeration**: Saltpack's design prevents enumeration attacks - attackers cannot determine the list of recipients without access to a valid secret key.

4. **Information Leakage**: The `MessageInfo` structure does not leak any cryptographic material - it only contains key identifiers (public information).

## Limitations

1. **ReceiverIndex**: The current implementation sets `ReceiverIndex` to `-1` (unknown) because `saltpack.MessageKeyInfo` doesn't directly expose which recipient slot was used. This information is available in the internal message header but not exposed through the public API.

2. **Named Recipients**: The `MessageKeyInfo` includes `NamedReceivers` and `NumAnonReceivers` fields, but these are not cryptographically verified and are only repeated from the incoming message. Our implementation doesn't currently expose these fields.

## Future Enhancements

Potential improvements for future versions:

1. **Recipient Index Extraction**: Parse the message header directly to determine the exact recipient slot index
2. **Recipient List Parsing**: Expose the list of named recipients (with appropriate warnings about non-verification)
3. **Caching**: Cache parsed `MessageInfo` to avoid re-parsing on repeated access
4. **Extended Metadata**: Extract and expose additional message metadata like timestamp, format version, etc.

## References

- [Saltpack Encryption Format](https://saltpack.org/encryption-format-v2)
- [Keybase Saltpack Go Library](https://github.com/keybase/saltpack)
- Linear Issue: PUL-22 - Message header parsing
- Related: `ENCRYPT_METHOD_DOCUMENTATION.md`, `SALTPACK_INTEGRATION_SUMMARY.md`
