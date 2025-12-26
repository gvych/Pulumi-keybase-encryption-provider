### Architecture Overview

The Keybase provider integrates with Pulumi's Go Cloud Development Kit through the standard `driver.Keeper` interface. The architecture comprises three layers:

1. **API Integration Layer**: Calls Keybase REST API to lookup user public keys and resolve usernames
2. **Cryptographic Layer**: Uses the `github.com/keybase/saltpack` Go library for encryption/decryption with native multiple-recipient support
3. **Keyring Integration Layer**: Accesses local Keybase keyring for decryption operations

Encryption flow: Pulumi secrets → Recipient usernames → API lookup → Public keys → Saltpack encryption → Ciphertext. Decryption flow: Ciphertext → Saltpack decryption (auto-finds matching private key) → Plaintext.

### Keybase API \& Encryption Specifications

#### REST API for Public Key Lookup

Keybase provides a public REST API endpoint at `https://keybase.io/_/api/1.0/user/lookup.json` that accepts comma-separated usernames and returns user objects including their public keys. The response includes the `public_keys.primary` field containing both key ID and the full PGP public key bundle. The API has CORS enabled, supporting multiple simultaneous lookups in a single request. Caching mechanisms should implement 24-hour TTLs to reduce API calls, with invalidation triggered by key rotation events.[^1]

#### Saltpack Encryption Format

Keybase's modern encryption format, Saltpack, provides native multiple-recipient support through the `github.com/keybase/saltpack` Go library. The encryption mechanism encrypts data once with a symmetric session key (ChaCha20-Poly1305), then encrypts this session key separately for each recipient using NaCl Box (Curve25519-based DHKE + XSalsa20-Poly1305). The message header contains an array of encrypted session keys (one per recipient), allowing any recipient with a matching private key to decrypt.[^2]

Key functions for multiple recipients:

```go
// Encrypts plaintext to multiple recipients
func Seal(version Version, plaintext []byte, sender BoxSecretKey, 
  receivers []BoxPublicKey) (out []byte, err error)

// Returns MessageKeyInfo indicating which recipient key was used for decryption
func Open(versionValidator VersionValidator, ciphertext []byte, 
  keyring Keyring) (i *MessageKeyInfo, plaintext []byte, err error)
```


#### Multiple Recipients Capability

The Saltpack format's `receivers []saltpack.BoxPublicKey` parameter accepts an arbitrary-length slice of recipient public keys. Each recipient receives an independently encrypted copy of the session key in the message header. Decryption automatically attempts to recover the session key using the decrypting party's private key, returning `MessageKeyInfo` that indicates which recipient key was successfully used. This design prevents enumeration attacks (no recipient list visible to attackers) while supporting team and multi-user scenarios.[^2]

### URL Scheme Design

The Keybase provider uses a custom URL scheme for configuration:

```
keybase://user1,user2,user3?format=saltpack&cache_ttl=86400
```

| Component | Description | Required |
| :-- | :-- | :-- |
| `keybase://` | Scheme identifier | Yes |
| `user1,user2,user3` | Comma-separated recipient usernames | Yes |
| `format` | Encryption format: `saltpack` (default) or `pgp` | No |
| `cache_ttl` | Public key cache TTL in seconds (default: 86400) | No |
| `verify_proofs` | Require identity proof verification (boolean) | No |

Stack configuration example:

```yaml
secretsprovider: keybase://alice,bob,charlie?format=saltpack
encryptedkey: <encrypted-dek>
```


### Core Interface Implementation Requirements

The `driver.Keeper` interface requires five methods:

**Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)**: Accepts plaintext and returns Saltpack-encrypted ciphertext for all configured recipients. Must:

- Fetch public keys for each recipient via API lookup (with caching)
- Call `saltpack.Seal()` with sender's secret key and recipient public keys
- Return binary-encoded ciphertext

**Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)**: Accepts ciphertext and returns plaintext using the local Keybase keyring. Must:

- Load user's private key from `~/.config/keybase/`
- Call `saltpack.Open()` with the ciphertext and keyring
- Return plaintext

**Close() error**: Releases any open resources (cache connections, HTTP clients).

**ErrorAs(err error, i any) bool**: Maps Keybase and Saltpack errors to driver-specific types, enabling downstream error handling.

**ErrorCode(error) gcerrors.ErrorCode**: Maps all errors to standard Go Cloud error codes (InvalidArgument, NotFound, Internal, Unavailable).

### Development Task List

#### Phase 1: Keybase Integration \& Public Key Fetching (Weeks 1-2)

1. **Design public key caching strategy**: Decide on cache invalidation (TTL-based vs. event-based). Design cache file format (JSON with expiration timestamps). Plan handling of key rotation events where public keys change. Implement separate cache for each Keybase username.
2. **Implement Keybase API client wrapper**: Create Go HTTP client for `https://keybase.io/_/api/1.0/user/lookup.json`. Handle comma-separated username lists. Parse JSON response and extract `public_keys.primary.bundle`. Implement retry logic with exponential backoff for transient failures.
3. **Parse URL scheme and configuration**: Parse `keybase://user1,user2,user3?format=saltpack` to extract recipient list and options. Validate username format (alphanumeric + underscore). Validate format parameter (saltpack or pgp).
4. **Implement public key caching layer**: Write cache file to `~/.config/pulumi/keybase_keyring_cache.json`. Implement TTL checking with timestamp comparison. Add cache invalidation on demand. Return cached keys if valid, otherwise fetch fresh.
5. **Handle API errors and rate limiting**: Implement 429 status code handling with backoff. Wrap `net.Error` and `context.DeadlineExceeded` with proper error codes. Provide clear error messages for network failures vs. user lookup failures.
6. **Implement credential discovery**: Detect if Keybase CLI is installed and configured. Verify Keybase user is logged in. Read authentication status from Keybase directory. Fail gracefully with clear error if Keybase not available.
7. **Test API integration with mock server**: Create mock HTTP server returning Keybase API responses. Test single and multiple username lookups. Test cache behavior and TTL expiration. Test error responses (404, 500, timeout).
8. **Create example configuration**: Document URL scheme with examples. Create sample Pulumi.yaml showing keybase provider configuration. Document environment variables needed.

#### Phase 2: Encryption Implementation with Multiple Recipients (Weeks 2-3)

9. **Import and integrate saltpack library**: Add `github.com/keybase/saltpack` to go.mod. Review saltpack API documentation. Plan which functions to use (streaming vs. non-streaming). Test saltpack examples.
10. **Create ephemeral key generation wrapper**: Implement `EphemeralKeyCreator` for generating temporary keys. Use `crypto/rand` for randomness. Ensure proper error handling for insufficient entropy.
11. **Implement recipient public key conversion**: Convert PGP public key bundle (ASCII text) to saltpack `BoxPublicKey`. Parse `public_keys.primary.kid` as key identifier. Validate key format and size. Handle both PGP and saltpack key formats.
12. **Implement Encrypt method with multiple recipients**: Fetch all recipient public keys via API/cache. Create `[]saltpack.BoxPublicKey` array from fetched keys. Call `saltpack.EncryptArmor62Seal()` for ASCII output or `saltpack.Seal()` for binary. Encode result appropriately (base64 or hex for state file storage).
13. **Handle sender key configuration**: Determine sender identity (current Keybase user or configured username). Load sender's private key from `~/.config/keybase/`. Validate sender key format. Handle missing or invalid sender key.
14. **Implement armoring strategy**: Decide binary vs. ASCII armoring for state files. Binary is more compact; ASCII is more readable/diffable. Implement Base62 armoring if ASCII chosen. Document choice in comments.
15. **Optimize for large messages**: Implement streaming encryption using `saltpack.NewEncryptArmor62Stream()` for messages >10 MiB. Avoid loading entire ciphertext in memory. Write directly to buffer/file.
16. **Test encryption with multiple recipients**: Create unit tests with 1, 5, 10 recipients. Verify all recipients can decrypt independently. Test very large messages. Benchmark encryption latency (<500ms target).

