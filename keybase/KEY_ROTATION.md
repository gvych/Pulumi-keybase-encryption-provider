# Key Rotation Support

The Keybase encryption provider supports key rotation with lazy re-encryption, allowing you to migrate encrypted secrets when keys are rotated without breaking existing deployments.

## Overview

Key rotation is a critical security practice where encryption keys are periodically replaced with new ones. This package provides:

- **Automatic detection** of when messages were encrypted with retired keys
- **Lazy re-encryption** that only happens when explicitly requested
- **Bulk migration tools** for updating multiple encrypted values
- **Zero-downtime migration** path for Pulumi state files

## Table of Contents

- [Why Key Rotation?](#why-key-rotation)
- [How It Works](#how-it-works)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Migration Workflows](#migration-workflows)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Why Key Rotation?

Key rotation is essential for:

1. **Limiting exposure**: Reduces the impact if a key is compromised
2. **Compliance**: Many security standards require periodic key rotation
3. **Team changes**: When team members leave, rotate keys to maintain security
4. **Defense in depth**: Regular rotation reduces the window of vulnerability
5. **Key hygiene**: Ensures keys don't become stale or forgotten

### When to Rotate Keys

Rotate keys when:

- **Scheduled rotation**: Every 90 days (recommended) or per your security policy
- **Team member departure**: Someone with key access leaves the organization
- **Suspected compromise**: Any indication a key may have been exposed
- **Security incident**: As part of incident response procedures
- **Access change**: Adding or removing team members from encrypted secrets

## How It Works

### Detection Phase

When you decrypt a message, the provider can detect if it was encrypted with an old key:

```
┌─────────────────────────────────────────────────────────────┐
│                    Decryption Process                       │
│                                                             │
│  1. Decrypt ciphertext with local keyring                  │
│  2. Extract MessageKeyInfo (which key was used)            │
│  3. Fetch current keys for all recipients (via API/cache)  │
│  4. Compare decryption key against current keys            │
│  5. Flag if keys don't match (rotation detected)           │
└─────────────────────────────────────────────────────────────┘
```

### Re-encryption Phase

When rotation is detected, you can trigger lazy re-encryption:

```
┌─────────────────────────────────────────────────────────────┐
│                  Re-encryption Process                      │
│                                                             │
│  1. Use plaintext from decryption                          │
│  2. Fetch current public keys for all recipients           │
│  3. Encrypt plaintext with current keys                    │
│  4. Return new ciphertext                                  │
│  5. Application updates stored ciphertext                  │
└─────────────────────────────────────────────────────────────┘
```

### Lazy Re-encryption

**Why "lazy"?**

Re-encryption only happens when explicitly requested, not automatically during every decryption. This gives you control over:

- **When** migration happens (during maintenance window, background job, etc.)
- **What** gets migrated (all secrets, or just critical ones)
- **How** migration is performed (bulk operation, gradual rollout, etc.)

## Quick Start

### Basic Key Rotation Detection

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/pulumi/pulumi-keybase-encryption/keybase"
)

func main() {
    // Create keeper
    keeper, err := keybase.NewKeeperFromURL("keybase://alice,bob,charlie")
    if err != nil {
        log.Fatal(err)
    }
    defer keeper.Close()
    
    ctx := context.Background()
    
    // Decrypt and check for key rotation
    oldCiphertext := []byte("...encrypted data...")
    plaintext, messageInfo, rotationInfo, err := keeper.DecryptAndDetectRotation(ctx, oldCiphertext)
    if err != nil {
        log.Fatal(err)
    }
    
    // Check if rotation was detected
    if rotationInfo != nil && rotationInfo.NeedsReEncryption {
        fmt.Printf("⚠️  Key rotation detected!\n")
        fmt.Printf("   Reason: %s\n", rotationInfo.RetirementReason)
        fmt.Printf("   Receiver key retired: %v\n", rotationInfo.ReceiverKeyRetired)
        fmt.Printf("   Sender key retired: %v\n", rotationInfo.SenderKeyRetired)
        
        // Trigger re-encryption (see next section)
    } else {
        fmt.Println("✓ Keys are current")
    }
    
    fmt.Printf("Decrypted: %s\n", string(plaintext))
}
```

### Automatic Lazy Re-encryption

```go
// Perform complete decrypt-detect-reencrypt workflow
plaintext, reencResult, err := keeper.PerformLazyReEncryption(ctx, oldCiphertext)
if err != nil {
    log.Fatal(err)
}

// Use the plaintext
fmt.Printf("Decrypted: %s\n", string(plaintext))

// Check if re-encryption was performed
if reencResult != nil {
    fmt.Println("✓ Key rotation detected and data re-encrypted")
    fmt.Printf("  New ciphertext ready (%d bytes)\n", len(reencResult.Ciphertext))
    fmt.Printf("  Re-encrypted at: %s\n", reencResult.ReEncryptedAt)
    
    // Update your storage with the new ciphertext
    // Example: Save to Pulumi state, database, file, etc.
    if err := updateStoredCiphertext(reencResult.Ciphertext); err != nil {
        log.Fatal(err)
    }
} else {
    fmt.Println("✓ No rotation detected, ciphertext unchanged")
}
```

### Bulk Migration

For migrating multiple encrypted values (e.g., entire Pulumi state file):

```go
// Map of secret IDs to encrypted data
secrets := map[string][]byte{
    "db_password":    []byte("...encrypted..."),
    "api_key":        []byte("...encrypted..."),
    "private_key":    []byte("...encrypted..."),
}

// Migrate all secrets
results := keeper.MigrateEncryptedData(ctx, secrets)

// Process results
for id, result := range results {
    if result.Error != nil {
        fmt.Printf("❌ %s: failed - %v\n", id, result.Error)
        continue
    }
    
    if result.RotationDetected {
        fmt.Printf("🔄 %s: re-encrypted\n", id)
        // Update stored ciphertext
        if err := updateSecret(id, result.NewCiphertext); err != nil {
            fmt.Printf("❌ %s: failed to update - %v\n", id, err)
        }
    } else {
        fmt.Printf("✓ %s: current (no action needed)\n", id)
    }
}
```

## API Reference

### KeyRotationDetector

#### `NewKeyRotationDetector(cacheManager *cache.Manager) *KeyRotationDetector`

Creates a new key rotation detector.

#### `DetectRotation(ctx context.Context, messageInfo *saltpack.MessageKeyInfo, configuredRecipients []string) (*KeyRotationInfo, error)`

Detects if a decrypted message used retired keys.

**Parameters:**
- `ctx`: Context for API calls
- `messageInfo`: The MessageKeyInfo from decryption
- `configuredRecipients`: Current list of recipient usernames

**Returns:**
- `KeyRotationInfo`: Details about detected rotation
- `error`: If detection fails

### KeyRotationInfo

Information about a detected key rotation:

```go
type KeyRotationInfo struct {
    // ReceiverKeyRetired indicates if the receiver key is retired
    ReceiverKeyRetired bool
    
    // SenderKeyRetired indicates if the sender key is retired
    SenderKeyRetired bool
    
    // ReceiverUsername is the username of the receiver
    ReceiverUsername string
    
    // SenderUsername is the username of the sender
    SenderUsername string
    
    // DecryptedAt is when the message was decrypted
    DecryptedAt time.Time
    
    // NeedsReEncryption indicates if re-encryption is needed
    NeedsReEncryption bool
    
    // RetirementReason describes why the key is retired
    RetirementReason string
}
```

### Keeper Methods

#### `DecryptAndDetectRotation(ctx context.Context, ciphertext []byte) ([]byte, *saltpack.MessageKeyInfo, *KeyRotationInfo, error)`

Decrypts ciphertext and checks for key rotation in one operation.

**Returns:**
- `plaintext`: The decrypted data
- `messageInfo`: Decryption metadata
- `rotationInfo`: Rotation info (nil if no rotation)
- `error`: If operation fails

#### `ReEncrypt(ctx context.Context, request *ReEncryptionRequest) (*ReEncryptionResult, error)`

Re-encrypts plaintext with current keys.

**Parameters:**
- `ctx`: Context for API calls
- `request`: Re-encryption request with plaintext and optional new recipients

**Returns:**
- `ReEncryptionResult`: The new ciphertext and metadata
- `error`: If re-encryption fails

#### `PerformLazyReEncryption(ctx context.Context, oldCiphertext []byte) ([]byte, *ReEncryptionResult, error)`

Performs complete decrypt-detect-reencrypt workflow.

**Returns:**
- `plaintext`: The decrypted data (always returned)
- `reencResult`: Re-encryption result (nil if no rotation detected)
- `error`: If operation fails

#### `MigrateEncryptedData(ctx context.Context, ciphertexts map[string][]byte) map[string]*MigrationResult`

Bulk migration of multiple encrypted values.

**Parameters:**
- `ctx`: Context for API calls
- `ciphertexts`: Map of identifiers to encrypted data

**Returns:**
- Map of identifiers to `MigrationResult`

### MigrationResult

Result of migrating a single encrypted value:

```go
type MigrationResult struct {
    // Plaintext is the decrypted data
    Plaintext []byte
    
    // NewCiphertext is the re-encrypted data (nil if no rotation)
    NewCiphertext []byte
    
    // RotationDetected indicates if rotation was detected
    RotationDetected bool
    
    // RotationInfo contains rotation details
    RotationInfo *KeyRotationInfo
    
    // Error is any error during migration
    Error error
}
```

## Migration Workflows

### Scenario 1: Scheduled Key Rotation

**Situation:** You rotate keys every 90 days as part of security policy.

**Workflow:**

1. **Rotate Keybase keys** (outside this provider):
   ```bash
   # Each team member rotates their Keybase encryption key
   keybase pgp select --import
   ```

2. **Update Pulumi configuration** with new recipients (if usernames changed):
   ```yaml
   config:
     pulumi:secretsprovider: keybase://alice,bob,charlie
   ```

3. **Run migration script**:
   ```bash
   # Use the bulk migration example above
   go run migration.go
   ```

4. **Verify migration**:
   ```bash
   pulumi preview  # Should show no changes
   ```

### Scenario 2: Team Member Departure

**Situation:** Bob leaves the team, need to remove his access.

**Workflow:**

1. **Update recipients** to remove Bob:
   ```yaml
   config:
     pulumi:secretsprovider: keybase://alice,charlie
   ```

2. **Decrypt and re-encrypt all secrets**:
   ```go
   // Old ciphertext encrypted for: alice,bob,charlie
   // New ciphertext encrypted for: alice,charlie
   plaintext, reencResult, err := keeper.PerformLazyReEncryption(ctx, oldCiphertext)
   ```

3. **Update Pulumi state**:
   ```bash
   pulumi up  # Updates state with new ciphertext
   ```

4. **Verify Bob can't decrypt**:
   ```bash
   # Bob's decryption should fail
   pulumi config get db_password  # Error: no matching key
   ```

### Scenario 3: Key Compromise

**Situation:** Alice's key is compromised, need emergency rotation.

**Workflow:**

1. **Alice revokes compromised key**:
   ```bash
   keybase pgp drop  # Revoke compromised key
   keybase pgp select --import  # Import new key
   ```

2. **Force cache invalidation**:
   ```go
   manager.InvalidateUser("alice")
   ```

3. **Bulk re-encryption** of all secrets:
   ```go
   results := keeper.MigrateEncryptedData(ctx, allSecrets)
   // Update all secrets in parallel
   ```

4. **Audit access**:
   ```bash
   # Review who could decrypt with old key
   # Update incident response logs
   ```

### Scenario 4: Gradual Migration

**Situation:** Large number of secrets, want to migrate gradually.

**Workflow:**

```go
// Phase 1: Identify secrets needing rotation
needsRotation := []string{}
for id, ciphertext := range allSecrets {
    _, _, rotationInfo, err := keeper.DecryptAndDetectRotation(ctx, ciphertext)
    if rotationInfo != nil && rotationInfo.NeedsReEncryption {
        needsRotation = append(needsRotation, id)
    }
}

// Phase 2: Migrate in batches (e.g., 100 at a time)
batchSize := 100
for i := 0; i < len(needsRotation); i += batchSize {
    end := i + batchSize
    if end > len(needsRotation) {
        end = len(needsRotation)
    }
    
    batch := needsRotation[i:end]
    migrateBatch(keeper, batch)
    
    // Add delay between batches
    time.Sleep(1 * time.Second)
}
```

## Best Practices

### 1. Regular Rotation Schedule

Establish a regular rotation schedule:

- **Every 90 days**: Industry standard for encryption keys
- **Every 30 days**: High-security environments
- **Every 6 months**: Low-risk systems

### 2. Monitor for Retired Keys

Set up monitoring to alert when old keys are detected:

```go
_, _, rotationInfo, _ := keeper.DecryptAndDetectRotation(ctx, ciphertext)
if rotationInfo != nil && rotationInfo.NeedsReEncryption {
    // Send alert to monitoring system
    metrics.Increment("keybase.rotation_detected")
    alerts.Send("Key rotation detected for secret: " + secretID)
}
```

### 3. Automated Migration

Create automated scripts for key rotation:

```bash
#!/bin/bash
# key-rotation.sh

# 1. Verify Keybase is available
keybase status

# 2. Run migration tool
go run ./cmd/migrate-keys/main.go

# 3. Commit updated state
git add Pulumi.*.yaml
git commit -m "Key rotation: updated encrypted secrets"

# 4. Notify team
slack-notify "Key rotation completed successfully"
```

### 4. Test Migration in Non-Production First

Always test migration workflow in dev/staging:

```bash
# Test in dev environment
pulumi stack select dev
go run migration.go

# Verify secrets still work
pulumi config get db_password

# Then migrate production
pulumi stack select prod
go run migration.go
```

### 5. Keep Rotation Audit Log

Maintain a log of all key rotations:

```go
type RotationAuditLog struct {
    Timestamp      time.Time
    User           string
    Reason         string
    SecretsUpdated int
    Success        bool
}

// Log each rotation event
auditLog := RotationAuditLog{
    Timestamp:      time.Now(),
    User:           "alice",
    Reason:         "scheduled 90-day rotation",
    SecretsUpdated: len(results),
    Success:        allSuccessful(results),
}
saveAuditLog(auditLog)
```

### 6. Backup Before Migration

Always backup before bulk migration:

```bash
# Backup Pulumi state
pulumi stack export > backup-$(date +%Y%m%d).json

# Backup environment secrets
cp .env .env.backup

# Run migration
go run migration.go

# If something goes wrong, restore:
pulumi stack import --file backup-20240101.json
```

## Troubleshooting

### "Rotation detection failed"

**Cause:** Can't fetch current keys from Keybase API.

**Solution:**
- Check network connectivity
- Verify Keybase API is accessible
- Check cache configuration
- Ensure usernames are valid

### "No matching key in keyring"

**Cause:** Your local keyring doesn't have the key needed for decryption.

**Solution:**
```bash
# Verify you're logged into Keybase
keybase status

# Check your encryption keys
keybase pgp list

# If missing, import your key
keybase pgp select --import
```

### "Re-encryption succeeded but state not updated"

**Cause:** New ciphertext generated but not saved.

**Solution:**
- Ensure you're updating storage with `reencResult.Ciphertext`
- Check file permissions on state file
- Verify Pulumi backend is writable

### False Positives in Rotation Detection

**Cause:** Key comparison failing due to format differences.

**Solution:**
- Update to latest version of provider
- Clear public key cache: `manager.InvalidateAll()`
- Re-fetch keys from API: `manager.RefreshUser("alice")`

### Performance Issues with Bulk Migration

**Cause:** Too many secrets being migrated at once.

**Solution:**
- Use batch migration (see Gradual Migration workflow)
- Add delays between batches
- Run during maintenance window
- Consider parallelization (carefully)

## Migration Path for Different Scenarios

### Pulumi State File Migration

When migrating Pulumi state files:

1. **Export current state**:
   ```bash
   pulumi stack export > state-backup.json
   ```

2. **Run migration**:
   ```go
   // Read state file
   // Extract encrypted secrets
   // Migrate each secret
   // Update state file
   ```

3. **Import updated state**:
   ```bash
   pulumi stack import --file state-migrated.json
   ```

4. **Verify**:
   ```bash
   pulumi config get db_password
   ```

### Environment Variables

For environment variables encrypted with Keybase:

```bash
# Decrypt all secrets
for var in $(env | grep ^ENC_ | cut -d= -f1); do
    decrypted=$(keybase decrypt -i ${!var})
    
    # Re-encrypt with new keys
    encrypted=$(echo "$decrypted" | keybase encrypt alice,bob,charlie)
    
    # Update environment
    export $var="$encrypted"
done
```

### Configuration Files

For config files with embedded encrypted values:

```go
// Read config file
config, _ := loadConfig("app.yaml")

// Migrate each encrypted field
for key, encryptedValue := range config.Secrets {
    plaintext, reencResult, _ := keeper.PerformLazyReEncryption(ctx, encryptedValue)
    if reencResult != nil {
        config.Secrets[key] = reencResult.Ciphertext
    }
}

// Save updated config
saveConfig("app.yaml", config)
```

## Security Considerations

### 1. Old Keys Should Be Revoked

After rotating keys and re-encrypting all secrets:

```bash
# Revoke old Keybase key
keybase pgp drop <key-id>

# This prevents anyone from using the old key
```

### 2. Secure the Migration Process

During migration:

- Run migration on secure machine
- Use encrypted connections (TLS)
- Don't log plaintext secrets
- Secure temporary storage
- Clean up decrypted data immediately

### 3. Verify Key Ownership

Before adding new recipients:

```bash
# Verify the user's Keybase identity
keybase id alice

# Check their proofs (GitHub, Twitter, etc.)
keybase prove twitter
```

### 4. Audit Trail

Maintain audit trail of all key rotations:

- Who rotated keys
- When rotation occurred
- What secrets were affected
- Success/failure status
- Any errors encountered

### 5. Defense in Depth

Key rotation is one layer of defense:

- Combine with access controls
- Monitor for unauthorized access
- Use least-privilege principles
- Regular security audits
- Incident response plan

## Examples

See the [examples/rotation](../../examples/rotation) directory for complete working examples:

- Basic rotation detection
- Lazy re-encryption
- Bulk migration
- Automated rotation scripts
- Error handling

## Related Documentation

- [Crypto Package README](./crypto/README.md)
- [Cache Manager README](./cache/README.md)
- [Pulumi Configuration Guide](../PULUMI_CONFIGURATION.md)
- [Security Best Practices](../SECURITY.md)

## Support

For issues with key rotation:

1. Check this documentation
2. Review the examples
3. Check GitHub issues
4. Open a new issue with details

## Contributing

When contributing to key rotation features:

1. Add tests for all scenarios
2. Update this documentation
3. Follow security best practices
4. Never log plaintext or keys
5. Maintain >90% test coverage
