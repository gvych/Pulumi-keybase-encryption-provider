# Environment Variables

This document describes all environment variables supported by the Keybase encryption provider for Pulumi.

## Overview

Environment variables provide an alternative to stack configuration files for configuring the Keybase encryption provider. They are useful for:

- CI/CD pipelines
- Containerized environments
- Dynamic configuration
- Local development overrides

## Variable Reference

### Core Configuration

#### `KEYBASE_RECIPIENTS`

**Description:** Comma-separated list of Keybase usernames who can decrypt secrets.

**Type:** String (comma-separated list)

**Required:** No (but must be set via stack config or environment)

**Default:** None

**Example:**
```bash
export KEYBASE_RECIPIENTS="alice,bob,charlie"
```

**Usage:**
```bash
# Single recipient
export KEYBASE_RECIPIENTS="alice"

# Multiple recipients
export KEYBASE_RECIPIENTS="alice,bob,charlie"

# Team members with descriptive usernames
export KEYBASE_RECIPIENTS="alice_ops,bob_platform,charlie_security"
```

**Notes:**
- Usernames must be valid Keybase usernames (alphanumeric + underscore only)
- Spaces are ignored (trimmed automatically)
- Each username must exist on Keybase

---

#### `KEYBASE_FORMAT`

**Description:** Encryption format to use for secrets.

**Type:** String (enum)

**Required:** No

**Default:** `saltpack`

**Valid Values:**
- `saltpack` - Modern Saltpack encryption format (recommended)
- `pgp` - Legacy PGP encryption format

**Example:**
```bash
export KEYBASE_FORMAT="saltpack"
```

**Usage:**
```bash
# Use Saltpack (recommended)
export KEYBASE_FORMAT="saltpack"

# Use PGP (legacy)
export KEYBASE_FORMAT="pgp"
```

**Notes:**
- Case-insensitive (`SALTPACK`, `saltpack`, `SaltPack` all work)
- Saltpack is recommended for better performance and security

---

#### `KEYBASE_CACHE_TTL`

**Description:** Time-to-live for cached public keys, in seconds.

**Type:** Integer (seconds)

**Required:** No

**Default:** `86400` (24 hours)

**Valid Range:** 0 to 2147483647

**Example:**
```bash
export KEYBASE_CACHE_TTL="43200"  # 12 hours
```

**Usage:**
```bash
# No caching (always fetch from API)
export KEYBASE_CACHE_TTL="0"

# 1 hour
export KEYBASE_CACHE_TTL="3600"

# 12 hours
export KEYBASE_CACHE_TTL="43200"

# 24 hours (default)
export KEYBASE_CACHE_TTL="86400"

# 7 days
export KEYBASE_CACHE_TTL="604800"
```

**Notes:**
- Set to `0` to disable caching (not recommended for production)
- Longer TTL reduces API calls but may delay key rotation detection
- Recommended: 24 hours for production, 1 hour for development

---

#### `KEYBASE_VERIFY_PROOFS`

**Description:** Enable identity proof verification before accepting public keys.

**Type:** Boolean

**Required:** No

**Default:** `false`

**Valid Values:** `true`, `false`, `1`, `0`, `yes`, `no`

**Example:**
```bash
export KEYBASE_VERIFY_PROOFS="true"
```

**Usage:**
```bash
# Enable verification (recommended for production)
export KEYBASE_VERIFY_PROOFS="true"

# Disable verification (faster, but less secure)
export KEYBASE_VERIFY_PROOFS="false"
```

**Notes:**
- When enabled, verifies cryptographic proofs of user identity
- Recommended for production environments
- Adds slight overhead to initial key fetch

---

### Advanced Configuration

#### `KEYBASE_CACHE_PATH`

**Description:** Custom path for the public key cache file.

**Type:** String (file path)

**Required:** No

**Default:** `~/.config/pulumi/keybase_keyring_cache.json`

**Example:**
```bash
export KEYBASE_CACHE_PATH="/custom/path/cache.json"
```

**Usage:**
```bash
# Custom location
export KEYBASE_CACHE_PATH="/opt/pulumi/keybase_cache.json"

# Temporary directory
export KEYBASE_CACHE_PATH="/tmp/keybase_cache.json"

# Project-specific cache
export KEYBASE_CACHE_PATH="./keybase-cache/cache.json"
```

**Notes:**
- Directory must exist and be writable
- File permissions should be `0600` (owner read/write only)
- Useful for shared environments or containers

---

#### `KEYBASE_API_TIMEOUT`

**Description:** HTTP timeout for Keybase API requests, in seconds.

**Type:** Integer (seconds)

**Required:** No

**Default:** `30`

**Valid Range:** 1 to 300

