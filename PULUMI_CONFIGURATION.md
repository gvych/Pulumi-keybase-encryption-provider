# Pulumi Configuration Guide

This guide explains how to configure Pulumi to use the Keybase encryption provider for secret management.

## Overview

The Keybase encryption provider integrates with Pulumi's secrets management system, allowing you to encrypt stack secrets using Keybase public keys. Multiple team members can be configured as recipients, enabling secure secret sharing across your infrastructure team.

## Configuration Methods

Pulumi supports three methods for configuring the Keybase encryption provider:

1. **Stack Configuration File** (`Pulumi.<stack-name>.yaml`)
2. **Environment Variables**
3. **Command Line Arguments**

## Stack Configuration File

The recommended method is to configure the secrets provider in your stack configuration file.

### Basic Configuration

Create or update your `Pulumi.<stack-name>.yaml` file:

```yaml
config:
  # Configure the Keybase secrets provider
  pulumi:secretsprovider: keybase://alice,bob,charlie
```

### With Encryption Format

Specify the encryption format (default is `saltpack`):

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie?format=saltpack
```

### With Custom Cache TTL

Configure public key caching duration (in seconds):

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie?cache_ttl=43200
```

This sets a 12-hour cache TTL (43200 seconds).

### With Identity Verification

Enable Keybase identity proof verification:

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie?verify_proofs=true
```

### Full Configuration

Combine all options:

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie?format=saltpack&cache_ttl=43200&verify_proofs=true
```

### With Application Configuration

Include your application-specific configuration alongside the secrets provider:

```yaml
config:
  # Secrets provider configuration
  pulumi:secretsprovider: keybase://alice,bob,charlie?format=saltpack&cache_ttl=86400
  
  # Application configuration
  myapp:region: us-west-2
  myapp:environment: production
  
  # Encrypted secrets (these will use the Keybase provider)
  myapp:databasePassword:
    secure: AAAAAQAAABBQdWx1bWkgU2VjcmV0...
  myapp:apiKey:
    secure: AAAAAQAAABBQdWx1bWkgU2VjcmV0...
```

## Environment Variables

Configure the Keybase encryption provider using environment variables.

### Required Variables

#### `PULUMI_CONFIG_PASSPHRASE_FILE` (Alternative)

If you're migrating from passphrase-based encryption:

```bash
export PULUMI_CONFIG_PASSPHRASE_FILE=/path/to/passphrase.txt
```

### Keybase-Specific Variables

#### `KEYBASE_RECIPIENTS`

Comma-separated list of recipient usernames:

```bash
export KEYBASE_RECIPIENTS="alice,bob,charlie"
```

#### `KEYBASE_FORMAT`

Encryption format (`saltpack` or `pgp`):

```bash
export KEYBASE_FORMAT="saltpack"
```

#### `KEYBASE_CACHE_TTL`

Cache TTL in seconds:

```bash
export KEYBASE_CACHE_TTL="86400"
```

#### `KEYBASE_VERIFY_PROOFS`

Enable identity proof verification:

```bash
export KEYBASE_VERIFY_PROOFS="true"
```

#### `KEYBASE_CACHE_PATH`

Custom cache file location:

```bash
export KEYBASE_CACHE_PATH="$HOME/.config/pulumi/keybase_cache.json"
```

### Complete Environment Configuration

```bash
#!/bin/bash
# Pulumi Keybase Provider Configuration

# Recipients (required)
export KEYBASE_RECIPIENTS="alice,bob,charlie"

# Encryption format (optional, default: saltpack)
export KEYBASE_FORMAT="saltpack"

# Cache TTL in seconds (optional, default: 86400)
export KEYBASE_CACHE_TTL="86400"

# Verify identity proofs (optional, default: false)
export KEYBASE_VERIFY_PROOFS="true"

# Custom cache path (optional)
export KEYBASE_CACHE_PATH="$HOME/.config/pulumi/keybase_cache.json"

# Run Pulumi command
pulumi up
```

Save this as `pulumi-keybase-env.sh` and source it before running Pulumi commands:

```bash
source pulumi-keybase-env.sh
pulumi up
```

## Command Line Configuration

Configure the secrets provider via command line when creating a new stack:

### Using URL Scheme

```bash
pulumi stack init production --secrets-provider="keybase://alice,bob,charlie"
```

### With Options

```bash
pulumi stack init production \
  --secrets-provider="keybase://alice,bob,charlie?format=saltpack&cache_ttl=86400&verify_proofs=true"
```

### Changing Provider for Existing Stack

