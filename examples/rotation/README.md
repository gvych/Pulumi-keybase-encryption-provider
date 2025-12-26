# Key Rotation Example

This example demonstrates how to use the Keybase encryption provider's key rotation features.

## What This Example Shows

1. **Initial encryption** with a set of recipient keys
2. **Key rotation simulation** (generating new keys)
3. **Rotation detection** (verifying old keys can't decrypt new data)
4. **Re-encryption** with new keys
5. **Bulk migration** of multiple secrets

## Running the Example

```bash
cd examples/rotation
go run main.go
```

## Expected Output

```
🔐 Keybase Key Rotation Example
================================

Step 1: Encrypting with initial keys...
  ✓ Encrypted secret for Alice and Bob
  ✓ Ciphertext length: XXX bytes
  ✓ Recipients: alice, bob
  ✓ Alice successfully decrypted: Database password: super-secret-password-123

Step 2: Simulating key rotation...
  ✓ Alice rotated her encryption key
  ✓ Bob kept his existing key
  ✓ Sender rotated their signing key

Step 3: Detecting rotation and re-encrypting...
  ✓ Re-encrypted with new keys
  ✓ New ciphertext length: XXX bytes
  ✓ Verified: Old Alice key CANNOT decrypt new ciphertext (expected)
  ✓ Verified: New Alice key successfully decrypted
  ✓ Plaintext matches: Database password: super-secret-password-123

Step 4: Bulk migration example...
  Migrating 5 secrets...
  ✓ Migrated: db_password (XXX bytes)
  ✓ Migrated: api_key (XXX bytes)
  ✓ Migrated: private_key (XXX bytes)
  ✓ Migrated: oauth_token (XXX bytes)
  ✓ Migrated: encryption_key (XXX bytes)
  ✓ Successfully migrated 5/5 secrets

✅ Key rotation example completed successfully!
```

## Key Concepts

### Why Key Rotation?

Key rotation is a critical security practice:

- **Limits exposure**: If a key is compromised, rotation limits the time window of vulnerability
- **Compliance**: Many security standards require periodic key rotation
- **Team changes**: When team members leave, rotate keys to maintain security
- **Defense in depth**: Regular rotation is a security best practice

### Detection

The example shows how old keys cannot decrypt data encrypted with new keys. This is the foundation of rotation detection.

### Lazy Re-encryption

Re-encryption only happens when:
- Rotation is detected
- You explicitly request it
- During a migration operation

This gives you control over when and how migration happens.

### Bulk Migration

For large deployments with many secrets, the provider supports bulk migration operations that can process multiple secrets efficiently.

## Real-World Usage

### With Pulumi

```go
keeper, _ := keybase.NewKeeperFromURL("keybase://alice,bob,charlie")

// Decrypt and check for rotation
plaintext, _, rotationInfo, err := keeper.DecryptAndDetectRotation(ctx, oldCiphertext)
if err != nil {
    log.Fatal(err)
}

if rotationInfo != nil && rotationInfo.NeedsReEncryption {
    fmt.Println("⚠️  Key rotation detected! Re-encrypting...")
    
    // Perform lazy re-encryption
    plaintext, result, err := keeper.PerformLazyReEncryption(ctx, oldCiphertext)
    if err != nil {
        log.Fatal(err)
    }
    
    if result != nil {
        // Update stored ciphertext
        updatePulumiState(result.Ciphertext)
    }
}
```

### Scheduled Rotation

```go
// Run this on a schedule (e.g., every 90 days)
func rotateAllSecrets(keeper *keybase.Keeper) {
    secrets := loadAllSecrets()
    
    results := keeper.MigrateEncryptedData(context.Background(), secrets)
    
    for id, result := range results {
        if result.RotationDetected {
            fmt.Printf("🔄 Rotated: %s\n", id)
            saveSecret(id, result.NewCiphertext)
        }
    }
}
```

## See Also

- [Key Rotation Documentation](../../keybase/KEY_ROTATION.md)
- [Keeper Implementation](../../keybase/keeper.go)
- [Rotation Tests](../../keybase/rotation_test.go)

## Security Notes

1. **Always backup** before rotating keys
2. **Test in dev/staging** before production
3. **Monitor rotation events** for anomalies
4. **Maintain audit logs** of all rotations
5. **Revoke old keys** after successful migration

## Troubleshooting

### "Failed to decrypt with new keys"

- Ensure new keys were generated correctly
- Verify keyring contains the correct keys
- Check that sender key matches

### "Rotation not detected"

- Verify keys actually changed
- Check cache invalidation
- Ensure API returns current keys

## Contributing

To improve this example:

1. Add error handling improvements
2. Show more complex scenarios
3. Demonstrate with real Keybase keys
4. Add performance metrics