**Example:**
```bash
export KEYBASE_API_TIMEOUT="60"
```

**Usage:**
```bash
# Short timeout (fast networks)
export KEYBASE_API_TIMEOUT="10"

# Default timeout
export KEYBASE_API_TIMEOUT="30"

# Long timeout (slow networks)
export KEYBASE_API_TIMEOUT="60"
```

**Notes:**
- Increase for slow or unreliable networks
- Decrease for fast networks to fail faster
- Affects initial key fetch only (not decryption)

---

#### `KEYBASE_API_MAX_RETRIES`

**Description:** Maximum number of retry attempts for failed API requests.

**Type:** Integer

**Required:** No

**Default:** `3`

**Valid Range:** 0 to 10

**Example:**
```bash
export KEYBASE_API_MAX_RETRIES="5"
```

**Usage:**
```bash
# No retries (fail immediately)
export KEYBASE_API_MAX_RETRIES="0"

# Default retries
export KEYBASE_API_MAX_RETRIES="3"

# More retries (unreliable networks)
export KEYBASE_API_MAX_RETRIES="5"
```

**Notes:**
- Set to `0` to disable retries
- Uses exponential backoff between retries
- Recommended: 3-5 for production, 0-1 for development

---

#### `KEYBASE_API_RETRY_DELAY`

**Description:** Initial delay between retry attempts, in seconds.

**Type:** Integer (seconds)

**Required:** No

**Default:** `1`

**Valid Range:** 0 to 60

**Example:**
```bash
export KEYBASE_API_RETRY_DELAY="2"
```

**Usage:**
```bash
# No delay (immediate retry)
export KEYBASE_API_RETRY_DELAY="0"

# Default delay
export KEYBASE_API_RETRY_DELAY="1"

# Longer delay (rate limiting)
export KEYBASE_API_RETRY_DELAY="5"
```

**Notes:**
- Uses exponential backoff (1s, 2s, 4s, 8s, ...)
- Increase if experiencing rate limiting
- Set to `0` for immediate retries (not recommended)

---

#### `KEYBASE_CONFIG_DIR`

**Description:** Custom path to Keybase configuration directory.

**Type:** String (directory path)

**Required:** No

**Default:** Platform-specific:
- Linux: `~/.config/keybase`
- macOS: `~/Library/Application Support/Keybase`
- Windows: `%LOCALAPPDATA%\Keybase`

**Example:**
```bash
export KEYBASE_CONFIG_DIR="/custom/keybase/config"
```

**Usage:**
```bash
# Custom config directory
export KEYBASE_CONFIG_DIR="/opt/keybase/config"

# Portable installation
export KEYBASE_CONFIG_DIR="./keybase-config"
```

**Notes:**
- Must contain valid Keybase configuration
- Used for decryption operations
- Rarely needed (auto-detected by default)

---

### Pulumi Integration

#### `PULUMI_CONFIG_PASSPHRASE`

**Description:** Pulumi configuration passphrase (used when migrating from passphrase-based encryption).

**Type:** String

**Required:** No (unless migrating)

**Default:** None

**Example:**
```bash
export PULUMI_CONFIG_PASSPHRASE="my-secret-passphrase"
```

**Notes:**
- Not used by Keybase provider directly
- Needed during migration from passphrase provider
- Can be replaced by Keybase provider after migration

---

#### `PULUMI_CONFIG_PASSPHRASE_FILE`

**Description:** Path to file containing Pulumi configuration passphrase.

**Type:** String (file path)

**Required:** No (unless migrating)

**Default:** None

**Example:**
```bash
export PULUMI_CONFIG_PASSPHRASE_FILE="$HOME/.pulumi/passphrase.txt"
```

**Notes:**
- Alternative to `PULUMI_CONFIG_PASSPHRASE`
- More secure than inline passphrase
- File should have `0600` permissions

---

### Logging and Debugging

#### `KEYBASE_DEBUG`

**Description:** Enable debug logging for the Keybase provider.

**Type:** Boolean

**Required:** No

**Default:** `false`

**Valid Values:** `true`, `false`, `1`, `0`

**Example:**
```bash
export KEYBASE_DEBUG="true"
```

**Usage:**
```bash
# Enable debug logging
export KEYBASE_DEBUG="true"

# Disable debug logging
export KEYBASE_DEBUG="false"
```

**Notes:**
- Logs API requests, cache operations, and timing
- **Warning:** May log sensitive information (usernames, key IDs)
- Use only for troubleshooting
- Never enable in production

---

#### `KEYBASE_LOG_LEVEL`

**Description:** Set logging verbosity level.

**Type:** String (enum)

**Required:** No

**Default:** `info`