#### Phase 3: Decryption \& Keyring Integration (Weeks 3-4)

17. **Implement Keybase keyring loading**: Read private keys from `~/.config/keybase/`. Implement `saltpack.Keyring` interface by wrapping Keybase key storage. Cache loaded keys in memory with TTL.
18. **Parse saltpack message header**: Deserialize message format to identify recipients. Extract `MessageKeyInfo` from `saltpack.Open()`. Determine which recipient key was used for decryption.
19. **Implement Decrypt method**: Call `saltpack.Open()` with ciphertext and keyring. Handle case where multiple recipients exist but only current user's key is available. Return plaintext on success.
20. **Implement error handling for decryption**: Map `saltpack.ErrBadCiphertext` to GCError InvalidArgument. Map `ErrNoDecryptionKey` to GCError NotFound. Map timeout/network errors to Unavailable. Provide detailed error messages.
21. **Add key rotation support**: Detect when old messages use retired keys. Implement lazy re-encryption path. Document that re-decryption with new keys may be needed after rotation.
22. **Implement offline decryption fallback**: Allow decryption without network access (keys already fetched). Document that initial encryption requires network for key lookup, but later decryption works offline.
23. **Test decryption with multiple recipients**: Create messages encrypted to 3+ recipients. Test each recipient can decrypt independently. Test decryption fails gracefully for non-recipients. Test with expired/revoked keys.

#### Phase 4: Provider Integration with Pulumi (Weeks 4-5)

24. **Implement Keeper interface completely**: Implement all five required methods (Encrypt, Decrypt, Close, ErrorAs, ErrorCode). Ensure proper context propagation and cancellation. Validate input/output sizes.
25. **Implement URLOpener interface**: Create function to parse keybase:// URL and return initialized Keeper. Register with Go CDK's URL multiplexer. Support URL parameters for format, TTL, etc.
26. **Create provider package structure**: Organize code into: `keybase/` main package, `keybase/internal/` for private APIs, `keybase/api/` for REST client, `keybase/crypto/` for encryption logic. Use clear separation of concerns.
27. **Implement blank import registration**: Create `register.go` with init() function that registers the driver. Make registration automatic on import. Document import requirement in README.
28. **Build and version the provider**: Set up version numbering (semver). Create Go build script. Set build metadata and version strings. Test reproducible builds.
29. **Create integration with Pulumi SDK**: Write integration code that bridges provider to Pulumi's secrets manager. Test with actual `pulumi stack init --secrets-provider=keybase://...` command. Verify secrets encrypt/decrypt in real stacks.
30. **Document provider usage in Pulumi context**: Write guide showing how to use keybase provider with Pulumi. Document configuration precedence (URL params vs. env vars). Show example stack files and configurations.

#### Phase 5: Testing \& Validation (Weeks 5-7)

