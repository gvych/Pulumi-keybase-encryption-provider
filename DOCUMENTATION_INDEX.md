# Documentation Index

Complete guide to the Keybase Encryption Provider for Pulumi documentation.

## 📚 Documentation Overview

This project includes comprehensive documentation covering setup, configuration, API usage, and examples.

## 🚀 Getting Started

**New to this provider?** Start here:

1. **[Quick Start Guide](QUICKSTART.md)** - 5-minute setup guide
   - Prerequisites and installation
   - Basic configuration
   - Setting and using secrets
   - Complete working example

2. **[Pulumi Configuration Guide](PULUMI_CONFIGURATION.md)** - Comprehensive Pulumi setup
   - Stack configuration files
   - Environment variables
   - Command line options
   - Multi-environment setup
   - Migration from other providers

3. **[Example Configurations](examples/pulumi_configs/)** - Real-world examples
   - Single user setup
   - Team configurations
   - Development/Staging/Production
   - Advanced scenarios

## 📖 User Documentation

### Configuration Guides

| Document | Description | Audience |
|----------|-------------|----------|
| **[Quick Start Guide](QUICKSTART.md)** | Get up and running in 5 minutes | All users |
| **[Pulumi Configuration](PULUMI_CONFIGURATION.md)** | Complete Pulumi setup guide | DevOps, Infrastructure Engineers |
| **[Environment Variables](ENVIRONMENT_VARIABLES.md)** | Environment configuration reference | CI/CD, Automation Engineers |
| **[URL Scheme](keybase/URL_PARSING.md)** | URL format specification | Advanced users, Developers |

### Example Collections

| Directory | Description | Contents |
|-----------|-------------|----------|
| **[Pulumi Configs](examples/pulumi_configs/)** | Complete stack configurations | 8+ real-world examples |
| **[Code Examples](examples/)** | Working Go code | API, cache, credentials examples |

## 🔧 Technical Documentation

### API References

| Document | Description | Use Case |
|----------|-------------|----------|
| **[Cache Manager API](keybase/cache/README.md)** | Public key caching | Programmatic cache management |
| **[API Client](keybase/api/README.md)** | Keybase API integration | Direct API usage |
| **[Credentials](keybase/credentials/README.md)** | Credential discovery | Keybase availability checking |

### Specifications

| Document | Description | Details |
|----------|-------------|---------|
| **[URL Scheme](keybase/URL_PARSING.md)** | URL format spec | Parsing, validation, examples |
| **[Armoring Strategy](ARMORING_STRATEGY.md)** | Encryption format decision | ASCII vs binary, Base62 encoding |
| **[Offline Decryption](OFFLINE_DECRYPTION.md)** | Offline mode and air-gapped environments | Cache management, network independence ⭐ |

## 📂 Documentation by Topic

### Configuration

