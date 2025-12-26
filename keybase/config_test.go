package keybase

import (
	"testing"
	"time"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantConfig  *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "basic single recipient",
			url:  "keybase://alice",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "multiple recipients",
			url:  "keybase://alice,bob,charlie",
			wantConfig: &Config{
				Recipients:   []string{"alice", "bob", "charlie"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "with saltpack format",
			url:  "keybase://alice,bob?format=saltpack",
			wantConfig: &Config{
				Recipients:   []string{"alice", "bob"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "with pgp format",
			url:  "keybase://alice?format=pgp",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatPGP,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "with custom cache TTL",
			url:  "keybase://alice?cache_ttl=3600",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     1 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "with verify_proofs enabled",
			url:  "keybase://alice?verify_proofs=true",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: true,
			},
			wantErr: false,
		},
		{
			name: "all parameters specified",
			url:  "keybase://alice,bob,charlie?format=pgp&cache_ttl=7200&verify_proofs=true",
			wantConfig: &Config{
				Recipients:   []string{"alice", "bob", "charlie"},
				Format:       FormatPGP,
				CacheTTL:     2 * time.Hour,
				VerifyProofs: true,
			},
			wantErr: false,
		},
		{
			name: "recipients with underscores",
			url:  "keybase://alice_test,bob_123",
			wantConfig: &Config{
				Recipients:   []string{"alice_test", "bob_123"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "trailing comma (empty recipients filtered)",
			url:  "keybase://alice,bob,",
			wantConfig: &Config{
				Recipients:   []string{"alice", "bob"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name:        "empty URL",
			url:         "",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "URL cannot be empty",
		},
		{
			name:        "invalid scheme",
			url:         "https://alice,bob",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "invalid URL scheme",
		},
		{
			name:        "no recipients",
			url:         "keybase://",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "no recipients specified",
		},
		{
			name:        "invalid username with special chars",
			url:         "keybase://alice@example.com",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "invalid recipient username",
		},
		{
			name:        "empty recipients list",
			url:         "keybase://,,,",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "no valid recipients",
		},
		{
			name:        "invalid username with hyphen",
			url:         "keybase://alice-bob",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "invalid recipient username",
		},
		{
			name:        "invalid format",
			url:         "keybase://alice?format=aes",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "invalid format parameter",
		},
		{
			name:        "invalid cache_ttl (non-numeric)",
			url:         "keybase://alice?cache_ttl=invalid",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "invalid cache_ttl parameter",
		},
		{
			name:        "invalid cache_ttl (negative)",
			url:         "keybase://alice?cache_ttl=-100",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "cache_ttl must be non-negative",
		},
		{
			name:        "invalid verify_proofs",
			url:         "keybase://alice?verify_proofs=invalid",
			wantConfig:  nil,
			wantErr:     true,
			errContains: "invalid verify_proofs parameter",
		},
		{
			name: "zero cache_ttl (valid)",
			url:  "keybase://alice?cache_ttl=0",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     0,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "verify_proofs false",
			url:  "keybase://alice?verify_proofs=false",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "case insensitive format",
			url:  "keybase://alice?format=SALTPACK",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
		{
			name: "case insensitive format PGP",
			url:  "keybase://alice?format=PGP",
			wantConfig: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatPGP,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseURL() expected error containing '%s', got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ParseURL() error = %v, want error containing '%s'", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseURL() unexpected error = %v", err)
				return
			}

			if config == nil {
				t.Errorf("ParseURL() returned nil config")
				return
			}

			// Compare Recipients
			if len(config.Recipients) != len(tt.wantConfig.Recipients) {
				t.Errorf("ParseURL() Recipients length = %d, want %d", len(config.Recipients), len(tt.wantConfig.Recipients))
			} else {
				for i, recipient := range config.Recipients {
					if recipient != tt.wantConfig.Recipients[i] {
						t.Errorf("ParseURL() Recipients[%d] = %s, want %s", i, recipient, tt.wantConfig.Recipients[i])
					}
				}
			}

			// Compare Format
			if config.Format != tt.wantConfig.Format {
				t.Errorf("ParseURL() Format = %s, want %s", config.Format, tt.wantConfig.Format)
			}

			// Compare CacheTTL
			if config.CacheTTL != tt.wantConfig.CacheTTL {
				t.Errorf("ParseURL() CacheTTL = %s, want %s", config.CacheTTL, tt.wantConfig.CacheTTL)
			}

			// Compare VerifyProofs
			if config.VerifyProofs != tt.wantConfig.VerifyProofs {
				t.Errorf("ParseURL() VerifyProofs = %t, want %t", config.VerifyProofs, tt.wantConfig.VerifyProofs)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  EncryptionFormat
		wantErr bool
	}{
		{
			name:    "valid saltpack",
			format:  FormatSaltpack,
			wantErr: false,
		},
		{
			name:    "valid pgp",
			format:  FormatPGP,
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  EncryptionFormat("aes"),
			wantErr: true,
		},
		{
			name:    "empty format",
			format:  EncryptionFormat(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if len(config.Recipients) != 0 {
		t.Errorf("DefaultConfig() Recipients = %v, want empty slice", config.Recipients)
	}

	if config.Format != FormatSaltpack {
		t.Errorf("DefaultConfig() Format = %s, want %s", config.Format, FormatSaltpack)
	}

	if config.CacheTTL != 24*time.Hour {
		t.Errorf("DefaultConfig() CacheTTL = %s, want %s", config.CacheTTL, 24*time.Hour)
	}

	if config.VerifyProofs {
		t.Errorf("DefaultConfig() VerifyProofs = true, want false")
	}
}

func TestConfigString(t *testing.T) {
	config := &Config{
		Recipients:   []string{"alice", "bob"},
		Format:       FormatSaltpack,
		CacheTTL:     24 * time.Hour,
		VerifyProofs: false,
	}

	str := config.String()
	if str == "" {
		t.Error("Config.String() returned empty string")
	}

	// Check that string contains key information
	if !contains(str, "alice") {
		t.Errorf("Config.String() = %s, want string containing 'alice'", str)
	}
	if !contains(str, "bob") {
		t.Errorf("Config.String() = %s, want string containing 'bob'", str)
	}
	if !contains(str, "saltpack") {
		t.Errorf("Config.String() = %s, want string containing 'saltpack'", str)
	}
}

func TestConfigToURL(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		wantURL    string
		checkParse bool // Whether to parse the URL back and compare
	}{
		{
			name: "basic config",
			config: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantURL:    "keybase://alice",
			checkParse: true,
		},
		{
			name: "multiple recipients",
			config: &Config{
				Recipients:   []string{"alice", "bob", "charlie"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantURL:    "keybase://alice,bob,charlie",
			checkParse: true,
		},
		{
			name: "with PGP format",
			config: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatPGP,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: false,
			},
			wantURL:    "keybase://alice?format=pgp",
			checkParse: true,
		},
		{
			name: "with custom cache TTL",
			config: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     1 * time.Hour,
				VerifyProofs: false,
			},
			wantURL:    "keybase://alice?cache_ttl=3600",
			checkParse: true,
		},
		{
			name: "with verify proofs",
			config: &Config{
				Recipients:   []string{"alice"},
				Format:       FormatSaltpack,
				CacheTTL:     24 * time.Hour,
				VerifyProofs: true,
			},
			wantURL:    "keybase://alice?verify_proofs=true",
			checkParse: true,
		},
		{
			name: "all parameters",
			config: &Config{
				Recipients:   []string{"alice", "bob"},
				Format:       FormatPGP,
				CacheTTL:     2 * time.Hour,
				VerifyProofs: true,
			},
			checkParse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.config.ToURL()

			if url == "" {
				t.Error("Config.ToURL() returned empty string")
			}

			if !hasPrefix(url, "keybase://") {
				t.Errorf("Config.ToURL() = %s, want URL starting with 'keybase://'", url)
			}

			// If checkParse is true, parse the URL back and compare configs
			if tt.checkParse {
				parsedConfig, err := ParseURL(url)
				if err != nil {
					t.Errorf("ParseURL() failed to parse generated URL: %v", err)
					return
				}

				// Compare Recipients
				if len(parsedConfig.Recipients) != len(tt.config.Recipients) {
					t.Errorf("Recipients length mismatch: got %d, want %d", len(parsedConfig.Recipients), len(tt.config.Recipients))
				} else {
					for i, recipient := range parsedConfig.Recipients {
						if recipient != tt.config.Recipients[i] {
							t.Errorf("Recipients[%d] = %s, want %s", i, recipient, tt.config.Recipients[i])
						}
					}
				}

				// Compare Format
				if parsedConfig.Format != tt.config.Format {
					t.Errorf("Format = %s, want %s", parsedConfig.Format, tt.config.Format)
				}

				// Compare CacheTTL
				if parsedConfig.CacheTTL != tt.config.CacheTTL {
					t.Errorf("CacheTTL = %s, want %s", parsedConfig.CacheTTL, tt.config.CacheTTL)
				}

				// Compare VerifyProofs
				if parsedConfig.VerifyProofs != tt.config.VerifyProofs {
					t.Errorf("VerifyProofs = %t, want %t", parsedConfig.VerifyProofs, tt.config.VerifyProofs)
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that ParseURL(config.ToURL()) produces the same config
	originalConfigs := []*Config{
		{
			Recipients:   []string{"alice"},
			Format:       FormatSaltpack,
			CacheTTL:     24 * time.Hour,
			VerifyProofs: false,
		},
		{
			Recipients:   []string{"alice", "bob", "charlie"},
			Format:       FormatPGP,
			CacheTTL:     12 * time.Hour,
			VerifyProofs: true,
		},
		{
			Recipients:   []string{"test_user", "another_user123"},
			Format:       FormatSaltpack,
			CacheTTL:     1 * time.Hour,
			VerifyProofs: false,
		},
	}

	for i, original := range originalConfigs {
		t.Run("round_trip_"+string(rune('0'+i)), func(t *testing.T) {
			url := original.ToURL()
			parsed, err := ParseURL(url)
			if err != nil {
				t.Errorf("ParseURL() error = %v", err)
				return
			}

			// Compare all fields
			if len(parsed.Recipients) != len(original.Recipients) {
				t.Errorf("Recipients length = %d, want %d", len(parsed.Recipients), len(original.Recipients))
			} else {
				for i, recipient := range parsed.Recipients {
					if recipient != original.Recipients[i] {
						t.Errorf("Recipients[%d] = %s, want %s", i, recipient, original.Recipients[i])
					}
				}
			}

			if parsed.Format != original.Format {
				t.Errorf("Format = %s, want %s", parsed.Format, original.Format)
			}

			if parsed.CacheTTL != original.CacheTTL {
				t.Errorf("CacheTTL = %s, want %s", parsed.CacheTTL, original.CacheTTL)
			}

			if parsed.VerifyProofs != original.VerifyProofs {
				t.Errorf("VerifyProofs = %t, want %t", parsed.VerifyProofs, original.VerifyProofs)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