31. **Write comprehensive unit tests**: Test Encrypt/Decrypt with 1-10 recipients. Test error paths (missing users, invalid keys, network errors). Test URL parsing. Test caching. Aim for >90% code coverage.
32. **Integration testing with real Keybase**: Test against production Keybase API. Use multiple real test users. Verify encryption/decryption works end-to-end. Test rate limiting behavior.
33. **Test with Pulumi stacks**: Create test Pulumi program using `keybase://` provider. Test `pulumi config set --secret`. Test `pulumi preview` and `pulumi up`. Verify secrets stay encrypted in state file.
34. **Performance and load testing**: Benchmark encrypt/decrypt latency with different message sizes. Test with 100+ recipients. Measure API call impact on performance. Implement caching effectiveness metrics.
35. **Security testing**: Verify secrets never appear in logs. Test with invalid/revoked keys. Test with corrupted ciphertext. Verify proper key cleanup in memory. Audit error messages for information leaks.
36. **Test key rotation scenarios**: Encrypt with key version 1, then rotate. Verify old ciphertexts still decrypt. Test adding new recipients. Test removing recipients. Document migration path.
37. **Test edge cases**: Empty messages, very large messages (>1GB streaming). Messages with special characters. Unicode content. Binary data. Message truncation detection.
38. **Cross-platform testing**: Test on Linux, macOS, Windows. Verify keyring access paths on each OS. Test with different Keybase client versions. Document OS-specific requirements.

#### Phase 6: Documentation \& Release (Weeks 7-8)

39. **Write user documentation**: Create setup guide (installing Keybase, configuring stack). Document URL scheme syntax with examples. Document environment variables. Create FAQ and troubleshooting guide. Include examples of 1-recipient and multi-recipient scenarios.
40. **Create migration guides**: Document how to migrate from `--secrets-provider passphrase` to `--secrets-provider keybase://...`. Document re-encryption process. Explain backward compatibility (old stacks still decrypt).
41. **Write API documentation**: Document Go API for developers extending the provider. Document key interfaces (Keeper, URLOpener). Include code examples for each function.
42. **Submit to Pulumi Registry**: Package provider with README, LICENSE, examples. Follow Pulumi packaging standards. Submit for listing in official registry. Maintain registry entry.
43. **Set up CI/CD pipeline**: Create GitHub Actions workflow for testing on each commit. Automated build and test for multiple platforms. Automated release creation on tags. Publish to GitHub releases and registry.

### Complete Task Checklist with Details