```bash
pulumi stack change-secrets-provider "keybase://alice,bob,charlie"
```

## Configuration Examples by Use Case

### Single Developer

For individual development:

```yaml
config:
  pulumi:secretsprovider: keybase://alice
```

### Small Team

For a small team with 2-5 members:

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie
```

### Production Environment

For production with strict security requirements:

```yaml
config:
  pulumi:secretsprovider: keybase://ops_team,security_lead,platform_engineer?format=saltpack&verify_proofs=true&cache_ttl=43200
```

### Development Environment

For development with relaxed caching:

```yaml
config:
  pulumi:secretsprovider: keybase://dev_team,lead_dev?cache_ttl=3600
```

### Multi-Stack Setup

Create separate configurations for each environment:

**Pulumi.dev.yaml:**
```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob?cache_ttl=3600
  myapp:environment: development
  myapp:region: us-west-2
```

**Pulumi.staging.yaml:**
```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie?cache_ttl=43200
  myapp:environment: staging
  myapp:region: us-west-2
```

**Pulumi.production.yaml:**
```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie,ops_lead?format=saltpack&verify_proofs=true&cache_ttl=86400
  myapp:environment: production
  myapp:region: us-west-2
```

## Setting Secrets

Once configured, set secrets using the Pulumi CLI:

### Basic Secret

```bash
pulumi config set myapp:apiKey "sk_live_123456789" --secret
```

### From File

```bash
pulumi config set myapp:sshPrivateKey --secret < ~/.ssh/id_rsa
```

### From Standard Input

```bash
echo "my-secret-value" | pulumi config set myapp:password --secret
```

### Viewing Secrets

Encrypted secrets can only be viewed by configured recipients:

```bash
# View encrypted value
pulumi config get myapp:apiKey

# View decrypted value (requires Keybase authentication)
pulumi config get myapp:apiKey --show-secrets
```

## Prerequisites

### 1. Keybase Installation

Ensure Keybase is installed and configured:

```bash
# Verify Keybase is installed
keybase --version

# Check login status
keybase status

# Login if needed
keybase login
```

### 2. Verify Recipients Exist

Ensure all recipient usernames exist on Keybase:

```bash
keybase id alice
keybase id bob
keybase id charlie
```

### 3. Public Key Availability

Verify that all recipients have public keys available:

```bash
# Check a user's public key
keybase pgp pull alice
```

## Migration from Other Providers

### From Passphrase Provider

1. **Backup existing configuration:**
   ```bash
   cp Pulumi.<stack>.yaml Pulumi.<stack>.yaml.backup
   ```

2. **Export secrets:**
   ```bash
   pulumi config --show-secrets > secrets-backup.txt
   ```

3. **Change secrets provider:**
   ```bash
   pulumi stack change-secrets-provider "keybase://alice,bob,charlie"
   ```

4. **Re-encrypt secrets:**
   Pulumi will automatically re-encrypt secrets with the new provider.

### From AWS KMS

1. **Export current secrets:**
   ```bash
   pulumi stack export > stack-export.json
   ```

2. **Change provider:**
   ```bash
   pulumi stack change-secrets-provider "keybase://alice,bob,charlie"
   ```

3. **Import stack:**
   ```bash
   pulumi stack import --file stack-export.json
   ```

## Troubleshooting

### "Keybase not available" Error

Ensure Keybase is installed and in your PATH:

```bash
which keybase
keybase status
```

### "User not found" Error

Verify the username exists on Keybase:

```bash
keybase id <username>
```

### "No public key available" Error

The user needs to have a public key registered:

```bash
keybase pgp pull <username>
```

If the user doesn't have a key, they need to generate one:

```bash
keybase pgp gen
```

### Cache Issues

Clear the cache if experiencing issues:

```bash
rm ~/.config/pulumi/keybase_keyring_cache.json
```

### Permission Denied

Ensure cache directory has proper permissions:

```bash
chmod 700 ~/.config/pulumi
chmod 600 ~/.config/pulumi/keybase_keyring_cache.json
```

## Best Practices

### 1. Use Descriptive Usernames

Use clear, recognizable Keybase usernames that identify team members:

```yaml
config:
  pulumi:secretsprovider: keybase://alice_ops,bob_platform,charlie_security
```

### 2. Configure Per Environment

Use different recipients for different environments:

- **Development:** All developers
- **Staging:** Developers + QA team
- **Production:** Ops team + Security team

### 3. Enable Identity Verification for Production

```yaml
config:
  pulumi:secretsprovider: keybase://ops_team?verify_proofs=true
