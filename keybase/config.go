package keybase

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
)

// EncryptionFormat represents the encryption format to use
type EncryptionFormat string

const (
	// FormatSaltpack uses the modern Saltpack encryption format (default)
	FormatSaltpack EncryptionFormat = "saltpack"
	// FormatPGP uses legacy PGP encryption format
	FormatPGP EncryptionFormat = "pgp"
)

// Config represents parsed configuration from a Keybase URL scheme
type Config struct {
	// Recipients is the list of Keybase usernames to encrypt for
	Recipients []string

	// Format specifies the encryption format (saltpack or pgp)
	Format EncryptionFormat

	// CacheTTL is the time-to-live for cached public keys
	CacheTTL time.Duration

	// VerifyProofs requires identity proof verification
	VerifyProofs bool
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		Recipients:   []string{},
		Format:       FormatSaltpack,
		CacheTTL:     24 * time.Hour, // 24 hours default
		VerifyProofs: false,
	}
}

// ParseURL parses a Keybase URL scheme and returns a Config
//
// URL format: keybase://user1,user2,user3?format=saltpack&cache_ttl=86400&verify_proofs=true
//
// Components:
//   - Scheme: Must be "keybase"
//   - Host/Path: Comma-separated list of recipient usernames
//   - Query parameters:
//     - format: "saltpack" (default) or "pgp"
//     - cache_ttl: Cache TTL in seconds (default: 86400)
//     - verify_proofs: Require identity proof verification (default: false)
func ParseURL(rawURL string) (*Config, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Validate scheme
	if u.Scheme != "keybase" {
		return nil, fmt.Errorf("invalid URL scheme: expected 'keybase', got '%s'", u.Scheme)
	}

	// Start with default config
	config := DefaultConfig()

	// Extract recipients from host and path
	// The URL format is: keybase://user1,user2,user3
	// url.Parse treats the part after :// and before ? as host+path
	recipients := u.Host
	if recipients == "" {
		// Some URL parsers might put it in Opaque field
		recipients = u.Opaque
	}

	// Remove leading slashes if present (from path component)
	recipients = strings.TrimPrefix(recipients, "//")
	recipients = strings.TrimPrefix(recipients, "/")

	// Also check the path for recipients (in case they're there)
	if recipients == "" && u.Path != "" {
		recipients = strings.TrimPrefix(u.Path, "/")
	}

	if recipients == "" {
		return nil, fmt.Errorf("no recipients specified in URL")
	}

	// Split by comma and validate each username
	recipientList := strings.Split(recipients, ",")
	validRecipients := make([]string, 0, len(recipientList))

	for _, recipient := range recipientList {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			continue
		}

		// Validate username format
		if err := api.ValidateUsername(recipient); err != nil {
			return nil, fmt.Errorf("invalid recipient username '%s': %w", recipient, err)
		}

		validRecipients = append(validRecipients, recipient)
	}

	if len(validRecipients) == 0 {
		return nil, fmt.Errorf("no valid recipients specified in URL")
	}

	config.Recipients = validRecipients

	// Parse query parameters
	query := u.Query()

	// Parse format parameter
	if formatStr := query.Get("format"); formatStr != "" {
		format := EncryptionFormat(strings.ToLower(formatStr))
		if err := ValidateFormat(format); err != nil {
			return nil, fmt.Errorf("invalid format parameter: %w", err)
		}
		config.Format = format
	}

	// Parse cache_ttl parameter
	if cacheTTLStr := query.Get("cache_ttl"); cacheTTLStr != "" {
		cacheTTLSeconds, err := strconv.ParseInt(cacheTTLStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cache_ttl parameter: %w", err)
		}
		if cacheTTLSeconds < 0 {
			return nil, fmt.Errorf("cache_ttl must be non-negative, got %d", cacheTTLSeconds)
		}
		config.CacheTTL = time.Duration(cacheTTLSeconds) * time.Second
	}

	// Parse verify_proofs parameter
	if verifyProofsStr := query.Get("verify_proofs"); verifyProofsStr != "" {
		verifyProofs, err := strconv.ParseBool(verifyProofsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid verify_proofs parameter: %w", err)
		}
		config.VerifyProofs = verifyProofs
	}

	return config, nil
}

// ValidateFormat validates that the encryption format is supported
func ValidateFormat(format EncryptionFormat) error {
	switch format {
	case FormatSaltpack, FormatPGP:
		return nil
	default:
		return fmt.Errorf("unsupported format '%s': must be 'saltpack' or 'pgp'", format)
	}
}

// String returns a string representation of the Config
func (c *Config) String() string {
	return fmt.Sprintf("Config{Recipients: %v, Format: %s, CacheTTL: %s, VerifyProofs: %t}",
		c.Recipients, c.Format, c.CacheTTL, c.VerifyProofs)
}

// ToURL converts a Config back to a URL string
func (c *Config) ToURL() string {
	recipients := strings.Join(c.Recipients, ",")
	
	u := &url.URL{
		Scheme: "keybase",
		Host:   recipients,
	}

	query := url.Values{}
	
	if c.Format != FormatSaltpack {
		query.Set("format", string(c.Format))
	}
	
	if c.CacheTTL != 24*time.Hour {
		query.Set("cache_ttl", strconv.FormatInt(int64(c.CacheTTL.Seconds()), 10))
	}
	
	if c.VerifyProofs {
		query.Set("verify_proofs", "true")
	}

	u.RawQuery = query.Encode()
	
	return u.String()
}