| Phase | Task | Details | Status |
| :-- | :-- | :-- | :-- |
| **1** | Public key caching strategy | TTL-based with 24h default, JSON format with timestamps | [ ] |
| **1** | Keybase API client wrapper | HTTP client, comma-separated usernames, JSON parsing, retry logic | [ ] |
| **1** | URL scheme parsing | Extract recipients and options from keybase://user1,user2?format=saltpack | [ ] |
| **1** | Caching implementation | ~/.config/pulumi/keybase_keyring_cache.json, TTL checking, invalidation | [ ] |
| **1** | API error handling | Rate limiting (429), network errors, user lookup failures | [ ] |
| **1** | Credential discovery | Detect Keybase CLI, verify login status, fail gracefully | [ ] |
| **1** | API integration tests | Mock server, single/multi-user lookups, cache behavior, error responses | [ ] |
| **1** | Example configuration | URL scheme docs, sample Pulumi.yaml, environment variables | [ ] |
| **2** | Saltpack library integration | Import saltpack, review API, plan streaming vs. non-streaming | [ ] |
| **2** | Ephemeral key generation | Implement EphemeralKeyCreator, use crypto/rand, error handling | [ ] |
| **2** | PGP to saltpack conversion | Parse ASCII PGP keys, create BoxPublicKey, validate format | [ ] |
| **2** | Encrypt method implementation | Fetch keys, create recipient array, call saltpack.Seal() | [ ] |
| **2** | Sender key handling | Detect current user, load private key, validate format | [ ] |
| **2** | Armoring strategy | Choose binary vs. ASCII, implement Base62 encoding | [ ] |
| **2** | Streaming encryption | Use saltpack.NewEncryptArmor62Stream() for large messages | [ ] |
| **2** | Encryption testing | Unit tests with 1-10 recipients, large messages, benchmarks | [ ] |
| **3** | Keyring loading | Read ~/.config/keybase/, implement saltpack.Keyring | [ ] |
| **3** | Message header parsing | Deserialize format, extract MessageKeyInfo | [ ] |
| **3** | Decrypt method implementation | Call saltpack.Open(), handle multiple recipients | [ ] |
| **3** | Decryption error handling | Map errors to GCError codes, detailed messages | [ ] |
| **3** | Key rotation support | Lazy re-encryption, document migration path | [ ] |
| **3** | Offline decryption | Cache keys locally, work without network after initial setup | [ ] |
| **3** | Decryption testing | Multi-recipient messages, independent decryption, revoked keys | [ ] |
| **4** | Keeper interface completion | All 5 methods, context propagation, size validation | [ ] |
| **4** | URLOpener implementation | Parse URLs, register with CDK, support parameters | [ ] |
| **4** | Package structure | keybase/, internal/, api/, crypto/ directories | [ ] |
| **4** | Driver registration | init() function, automatic on import, document requirement | [ ] |
| **4** | Build \& versioning | Semver, build script, metadata strings, reproducible builds | [ ] |
| **4** | Pulumi SDK integration | Test `pulumi stack init --secrets-provider=keybase://...` | [ ] |
| **4** | Usage documentation | Configuration guide, precedence, example stack files | [ ] |
| **5** | Unit tests | >90% coverage, all error paths, URL parsing, caching | [ ] |
| **5** | Real Keybase testing | Production API, multiple users, encryption/decryption end-to-end | [ ] |
| **5** | Pulumi stack testing | config set --secret, preview, up, state file encryption | [ ] |
| **5** | Performance testing | Latency benchmarks, 100+ recipients, caching effectiveness | [ ] |
| **5** | Security testing | No secret logging, invalid keys, corrupted ciphertext, key cleanup | [ ] |
| **5** | Key rotation testing | Decrypt old ciphertexts, add/remove recipients, document path | [ ] |
| **5** | Edge case testing | Empty/large messages, special characters, Unicode, binary data | [ ] |
| **5** | Cross-platform testing | Linux/macOS/Windows, keyring paths, Keybase versions | [ ] |
| **6** | User documentation | Setup guide, URL syntax, environment variables, FAQ | [ ] |
| **6** | Migration guides | From passphrase provider, re-encryption, backward compatibility | [ ] |
| **6** | API documentation | Go API, interfaces, code examples | [ ] |
| **6** | Registry submission | Packaging, README, LICENSE, follow standards | [ ] |
| **6** | CI/CD pipeline | GitHub Actions, multi-platform testing, automated releases | [ ] |

### Key Technical Decisions

**Encryption Format**: Saltpack is recommended over legacy PGP due to simpler implementation, modern cryptography, and native multiple-recipient support. Saltpack's API directly accepts `receivers []BoxPublicKey`, eliminating manual PGP recipient handling.

**Caching Strategy**: 24-hour TTL with on-demand fetch provides good balance between API quota efficiency and freshness. Cache invalidation on key rotation requires monitoring Keybase activity (advanced feature for later phases).

