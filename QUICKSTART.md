# Quick Start Guide

Get up and running with the Keybase encryption provider for Pulumi in 5 minutes.

## Prerequisites

Before starting, ensure you have:

1. **Keybase installed and running**
   ```bash
   # macOS
   brew install keybase
   
   # Linux
   curl --remote-name https://prerelease.keybase.io/keybase_amd64.deb
   sudo dpkg -i keybase_amd64.deb
   
   # Verify installation
   keybase --version
   ```

2. **Keybase account logged in**
   ```bash
   # Login
   keybase login
   
   # Check status
   keybase status
   ```

3. **PGP key generated** (if you don't have one)
   ```bash
   keybase pgp gen
   ```

4. **Pulumi installed**
   ```bash
   # macOS
   brew install pulumi
   
   # Linux/WSL
   curl -fsSL https://get.pulumi.com | sh
   
   # Verify installation
   pulumi version
   ```

## Step 1: Install the Keybase Provider

Add the provider to your Go project:

```bash
go get github.com/pulumi/pulumi-keybase-encryption
```

Or for a new project:

```bash
mkdir my-infrastructure
cd my-infrastructure
go mod init my-infrastructure
go get github.com/pulumi/pulumi-keybase-encryption
```

## Step 2: Configure Pulumi Stack

### Option A: New Stack

Create a new stack with Keybase encryption:

```bash
# Initialize Pulumi project
pulumi new typescript  # or python, go, etc.

# Initialize stack with Keybase provider
pulumi stack init dev --secrets-provider="keybase://your-username"
```

### Option B: Existing Stack

Update an existing stack configuration file (`Pulumi.<stack-name>.yaml`):

```yaml
config:
  # Configure Keybase as secrets provider
  pulumi:secretsprovider: keybase://your-username
  
  # Your application configuration
  myapp:region: us-west-2
```

### Option C: Team Configuration

For team projects with multiple recipients:

```yaml
config:
  # Multiple team members can decrypt secrets
  pulumi:secretsprovider: keybase://alice,bob,charlie
  
  # Application configuration
  myapp:region: us-west-2
```

## Step 3: Set Secrets

Set encrypted secrets using the Pulumi CLI:

```bash
# Set a secret
pulumi config set myapp:apiKey "sk_live_123456789" --secret

# Set a secret from file
pulumi config set myapp:sshKey --secret < ~/.ssh/id_rsa

# Set a database password
pulumi config set myapp:dbPassword "secure-password-123" --secret
```

## Step 4: Use Secrets in Your Code

### TypeScript/JavaScript

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

const config = new pulumi.Config();

// Read secret (automatically decrypted)
const apiKey = config.requireSecret("apiKey");
const dbPassword = config.requireSecret("dbPassword");

// Use in resources
const db = new aws.rds.Instance("mydb", {
    password: dbPassword,
    // ... other config
});
```

### Python

```python
import pulumi
import pulumi_aws as aws

config = pulumi.Config()

# Read secret (automatically decrypted)
api_key = config.require_secret("apiKey")
db_password = config.require_secret("dbPassword")

# Use in resources
db = aws.rds.Instance("mydb",
    password=db_password,
    # ... other config
)
```

### Go

```go
package main

import (
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
    "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/rds"
)

func main() {
    pulumi.Run(func(ctx *pulumi.Context) error {
        conf := config.New(ctx, "myapp")
        
        // Read secret (automatically decrypted)
        apiKey := conf.RequireSecret("apiKey")
        dbPassword := conf.RequireSecret("dbPassword")
        
        // Use in resources
        _, err := rds.NewInstance(ctx, "mydb", &rds.InstanceArgs{
            Password: dbPassword,
            // ... other config
        })
        
        return err
    })
}
```

## Step 5: Deploy

Deploy your infrastructure:

```bash
# Preview changes
pulumi preview

# Deploy
pulumi up
```

## Complete Example

Here's a complete example project:

### Project Structure

```
my-infrastructure/
├── Pulumi.yaml
├── Pulumi.dev.yaml
├── index.ts
└── package.json
```

### `Pulumi.yaml`

```yaml
name: my-infrastructure
runtime: nodejs
description: My infrastructure with Keybase secrets
```

### `Pulumi.dev.yaml`

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob
  aws:region: us-west-2
  myapp:environment: development
  myapp:apiKey:
    secure: AAAAAQAAABBQdWx1bWkgU2VjcmV0...
```

### `index.ts`

```typescript
import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";

const config = new pulumi.Config();
const apiKey = config.requireSecret("apiKey");

// Create an S3 bucket
const bucket = new aws.s3.Bucket("my-bucket", {
    website: {
        indexDocument: "index.html",
    },
});

// Export the bucket name
export const bucketName = bucket.id;
```

### Deploy

```bash
# Install dependencies
npm install

# Deploy
pulumi up
```

## Common Configuration Patterns

### Single Developer

```yaml
config:
  pulumi:secretsprovider: keybase://alice
```

### Small Team

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie
```

### Development Environment

```yaml
config:
  pulumi:secretsprovider: keybase://dev_team?cache_ttl=3600
```

### Production Environment

```yaml
config:
  pulumi:secretsprovider: keybase://ops_team,security_lead?format=saltpack&verify_proofs=true&cache_ttl=86400
```

## Advanced Configuration

### Custom Cache TTL

Set cache duration in seconds:

```yaml
config:
  # 1 hour cache
  pulumi:secretsprovider: keybase://alice?cache_ttl=3600
  
  # 12 hour cache
  pulumi:secretsprovider: keybase://alice?cache_ttl=43200
  
  # No cache (always fetch fresh)
  pulumi:secretsprovider: keybase://alice?cache_ttl=0
```

### Identity Verification

Enable cryptographic identity verification:

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob?verify_proofs=true
```

### Multiple Parameters

Combine multiple options:

```yaml
config:
  pulumi:secretsprovider: keybase://alice,bob,charlie?format=saltpack&cache_ttl=43200&verify_proofs=true
```

## Working with Secrets

### View Secrets

```bash
# View encrypted value
pulumi config get myapp:apiKey

# View decrypted value
pulumi config get myapp:apiKey --show-secrets

# List all config (including secrets)
pulumi config --show-secrets
```

### Update Secrets

```bash
# Update existing secret
pulumi config set myapp:apiKey "new-value-123" --secret

# Remove a secret
pulumi config rm myapp:apiKey
```

### Export Configuration

```bash
# Export stack configuration
pulumi stack export > stack-backup.json

# Import configuration
pulumi stack import --file stack-backup.json
```

## Environment Variables

Alternative to stack configuration files:

```bash
# Set recipients
export KEYBASE_RECIPIENTS="alice,bob,charlie"

# Set cache TTL
export KEYBASE_CACHE_TTL="43200"

# Set format
export KEYBASE_FORMAT="saltpack"

# Enable verification
export KEYBASE_VERIFY_PROOFS="true"

# Run Pulumi
pulumi up
```

For more details, see [Environment Variables Documentation](ENVIRONMENT_VARIABLES.md).

## Migration from Other Providers

### From Passphrase Provider

```bash
# Export current secrets
pulumi config --show-secrets > secrets-backup.txt

# Change to Keybase provider
pulumi stack change-secrets-provider "keybase://alice,bob,charlie"

# Pulumi automatically re-encrypts secrets
# Verify with:
pulumi config --show-secrets
```

### From AWS KMS

```bash
# Export stack
pulumi stack export > stack-backup.json

# Update secrets provider in Pulumi.yaml
# Change from:
#   pulumi:secretsprovider: awskms://arn:aws:kms:...
# To:
#   pulumi:secretsprovider: keybase://alice,bob

# Import stack
pulumi stack import --file stack-backup.json
```

## Troubleshooting

### "Keybase not available"

Ensure Keybase is installed and running:

```bash
# Check installation
keybase --version

# Check status
keybase status

# Restart Keybase
keybase restart
```

### "User not found"

Verify the username exists:

```bash
keybase id alice
```

### "No public key"

Generate a PGP key:

```bash
keybase pgp gen
```

### Cache Issues

Clear the cache:

```bash
rm ~/.config/pulumi/keybase_keyring_cache.json
```

### Permission Errors

Fix file permissions:

```bash
chmod 700 ~/.config/pulumi
chmod 600 ~/.config/pulumi/keybase_keyring_cache.json
```

## Next Steps

### Learn More

- **[Pulumi Configuration Guide](PULUMI_CONFIGURATION.md)** - Comprehensive configuration documentation
- **[Environment Variables](ENVIRONMENT_VARIABLES.md)** - Environment variable reference
- **[URL Scheme](keybase/URL_PARSING.md)** - URL format specification
- **[Example Configs](examples/pulumi_configs/)** - Real-world configuration examples
- **[Code Examples](examples/)** - Working code examples

### Example Configurations

Explore complete configuration examples in [`examples/pulumi_configs/`](examples/pulumi_configs/):

- [`Pulumi.dev.yaml`](examples/pulumi_configs/Pulumi.dev.yaml) - Development setup
- [`Pulumi.staging.yaml`](examples/pulumi_configs/Pulumi.staging.yaml) - Staging setup
- [`Pulumi.production.yaml`](examples/pulumi_configs/Pulumi.production.yaml) - Production setup
- [`Pulumi.team.yaml`](examples/pulumi_configs/Pulumi.team.yaml) - Team setup

### API Reference

For programmatic usage, see:

- **[Cache Manager API](keybase/cache/README.md)** - Public key caching
- **[API Client](keybase/api/README.md)** - Keybase API integration
- **[Credentials](keybase/credentials/README.md)** - Credential discovery

## Getting Help

### Resources

- **GitHub Issues:** Report bugs and feature requests
- **Documentation:** See README.md for full API documentation
- **Community:** Pulumi Community Slack

### Common Issues

1. **Keybase not running:** `keybase status` and `keybase restart`
2. **Wrong username:** `keybase id <username>` to verify
3. **Missing PGP key:** `keybase pgp gen` to create one
4. **Cache problems:** Delete `~/.config/pulumi/keybase_keyring_cache.json`

## Security Best Practices

1. **Use identity verification for production**
   ```yaml
   pulumi:secretsprovider: keybase://team?verify_proofs=true
   ```

2. **Separate recipients by environment**
   - Dev: All developers
   - Staging: Dev + QA + Leads
   - Production: Ops + Security only

3. **Rotate team members**
   ```bash
   # When someone leaves, update recipients and re-encrypt
   pulumi stack change-secrets-provider "keybase://remaining,team,members"
   ```

4. **Document access control**
   - Maintain a `RECIPIENTS.md` file
   - Document who has access to each environment

5. **Use appropriate cache TTL**
   - Development: 1 hour (fast iteration)
   - Production: 24 hours (stability)

## Support

For additional help:

- **Documentation:** Check the comprehensive guides above
- **Examples:** Browse [`examples/`](examples/) directory
- **Issues:** Open a GitHub issue
- **Community:** Ask in Pulumi Community Slack

## License

This project follows Pulumi's licensing terms.

---

**Ready to get started?** Jump to [Step 1](#step-1-install-the-keybase-provider)!
