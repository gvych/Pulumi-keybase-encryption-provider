# Pulumi Configuration Examples

This directory contains example Pulumi stack configuration files demonstrating various ways to configure the Keybase encryption provider.

## Overview

Each `Pulumi.<stack-name>.yaml` file represents a different stack configuration with the Keybase secrets provider. These examples show common patterns and use cases.

## Example Files

### Basic Examples

#### `Pulumi.single-user.yaml`
**Use Case:** Individual developer working alone

- Single recipient
- Default settings
- Minimal configuration

```yaml
pulumi:secretsprovider: keybase://alice
```

#### `Pulumi.minimal.yaml`
**Use Case:** Bare minimum configuration

- Two recipients
- All defaults
- Simplest possible setup

```yaml
pulumi:secretsprovider: keybase://alice,bob
```

#### `Pulumi.team.yaml`
**Use Case:** Small team (2-10 members)

- Multiple recipients (5 team members)
- Default settings
- Shared team secrets

```yaml
pulumi:secretsprovider: keybase://alice,bob,charlie,dave,eve
```

### Environment-Specific Examples

#### `Pulumi.dev.yaml`
**Use Case:** Development environment

- Short cache TTL (1 hour)
- Fast iteration
- Debug settings enabled

```yaml
pulumi:secretsprovider: keybase://alice_dev,bob_dev?cache_ttl=3600
```

**Key Features:**
- Short cache TTL for testing key rotation
- Debug logging enabled
- Relaxed security settings

#### `Pulumi.staging.yaml`
**Use Case:** Pre-production testing

- Moderate cache TTL (12 hours)
- Dev + QA team members
- Balance between security and convenience

```yaml
pulumi:secretsprovider: keybase://alice_dev,bob_dev,charlie_qa,dave_lead?cache_ttl=43200
```

**Key Features:**
- Multiple team roles (dev, QA, lead)
- Moderate caching
- Production-like configuration

#### `Pulumi.production.yaml`
**Use Case:** Production deployment

- Long cache TTL (24 hours)
- Identity verification enabled
- Ops + Security + Platform teams
- Strict security settings

```yaml
pulumi:secretsprovider: keybase://alice_ops,bob_security,charlie_platform,dave_sre?format=saltpack&cache_ttl=86400&verify_proofs=true
```

**Key Features:**
- Identity proof verification
- Long cache TTL for stability
- Multiple specialized roles
- Warning-level logging

### Advanced Examples

#### `Pulumi.no-cache.yaml`
**Use Case:** Testing or frequent key rotation

- Cache disabled (`cache_ttl=0`)
- Always fetches fresh keys
- Useful for testing

```yaml
pulumi:secretsprovider: keybase://alice,bob?cache_ttl=0
```

**Warning:** Not recommended for production due to increased API calls.

#### `Pulumi.legacy-pgp.yaml`
**Use Case:** Legacy system compatibility

- PGP format instead of Saltpack
- Compatibility with older systems
- Same security level

```yaml
pulumi:secretsprovider: keybase://alice,bob,charlie?format=pgp&cache_ttl=86400
```

**Note:** Saltpack is recommended for new projects.

## Configuration Parameters

All examples use the Keybase URL scheme format:

```
keybase://user1,user2,user3?format=saltpack&cache_ttl=86400&verify_proofs=true
```

### URL Components

| Parameter | Description | Default | Example Values |
|-----------|-------------|---------|----------------|
| `user1,user2` | Recipient usernames | (required) | `alice,bob,charlie` |
| `format` | Encryption format | `saltpack` | `saltpack`, `pgp` |
| `cache_ttl` | Cache TTL in seconds | `86400` (24h) | `3600` (1h), `43200` (12h) |
| `verify_proofs` | Verify identity proofs | `false` | `true`, `false` |

## Using These Examples

### 1. Choose an Example

Select the example that best matches your use case:

```bash
# Copy the example to your project
cp examples/pulumi_configs/Pulumi.production.yaml /path/to/your/project/
```

### 2. Customize Recipients

Replace the example usernames with your actual Keybase usernames:

```yaml
# Before
pulumi:secretsprovider: keybase://alice,bob,charlie

# After (your team)
pulumi:secretsprovider: keybase://john_ops,jane_security,mike_platform
```

### 3. Verify Recipients Exist

Ensure all recipients have Keybase accounts:

```bash
keybase id john_ops
keybase id jane_security
keybase id mike_platform
```

### 4. Set Your Secrets

After configuring the provider, set your secrets:

```bash
# Set a secret
pulumi config set myapp:apiKey "sk_live_123456789" --secret

# View encrypted value
pulumi config get myapp:apiKey

# View decrypted value
pulumi config get myapp:apiKey --show-secrets
```

## Complete Project Structure

Here's a complete project structure using these examples:

```
my-infrastructure/
├── Pulumi.yaml              # Main project file
├── Pulumi.dev.yaml          # Development stack
├── Pulumi.staging.yaml      # Staging stack
├── Pulumi.production.yaml   # Production stack
├── index.ts                 # Infrastructure code
├── package.json             # Node dependencies
└── README.md                # Project documentation
```