**Armoring**: ASCII-armored Base62 (Saltpack's native armoring) produces more readable state files than binary, aiding debugging while maintaining security. No additional encoding layer needed.

**Keyring Integration**: Using local `~/.config/keybase/` directory rather than spawning `keybase` CLI subprocess provides better performance and clearer error handling. Requires Keybase to be installed and configured locally.

**Error Handling**: All Keybase-specific errors (API, key lookup, cryptographic failures) map to standard Go Cloud error codes for consistency with other Pulumi providers.

### Dependencies and Libraries

- **Go 1.18+** for generics and modern language features
- **github.com/keybase/saltpack**: Encryption format and Go library
- **net/http**: Standard library for REST API calls
- **encoding/json**: Standard library for JSON parsing
- **gocloud.dev/secrets**: Pulumi's secrets abstraction layer
- **crypto/rand**: Standard library for randomness


### Success Criteria

A production-ready Keybase encryption provider must:

1. Successfully encrypt secrets for 1-100 Keybase users in a single operation
2. Allow any recipient to independently decrypt using their private key
3. Support end-to-end encryption/decryption via `pulumi config set --secret` and state file access
4. Provide <500ms encrypt/decrypt latency at p95 (excluding network API calls)
5. Implement public key caching reducing API calls by >80% in typical usage
6. Pass all unit and integration tests with >90% code coverage
7. Support key rotation without breaking old state files
8. Work across Linux, macOS, and Windows with proper path handling
9. Include comprehensive documentation covering setup, usage, and troubleshooting
10. Be maintainable and extendable by community developers

The development timeline is approximately 8 weeks for a fully functional, well-tested, production-ready implementation.
<span style="display:none">[^10][^11][^12][^13][^14][^15][^16][^17][^18][^19][^20][^21][^22][^23][^24][^25][^26][^27][^28][^29][^3][^30][^31][^32][^33][^34][^35][^4][^5][^6][^7][^8][^9]</span>

<div align="center">⁂</div>

[^1]: https://keybase.io/docs/api/1.0/call/user/lookup

[^2]: https://pkg.go.dev/github.com/keybase/saltpack

[^3]: https://www.zesty.io/mindshare/developer-how-tos/sending-encrypted-files-with-keybase/

[^4]: https://github.com/keybase/keybase-issues/issues/160

[^5]: https://book.keybase.io/docs/crypto/key-exchange

[^6]: https://keybase.io/kbpgp/docs/decrypting

[^7]: https://book.keybase.io/docs/cli

[^8]: https://chrislaing.net/blog/keybase-keys-for-everyone-part-ii/

[^9]: https://developer.hashicorp.com/vault/docs/concepts/pgp-gpg-keybase

[^10]: https://gist.github.com/scottrigby/9c03b0db6100285d5b032b87fac00b8a

[^11]: https://book.keybase.io/docs/crypto

[^12]: http://keybase-python-api.readthedocs.io/en/latest/keybase.html

[^13]: https://www.reddit.com/r/Keybase/comments/qpiagq/how_exactly_are_files_encrypted_with_kbfs/

[^14]: https://www.andreagrandi.it/posts/keybase-pgp-encryption-made-easy/

[^15]: https://code.tutsplus.com/its-time-to-encrypt-your-email-using-keybase--cms-23724t

[^16]: https://saltpack.org/encryption-format-v2

[^17]: https://www.sitepoint.com/keybase-sending-receiving-and-sharing-encrypted-messages/

[^18]: https://treblle.com/blog/secure-your-first-rest-api

[^19]: https://saltpack.org/encryption-format-v1

[^20]: https://davidwinter.dev/2019/04/25/managing-gpg-with-keybase/

[^21]: https://stackoverflow.com/questions/597188/encryption-decryption-with-multiple-keys

[^22]: https://lists.gnupg.org/pipermail/gnupg-devel/2001-August/017540.html

[^23]: https://keybase-python-api.readthedocs.io

[^24]: https://rdrr.io/github/hrbrmstr/keybase/man/

[^25]: https://github.com/jamesjulich/saltpack4j

[^26]: https://keybase.io/docs/api/1.0/call/sig/post_auth

[^27]: https://news.ycombinator.com/item?id=11037257

[^28]: https://keybase.io/docs/api/1.0/call/signup

[^29]: https://github.com/keybase/client

[^30]: https://keybase.io/docs/api/1.0/call/user/key.asc

[^31]: https://github.com/keybase/client/issues/24614

[^32]: https://keybase.io/blog/crypto

[^33]: https://blog.aelterman.com/2020/02/22/importing-a-key-or-key-pair-in-keybase-on-windows/

[^34]: https://www.reddit.com/r/crypto/comments/auwrz8/encrypting_a_file_where_multiple_keys_can_decrypt/

[^35]: https://www.couchbase.com/forums/t/java-sdk-3-1-5-multiple-keys-for-encrypting-documents-in-the-same-application/30763