**Valid Values:** `debug`, `info`, `warn`, `error`

**Example:**
```bash
export KEYBASE_LOG_LEVEL="debug"
```

**Usage:**
```bash
# Debug logging (verbose)
export KEYBASE_LOG_LEVEL="debug"

# Info logging (default)
export KEYBASE_LOG_LEVEL="info"

# Warning logging (quiet)
export KEYBASE_LOG_LEVEL="warn"

# Error logging (silent)
export KEYBASE_LOG_LEVEL="error"
```

**Notes:**
- `debug` includes timing and cache statistics
- `info` includes configuration and operations
- `warn` only shows warnings and errors
- `error` only shows errors

---

## Configuration Files

### Shell Script Configuration

Create a configuration script for easy setup:

**keybase-env.sh:**
```bash
#!/bin/bash
# Keybase Encryption Provider Configuration

# Core settings
export KEYBASE_RECIPIENTS="alice,bob,charlie"
export KEYBASE_FORMAT="saltpack"
export KEYBASE_CACHE_TTL="86400"
export KEYBASE_VERIFY_PROOFS="false"

# Advanced settings
export KEYBASE_CACHE_PATH="$HOME/.config/pulumi/keybase_cache.json"
export KEYBASE_API_TIMEOUT="30"
export KEYBASE_API_MAX_RETRIES="3"
export KEYBASE_API_RETRY_DELAY="1"

# Debug settings (disable in production)
export KEYBASE_DEBUG="false"
export KEYBASE_LOG_LEVEL="info"

echo "Keybase provider configured for recipients: $KEYBASE_RECIPIENTS"
```

**Usage:**
```bash
source keybase-env.sh
pulumi up
```

---

### Docker Environment File

Create a `.env` file for Docker:

**.env:**
```bash
# Keybase Configuration
KEYBASE_RECIPIENTS=alice,bob,charlie
KEYBASE_FORMAT=saltpack
KEYBASE_CACHE_TTL=86400
KEYBASE_VERIFY_PROOFS=false
KEYBASE_CACHE_PATH=/app/cache/keybase_cache.json
KEYBASE_API_TIMEOUT=30
KEYBASE_API_MAX_RETRIES=3
```

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  pulumi:
    image: pulumi/pulumi:latest
    env_file:
      - .env
    volumes:
      - ./cache:/app/cache
      - ~/.config/keybase:/root/.config/keybase:ro
```

---

### CI/CD Configuration

#### GitHub Actions

```yaml
name: Deploy with Pulumi
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Configure Keybase Provider
        env:
          KEYBASE_RECIPIENTS: ${{ secrets.KEYBASE_RECIPIENTS }}
          KEYBASE_FORMAT: saltpack
          KEYBASE_CACHE_TTL: "43200"
          KEYBASE_VERIFY_PROOFS: "true"
        run: |
          echo "Keybase configured"
      
      - name: Deploy
        uses: pulumi/actions@v3
        with:
          command: up
          stack-name: production
```

#### GitLab CI

```yaml
deploy:
  image: pulumi/pulumi:latest
  variables:
    KEYBASE_RECIPIENTS: "alice,bob,charlie"
    KEYBASE_FORMAT: "saltpack"
    KEYBASE_CACHE_TTL: "43200"
    KEYBASE_VERIFY_PROOFS: "true"
  script:
    - pulumi up --yes
```

#### Jenkins

```groovy
pipeline {
    agent any
    environment {
        KEYBASE_RECIPIENTS = 'alice,bob,charlie'
        KEYBASE_FORMAT = 'saltpack'
        KEYBASE_CACHE_TTL = '43200'
        KEYBASE_VERIFY_PROOFS = 'true'
    }
    stages {
        stage('Deploy') {
            steps {
                sh 'pulumi up --yes'
            }
        }
    }
}
```

---

## Environment-Specific Configurations

### Development Environment

```bash
#!/bin/bash
# dev-env.sh

export KEYBASE_RECIPIENTS="dev_team,lead_dev"
export KEYBASE_FORMAT="saltpack"
export KEYBASE_CACHE_TTL="3600"        # 1 hour (short for testing)
export KEYBASE_VERIFY_PROOFS="false"   # Faster for development
export KEYBASE_DEBUG="true"            # Enable debug logging
export KEYBASE_LOG_LEVEL="debug"
```

### Staging Environment

```bash
#!/bin/bash
# staging-env.sh

export KEYBASE_RECIPIENTS="staging_team,qa_lead,dev_lead"
export KEYBASE_FORMAT="saltpack"
export KEYBASE_CACHE_TTL="43200"       # 12 hours
export KEYBASE_VERIFY_PROOFS="false"
export KEYBASE_DEBUG="false"
export KEYBASE_LOG_LEVEL="info"
```

### Production Environment

```bash
#!/bin/bash
# production-env.sh