### Example `index.ts`

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

const config = new pulumi.Config();

// Read secrets from config (automatically decrypted)
const dbPassword = config.requireSecret("databasePassword");
const apiKey = config.requireSecret("apiKey");

// Use secrets in resources
const db = new aws.rds.Instance("database", {
    password: dbPassword,
    engine: "postgres",
    instanceClass: "db.t3.micro",
    allocatedStorage: 20,
});

const api = new aws.apigatewayv2.Api("api", {
    protocolType: "HTTP",
});

// Export outputs
export const dbEndpoint = db.endpoint;
export const apiUrl = api.apiEndpoint;
```

## Migration from Other Providers

### From Passphrase Provider

1. **Backup current configuration:**
   ```bash
   cp Pulumi.production.yaml Pulumi.production.yaml.backup
   ```

2. **Update stack config:**
   ```bash
   # Change from passphrase to Keybase
   pulumi stack change-secrets-provider "keybase://alice,bob,charlie"
   ```

3. **Verify secrets:**
   ```bash
   pulumi config get myapp:apiKey --show-secrets
   ```

### From AWS KMS

1. **Export stack:**
   ```bash
   pulumi stack export > stack-backup.json
   ```

2. **Update secrets provider in YAML:**
   ```yaml
   # Before
   pulumi:secretsprovider: awskms://arn:aws:kms:...
   
   # After
   pulumi:secretsprovider: keybase://alice,bob,charlie
   ```

3. **Re-import stack:**
   ```bash
   pulumi stack import --file stack-backup.json
   ```

## Stack Initialization

### Create New Stack with Keybase

```bash
# Create dev stack
pulumi stack init dev --secrets-provider="keybase://alice,bob?cache_ttl=3600"

# Create production stack
pulumi stack init production --secrets-provider="keybase://ops_team,security_lead?verify_proofs=true"
```

### Switch Between Stacks

```bash
# List stacks
pulumi stack ls

# Switch to production
pulumi stack select production

# View configuration
pulumi config
```

## Best Practices

### 1. Use Environment-Specific Recipients

Different environments should have different recipient lists:

- **Dev:** All developers
- **Staging:** Developers + QA + Team leads
- **Production:** Ops + Security + Platform teams only

### 2. Enable Verification for Production

Always enable identity verification for production:

```yaml
pulumi:secretsprovider: keybase://ops?verify_proofs=true
```

### 3. Document Your Recipients

Create a `RECIPIENTS.md` file documenting who has access:

```markdown
# Secret Access Control

## Production
- alice_ops - Operations Lead
- bob_security - Security Team Lead
- charlie_platform - Platform Engineer

## Staging
- alice_ops - Operations Lead
- dave_qa - QA Lead
- eve_dev - Development Lead
```

### 4. Use Descriptive Usernames

Use usernames that identify roles:

```yaml
# Good
pulumi:secretsprovider: keybase://alice_ops,bob_security,charlie_platform

# Less Clear
pulumi:secretsprovider: keybase://alice,bob,charlie
```

### 5. Set Appropriate Cache TTL

| Environment | Recommended TTL | Reasoning |
|-------------|-----------------|-----------|
| Development | 1 hour (3600s) | Fast iteration, frequent key changes |
| Staging | 12 hours (43200s) | Balance between performance and freshness |
| Production | 24 hours (86400s) | Stability and reduced API calls |

## Troubleshooting

### "User not found" Error

Verify the username exists on Keybase:

```bash
keybase id alice
```

### "No public key available" Error

The user needs to generate a PGP key:

```bash
keybase pgp gen
```

### "Invalid format" Error

Ensure format is either `saltpack` or `pgp`:

```yaml
# Correct
pulumi:secretsprovider: keybase://alice?format=saltpack

# Incorrect
pulumi:secretsprovider: keybase://alice?format=aes
```

### Permission Issues

Ensure proper file permissions:

```bash
chmod 600 Pulumi.*.yaml
chmod 700 ~/.config/pulumi
```

## Testing Configuration

### Validate URL Parsing

Test your configuration URL:

```go
// examples/url_parsing/main.go
config, err := keybase.ParseURL("keybase://alice,bob?format=saltpack&cache_ttl=3600")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Recipients: %v\n", config.Recipients)
```

### Verify Recipients

Check all recipients have valid keys:

```bash
for user in alice bob charlie; do
    echo "Checking $user..."
    keybase pgp pull $user
done
```

## Additional Resources

- [Pulumi Configuration Guide](../../PULUMI_CONFIGURATION.md) - Complete configuration documentation
- [Environment Variables](../../ENVIRONMENT_VARIABLES.md) - Environment variable reference
- [URL Scheme](../../keybase/URL_PARSING.md) - URL format specification
- [Main README](../../README.md) - Project overview

## Getting Help

If you encounter issues:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review the [Pulumi Configuration Guide](../../PULUMI_CONFIGURATION.md)
3. Open an issue on GitHub
4. Ask in Pulumi Community Slack

## License

These examples follow Pulumi's licensing terms.