#### Stack Configuration Files
- [Pulumi Configuration Guide](PULUMI_CONFIGURATION.md#stack-configuration-file)
- [Example: Single User](examples/pulumi_configs/Pulumi.single-user.yaml)
- [Example: Team](examples/pulumi_configs/Pulumi.team.yaml)
- [Example: Production](examples/pulumi_configs/Pulumi.production.yaml)

#### Environment Variables
- [Environment Variables Guide](ENVIRONMENT_VARIABLES.md)
- [Complete Variable Reference](ENVIRONMENT_VARIABLES.md#variable-reference)
- [Configuration Files](ENVIRONMENT_VARIABLES.md#configuration-files)
- [CI/CD Configuration](ENVIRONMENT_VARIABLES.md#cicd-configuration)

#### URL Scheme
- [URL Scheme Specification](keybase/URL_PARSING.md)
- [URL Format](keybase/URL_PARSING.md#url-format)
- [Usage Examples](keybase/URL_PARSING.md#examples)
- [Error Handling](keybase/URL_PARSING.md#error-handling)

### Security

#### Access Control
- [Configuration by Environment](PULUMI_CONFIGURATION.md#configuration-examples-by-use-case)
- [Identity Verification](PULUMI_CONFIGURATION.md#with-identity-verification)
- [Best Practices](PULUMI_CONFIGURATION.md#best-practices)

#### Key Management
- [Public Key Caching](README.md#public-key-caching)
- [Offline Decryption](OFFLINE_DECRYPTION.md) ⭐
- [Cache Configuration](keybase/cache/README.md)
- [Cache TTL Settings](ENVIRONMENT_VARIABLES.md#keybase_cache_ttl)

#### Secrets Management
- [Setting Secrets](QUICKSTART.md#step-3-set-secrets)
- [Using Secrets](QUICKSTART.md#step-4-use-secrets-in-your-code)
- [Viewing Secrets](PULUMI_CONFIGURATION.md#setting-secrets)

### Migration

#### From Other Providers
- [Migration Overview](PULUMI_CONFIGURATION.md#migration-from-other-providers)
- [From Passphrase Provider](PULUMI_CONFIGURATION.md#from-passphrase-provider)
- [From AWS KMS](PULUMI_CONFIGURATION.md#from-aws-kms)

### Troubleshooting

#### Common Issues
- [Quick Start Troubleshooting](QUICKSTART.md#troubleshooting)
- [Configuration Troubleshooting](PULUMI_CONFIGURATION.md#troubleshooting)
- [Environment Variable Issues](ENVIRONMENT_VARIABLES.md#troubleshooting)

#### Error Messages
- [URL Parsing Errors](keybase/URL_PARSING.md#error-handling)
- [API Errors](keybase/api/README.md)
- [Cache Errors](keybase/cache/README.md)

## 📋 Documentation by User Type

### Individual Developers

**What you need:**
1. [Quick Start Guide](QUICKSTART.md) - Setup
2. [Single User Example](examples/pulumi_configs/Pulumi.single-user.yaml) - Configuration
3. [Setting Secrets](QUICKSTART.md#step-3-set-secrets) - Usage

**Recommended reading:**
- [URL Scheme Basics](keybase/URL_PARSING.md#url-format)
- [Cache Configuration](ENVIRONMENT_VARIABLES.md#keybase_cache_ttl)

### Team Leads

**What you need:**
1. [Pulumi Configuration Guide](PULUMI_CONFIGURATION.md) - Complete setup
2. [Team Example](examples/pulumi_configs/Pulumi.team.yaml) - Multi-user config
3. [Access Control](PULUMI_CONFIGURATION.md#best-practices) - Security

**Recommended reading:**
- [Environment-Specific Configs](examples/pulumi_configs/README.md#environment-specific-examples)
- [Identity Verification](PULUMI_CONFIGURATION.md#with-identity-verification)
- [Best Practices](PULUMI_CONFIGURATION.md#best-practices)

### DevOps Engineers

**What you need:**
1. [Pulumi Configuration Guide](PULUMI_CONFIGURATION.md) - Full configuration
2. [Environment Variables](ENVIRONMENT_VARIABLES.md) - Automation setup
3. [CI/CD Configuration](ENVIRONMENT_VARIABLES.md#cicd-configuration) - Pipeline integration
4. [Multi-Environment Setup](examples/pulumi_configs/README.md) - Dev/Staging/Prod

**Recommended reading:**
- [Cache Management](keybase/cache/README.md)
- [Performance Considerations](PULUMI_CONFIGURATION.md#performance-considerations)
- [Security Best Practices](PULUMI_CONFIGURATION.md#security-considerations)

### Security Engineers

**What you need:**
1. [Security Considerations](PULUMI_CONFIGURATION.md#security-considerations)
2. [Identity Verification](PULUMI_CONFIGURATION.md#with-identity-verification)
3. [Access Control](PULUMI_CONFIGURATION.md#best-practices)
4. [Production Config](examples/pulumi_configs/Pulumi.production.yaml)

**Recommended reading:**
- [Encryption Architecture](README.md#architecture)
- [Key Management](README.md#public-key-caching)
- [Audit Practices](PULUMI_CONFIGURATION.md#best-practices)

### Application Developers

**What you need:**
1. [Using Secrets in Code](QUICKSTART.md#step-4-use-secrets-in-your-code) - Integration
2. [Code Examples](examples/) - Working examples
3. [API Reference](README.md#api-reference) - Programmatic usage

**Recommended reading:**
- [TypeScript Example](QUICKSTART.md#typescriptjavascript)
- [Python Example](QUICKSTART.md#python)
- [Go Example](QUICKSTART.md#go)

### System Administrators

**What you need:**
1. [Prerequisites](QUICKSTART.md#prerequisites) - Installation
2. [Environment Variables](ENVIRONMENT_VARIABLES.md) - System config
3. [Credential Discovery](keybase/credentials/README.md) - Keybase setup

**Recommended reading:**
- [File Permissions](ENVIRONMENT_VARIABLES.md#1-file-permissions)
- [Cache Location](ENVIRONMENT_VARIABLES.md#keybase_cache_path)
- [Troubleshooting](QUICKSTART.md#troubleshooting)

## 🔍 Documentation by Use Case

### Setting Up for the First Time

1. Read [Quick Start Guide](QUICKSTART.md)
2. Follow [Prerequisites](QUICKSTART.md#prerequisites)
3. Use [Minimal Example](examples/pulumi_configs/Pulumi.minimal.yaml)
4. Set your first secret using [Step 3](QUICKSTART.md#step-3-set-secrets)

### Configuring for a Team

1. Review [Team Configuration](PULUMI_CONFIGURATION.md#small-team)
2. Use [Team Example](examples/pulumi_configs/Pulumi.team.yaml)
3. Set up [Access Control](PULUMI_CONFIGURATION.md#best-practices)
4. Document using [Best Practices](PULUMI_CONFIGURATION.md#best-practices)

### Setting Up Multiple Environments

1. Review [Multi-Stack Setup](PULUMI_CONFIGURATION.md#multi-stack-setup)
2. Copy examples:
   - [Development](examples/pulumi_configs/Pulumi.dev.yaml)
   - [Staging](examples/pulumi_configs/Pulumi.staging.yaml)
   - [Production](examples/pulumi_configs/Pulumi.production.yaml)
3. Configure [Per Environment](PULUMI_CONFIGURATION.md#configuration-examples-by-use-case)
4. Follow [Best Practices](PULUMI_CONFIGURATION.md#best-practices)

### Integrating with CI/CD

1. Review [CI/CD Configuration](ENVIRONMENT_VARIABLES.md#cicd-configuration)
2. Set up [Environment Variables](ENVIRONMENT_VARIABLES.md#variable-reference)
3. Configure for your platform:
   - [GitHub Actions](ENVIRONMENT_VARIABLES.md#github-actions)
   - [GitLab CI](ENVIRONMENT_VARIABLES.md#gitlab-ci)
   - [Jenkins](ENVIRONMENT_VARIABLES.md#jenkins)

### Migrating from Another Provider

1. Review [Migration Guide](PULUMI_CONFIGURATION.md#migration-from-other-providers)
2. Follow specific migration:
   - [From Passphrase](PULUMI_CONFIGURATION.md#from-passphrase-provider)
   - [From AWS KMS](PULUMI_CONFIGURATION.md#from-aws-kms)
3. Update configuration files
4. Test with [Verification Steps](PULUMI_CONFIGURATION.md#migration-from-other-providers)

### Troubleshooting Issues

1. Check [Common Issues](QUICKSTART.md#common-issues)
2. Review specific troubleshooting:
   - [Quick Start Issues](QUICKSTART.md#troubleshooting)
   - [Configuration Issues](PULUMI_CONFIGURATION.md#troubleshooting)
   - [Environment Variable Issues](ENVIRONMENT_VARIABLES.md#troubleshooting)
3. Check error-specific docs:
   - [URL Parsing Errors](keybase/URL_PARSING.md#error-handling)
   - [API Errors](keybase/api/README.md)

### Advanced Configuration

1. Review [Advanced Configuration](QUICKSTART.md#advanced-configuration)
2. Explore advanced examples:
   - [No Cache](examples/pulumi_configs/Pulumi.no-cache.yaml)
   - [Legacy PGP](examples/pulumi_configs/Pulumi.legacy-pgp.yaml)
3. Read [Performance Considerations](PULUMI_CONFIGURATION.md#performance-considerations)
4. Implement [Security Best Practices](PULUMI_CONFIGURATION.md#security-considerations)

## 📊 Documentation Statistics

### User Guides
- **Quick Start Guide:** 1 comprehensive guide
- **Configuration Guides:** 3 detailed guides
- **Troubleshooting Guides:** Integrated in all docs

### Technical Documentation
- **API References:** 3 complete references
- **Specifications:** 2 complete specs (URL scheme, Armoring strategy)
- **Architecture Docs:** Integrated in main README

### Examples
- **Pulumi Configuration Examples:** 8 complete stack examples
- **Code Examples:** 5 working Go programs
- **Integration Examples:** CI/CD for 3 platforms

### Total Documentation
- **Total Files:** 21+ documentation files
- **Total Examples:** 13+ complete examples
- **Total Words:** 55,000+ words of documentation

## 🔗 Quick Navigation

### Most Viewed Pages
1. [Quick Start Guide](QUICKSTART.md) ⭐
2. [Pulumi Configuration](PULUMI_CONFIGURATION.md) ⭐
3. [Example Configs](examples/pulumi_configs/) ⭐
4. [Environment Variables](ENVIRONMENT_VARIABLES.md)
5. [URL Scheme](keybase/URL_PARSING.md)

### Most Useful Examples
1. [Production Config](examples/pulumi_configs/Pulumi.production.yaml) ⭐
2. [Team Config](examples/pulumi_configs/Pulumi.team.yaml) ⭐
3. [Development Config](examples/pulumi_configs/Pulumi.dev.yaml) ⭐
4. [Staging Config](examples/pulumi_configs/Pulumi.staging.yaml)
5. [Single User Config](examples/pulumi_configs/Pulumi.single-user.yaml)

### Most Useful Guides
1. [Setting Secrets](QUICKSTART.md#step-3-set-secrets) ⭐
2. [Using Secrets in Code](QUICKSTART.md#step-4-use-secrets-in-your-code) ⭐
3. [Multi-Environment Setup](PULUMI_CONFIGURATION.md#multi-stack-setup) ⭐
4. [CI/CD Integration](ENVIRONMENT_VARIABLES.md#cicd-configuration)
5. [Migration Guide](PULUMI_CONFIGURATION.md#migration-from-other-providers)

## 📝 Documentation Feedback

Found an issue or have a suggestion? 

- **GitHub Issues:** Report documentation issues
- **Pull Requests:** Contribute improvements
- **Community:** Discuss in Pulumi Community Slack

## 🏗️ Documentation Structure

```
/
├── README.md                          # Main project overview
├── QUICKSTART.md                      # 5-minute setup guide
├── DOCUMENTATION_INDEX.md             # This file
├── PULUMI_CONFIGURATION.md            # Complete Pulumi setup
├── ENVIRONMENT_VARIABLES.md           # Environment variable reference
├── ARMORING_STRATEGY.md               # Encryption format decision
├── OFFLINE_DECRYPTION.md              # Offline mode guide
│
├── keybase/
│   ├── URL_PARSING.md                 # URL scheme specification
│   ├── cache/
│   │   └── README.md                  # Cache API reference
│   ├── api/
│   │   └── README.md                  # API client reference
│   └── credentials/
│       └── README.md                  # Credentials discovery
│
└── examples/
    ├── README.md                      # Examples overview
    ├── pulumi_configs/
    │   ├── README.md                  # Configuration examples guide
    │   ├── Pulumi.yaml                # Main project file
    │   ├── Pulumi.dev.yaml            # Development config
    │   ├── Pulumi.staging.yaml        # Staging config
    │   ├── Pulumi.production.yaml     # Production config
    │   ├── Pulumi.single-user.yaml    # Single user config
    │   ├── Pulumi.team.yaml           # Team config
    │   ├── Pulumi.minimal.yaml        # Minimal config
    │   ├── Pulumi.no-cache.yaml       # No cache config
    │   └── Pulumi.legacy-pgp.yaml     # Legacy PGP config
    │
    ├── api/                           # API client example
    ├── basic/                         # Basic usage example
    ├── credentials/                   # Credentials example
    ├── custom/                        # Custom config example
    └── url_parsing/                   # URL parsing example
```

## 🎯 Where to Start

**Choose your starting point based on your goal:**

- **Just getting started?** → [Quick Start Guide](QUICKSTART.md)
- **Setting up a team?** → [Pulumi Configuration](PULUMI_CONFIGURATION.md)
- **Configuring CI/CD?** → [Environment Variables](ENVIRONMENT_VARIABLES.md)
- **Need an example?** → [Example Configs](examples/pulumi_configs/)
- **Troubleshooting?** → [Common Issues](QUICKSTART.md#common-issues)
- **API documentation?** → [API References](#api-references)

## 📚 Related Resources

### External Documentation
- [Keybase Documentation](https://keybase.io/docs)
- [Saltpack Specification](https://saltpack.org/)
- [Pulumi Documentation](https://www.pulumi.com/docs/)
- [Go Cloud Development Kit](https://gocloud.dev/howto/secrets/)

### Community
- [Pulumi Community Slack](https://pulumi-community.slack.com/)
- [Keybase Teams](https://keybase.io/docs/teams)

---

**Last Updated:** December 26, 2025

**Documentation Version:** 1.0.0 (Phase 1)