export KEYBASE_RECIPIENTS="ops_team,security_lead,platform_engineer"
export KEYBASE_FORMAT="saltpack"
export KEYBASE_CACHE_TTL="86400"       # 24 hours
export KEYBASE_VERIFY_PROOFS="true"    # Verify identities
export KEYBASE_DEBUG="false"           # No debug logging
export KEYBASE_LOG_LEVEL="warn"        # Only warnings/errors
export KEYBASE_API_MAX_RETRIES="5"     # More retries for reliability
```

---

## Best Practices

### 1. Use Configuration Files

Store environment variables in configuration files:

```bash
# Load configuration
source ./config/production-env.sh

# Run Pulumi
pulumi up
```

### 2. Never Commit Secrets

Add environment files to `.gitignore`:

```gitignore
# Environment configuration
*-env.sh
.env
.env.*
keybase-env.sh
```

### 3. Use CI/CD Secrets

Store sensitive values in CI/CD secrets:

- GitHub Actions: Repository Secrets
- GitLab CI: CI/CD Variables (masked)
- Jenkins: Credentials Manager

### 4. Validate Configuration

Validate environment variables before use:

```bash
#!/bin/bash

if [ -z "$KEYBASE_RECIPIENTS" ]; then
    echo "Error: KEYBASE_RECIPIENTS not set"
    exit 1
fi

echo "Configuration valid"
pulumi up
```

### 5. Document Requirements

Create a `.env.example` file:

```bash
# Keybase Provider Configuration Template
# Copy to .env and fill in values

# Required: Comma-separated list of recipients
KEYBASE_RECIPIENTS=alice,bob,charlie

# Optional: Encryption format (default: saltpack)
KEYBASE_FORMAT=saltpack

# Optional: Cache TTL in seconds (default: 86400)
KEYBASE_CACHE_TTL=86400

# Optional: Verify identity proofs (default: false)
KEYBASE_VERIFY_PROOFS=false
```

---

## Troubleshooting

### Variable Not Recognized

**Problem:** Environment variable is set but not recognized.

**Solution:**
```bash
# Verify variable is exported
export KEYBASE_RECIPIENTS="alice,bob"

# Check it's set
echo $KEYBASE_RECIPIENTS

# Verify in subshell
env | grep KEYBASE
```

### Invalid Value

**Problem:** Error about invalid configuration value.

**Solution:**
```bash
# Check for typos
echo $KEYBASE_FORMAT

# Verify value is correct
export KEYBASE_FORMAT="saltpack"  # Not "Saltpack" or "SALTPACK"
```

### Variables Not Persisting

**Problem:** Variables reset between commands.

**Solution:**
```bash
# Add to shell profile
echo 'export KEYBASE_RECIPIENTS="alice,bob"' >> ~/.bashrc
source ~/.bashrc

# Or use a configuration script
source ./keybase-env.sh && pulumi up
```

---

## Priority Order

When the same configuration is specified in multiple places, the priority order is:

1. **Environment Variables** (highest priority)
2. **Stack Configuration File** (`Pulumi.<stack>.yaml`)
3. **Command Line Arguments**
4. **Default Values** (lowest priority)

Example:
```bash
# Stack config has: keybase://alice,bob
# Environment has:
export KEYBASE_RECIPIENTS="charlie,dave"

# Result: Uses charlie,dave (environment takes precedence)
```

---

## Security Considerations

### 1. File Permissions

Protect configuration files:

```bash
chmod 600 *-env.sh
chmod 600 .env
```

### 2. Avoid Hardcoding

Never hardcode sensitive values in scripts:

```bash
# Bad
export KEYBASE_RECIPIENTS="alice"

# Good
export KEYBASE_RECIPIENTS="${KEYBASE_RECIPIENTS:-alice}"
```

### 3. Clean Up

Clear sensitive environment variables when done:

```bash
unset KEYBASE_RECIPIENTS
unset KEYBASE_API_KEY  # If applicable
```

### 4. Use Secrets Management

For production, use dedicated secrets management:

- AWS Secrets Manager
- HashiCorp Vault
- Azure Key Vault
- GCP Secret Manager

---

## Related Documentation

- [Pulumi Configuration Guide](PULUMI_CONFIGURATION.md)
- [URL Scheme Documentation](keybase/URL_PARSING.md)
- [Main README](README.md)
- [Cache Configuration](keybase/cache/README.md)

---

## Support

For issues and questions:

- **GitHub Issues:** Report bugs and feature requests
- **Documentation:** See README.md for detailed API documentation
- **Community:** Pulumi Community Slack