```

### 4. Set Appropriate Cache TTL

- **Development:** Short TTL (1 hour) for frequent key rotation
- **Production:** Longer TTL (24 hours) for stability

### 5. Document Recipients

Maintain a `RECIPIENTS.md` file documenting who has access:

```markdown
# Secrets Access

## Production Stack
- alice_ops - Operations Lead
- bob_platform - Platform Engineer
- charlie_security - Security Team Lead

## Staging Stack
- alice_ops - Operations Lead
- bob_platform - Platform Engineer
- dave_qa - QA Lead

## Development Stack
- All developers
```

### 6. Audit Secret Access

Regularly review who has access to secrets:

```bash
# Extract recipients from stack config
grep secretsprovider Pulumi.*.yaml
```

### 7. Rotate Recipients

When team members leave, rotate secrets and update recipients:

```bash
# Remove user from recipients
pulumi stack change-secrets-provider "keybase://alice,bob"

# Re-encrypt all secrets
pulumi up
```

## Security Considerations

### Encryption at Rest

- Stack files contain encrypted secrets
- Secrets are encrypted using Saltpack format
- Each recipient can decrypt using their private key

### Key Management

- Private keys remain on individual machines
- No shared credentials required
- Public keys cached for performance

### Network Security

- Public key lookups use HTTPS (Keybase API)
- Local decryption (no network required)
- Cache reduces API exposure

### Access Control

- Only configured recipients can decrypt secrets
- Identity proofs provide additional verification
- Keybase maintains cryptographic proof of identity

## Performance Considerations

### Cache Configuration

Optimize cache TTL based on usage patterns:

```yaml
# High-frequency updates (development)
pulumi:secretsprovider: keybase://devs?cache_ttl=3600

# Stable environments (production)
pulumi:secretsprovider: keybase://ops?cache_ttl=86400
```

### Batch Operations

The provider automatically batches public key lookups for multiple recipients:

```yaml
# Single API call for all recipients
pulumi:secretsprovider: keybase://alice,bob,charlie,dave,eve
```

### Offline Decryption

Once public keys are cached, decryption works offline:

1. First run (online): Fetches public keys
2. Subsequent runs (offline): Uses cached keys

## Complete Example Project

Here's a complete example project structure with Keybase configuration:

```
my-infrastructure/
├── Pulumi.yaml
├── Pulumi.dev.yaml
├── Pulumi.staging.yaml
├── Pulumi.production.yaml
├── index.ts
├── package.json
└── RECIPIENTS.md
```

**Pulumi.yaml:**
```yaml
name: my-infrastructure
runtime: nodejs
description: My infrastructure project with Keybase secrets
```

**Pulumi.dev.yaml:**
```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob?cache_ttl=3600
  aws:region: us-west-2
  myapp:environment: development
  myapp:dbPassword:
    secure: AAAAAQA...
  myapp:apiKey:
    secure: AAAAAQA...
```

**Pulumi.production.yaml:**
```yaml
config:
  pulumi:secretsprovider: keybase://alice_ops,bob_security,charlie_platform?format=saltpack&verify_proofs=true&cache_ttl=86400
  aws:region: us-east-1
  myapp:environment: production
  myapp:dbPassword:
    secure: AAAAAQA...
  myapp:apiKey:
    secure: AAAAAQA...
  myapp:encryptionKey:
    secure: AAAAAQA...
```

**index.ts:**
```typescript
import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

const config = new pulumi.Config();

// Access secrets (automatically decrypted)
const dbPassword = config.requireSecret("dbPassword");
const apiKey = config.requireSecret("apiKey");

// Use secrets in resources
const db = new aws.rds.Instance("mydb", {
    password: dbPassword,
    // ... other configuration
});

// Export non-sensitive outputs
export const dbEndpoint = db.endpoint;
```

## Additional Resources

- [URL Scheme Documentation](keybase/URL_PARSING.md)
- [API Reference](README.md#api-reference)
- [Cache Configuration](keybase/cache/README.md)
- [Keybase Documentation](https://keybase.io/docs)
- [Pulumi Secrets Management](https://www.pulumi.com/docs/intro/concepts/secrets/)

## Support

For issues and questions:

- **GitHub Issues:** Report bugs and feature requests
- **Documentation:** See README.md for detailed API documentation
- **Community:** Pulumi Community Slack

## Version Compatibility

- **Pulumi:** >= 3.0.0
- **Keybase:** >= 5.0.0
- **Go:** >= 1.18 (for provider development)

## License

This provider follows Pulumi's licensing terms.
