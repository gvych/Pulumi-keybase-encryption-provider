package crypto

import (
	"strings"
	"testing"

	"github.com/keybase/saltpack"
)

// Test valid KID extraction
func TestExtractKeyFromKID_Valid(t *testing.T) {
	// Valid test KID (0120 + 64 hex chars)
	testKID := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	key, err := extractKeyFromKID(testKID)
	if err != nil {
		t.Fatalf("extractKeyFromKID() failed: %v", err)
	}
	
	// Verify the key bytes match
	expectedHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	actualHex := PublicKeyToHex(key)
	
	if actualHex != expectedHex {
		t.Errorf("Key mismatch: expected %s, got %s", expectedHex, actualHex)
	}
}

// Test KID with wrong length
func TestExtractKeyFromKID_InvalidLength(t *testing.T) {
	testCases := []struct {
		name string
		kid  string
	}{
		{"too short", "0120abc"},
		{"too long", "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345678extra"},
		{"empty", ""},
		{"only prefix", "0120"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractKeyFromKID(tc.kid)
			if err == nil {
				t.Errorf("Expected error for KID %q, got nil", tc.kid)
			}
		})
	}
}

// Test KID with wrong prefix
func TestExtractKeyFromKID_InvalidPrefix(t *testing.T) {
	// Wrong prefix (should be 0120, not 0121)
	testKID := "0121abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	_, err := extractKeyFromKID(testKID)
	if err == nil {
		t.Error("Expected error for wrong prefix, got nil")
	}
	
	if !strings.Contains(err.Error(), "invalid KID prefix") {
		t.Errorf("Expected 'invalid KID prefix' error, got: %v", err)
	}
}

// Test KID with invalid hex
func TestExtractKeyFromKID_InvalidHex(t *testing.T) {
	// Contains non-hex characters (g, h, z)
	testKID := "0120ghijkl0123456789abcdef0123456789abcdef0123456789abcdef0123456789z"
	
	_, err := extractKeyFromKID(testKID)
	if err == nil {
		t.Error("Expected error for invalid hex, got nil")
	}
}

// Test parseHexKey with valid input
func TestParseHexKey_Valid(t *testing.T) {
	testHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	key, err := parseHexKey(testHex)
	if err != nil {
		t.Fatalf("parseHexKey() failed: %v", err)
	}
	
	actualHex := PublicKeyToHex(key)
	if actualHex != testHex {
		t.Errorf("Key mismatch: expected %s, got %s", testHex, actualHex)
	}
}

// Test parseHexKey with invalid length
func TestParseHexKey_InvalidLength(t *testing.T) {
	testCases := []struct {
		name   string
		hexKey string
	}{
		{"too short", "abcdef"},
		{"too long", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567extra"},
		{"empty", ""},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseHexKey(tc.hexKey)
			if err == nil {
				t.Errorf("Expected error for hex key %q, got nil", tc.hexKey)
			}
		})
	}
}

// Test PGPToBoxPublicKey with valid KID
func TestPGPToBoxPublicKey_ValidKID(t *testing.T) {
	testKID := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	key, err := PGPToBoxPublicKey("", testKID)
	if err != nil {
		t.Fatalf("PGPToBoxPublicKey() failed: %v", err)
	}
	
	// Verify key is not all zeros
	err = ValidatePublicKey(key)
	if err != nil {
		t.Error("Key is all zeros")
	}
}

// Test PGPToBoxPublicKey with hex string
func TestPGPToBoxPublicKey_HexString(t *testing.T) {
	testHex := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	
	key, err := PGPToBoxPublicKey(testHex, "")
	if err != nil {
		t.Fatalf("PGPToBoxPublicKey() failed: %v", err)
	}
	
	actualHex := PublicKeyToHex(key)
	if actualHex != testHex {
		t.Errorf("Key mismatch: expected %s, got %s", testHex, actualHex)
	}
}

// Test PGPToBoxPublicKey with 32-byte binary
func TestPGPToBoxPublicKey_Binary(t *testing.T) {
	// Create a 32-byte test key
	testBytes := make([]byte, 32)
	for i := range testBytes {
		testBytes[i] = byte(i)
	}
	
	key, err := PGPToBoxPublicKey(string(testBytes), "")
	if err != nil {
		t.Fatalf("PGPToBoxPublicKey() failed: %v", err)
	}
	
	// Verify the key matches by converting back to bytes
	keyBytes := key.ToKID()
	for i := range keyBytes {
		if keyBytes[i] != testBytes[i] {
			t.Errorf("Key byte %d mismatch: expected %d, got %d", i, testBytes[i], keyBytes[i])
		}
	}
}

// Test PGPToBoxPublicKey with PGP bundle (should fail with helpful message)
func TestPGPToBoxPublicKey_PGPBundle(t *testing.T) {
	pgpBundle := `-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQENBFsomedata...
-----END PGP PUBLIC KEY BLOCK-----`
	
	_, err := PGPToBoxPublicKey(pgpBundle, "")
	if err == nil {
		t.Error("Expected error for PGP bundle, got nil")
	}
	
	// Should mention using KID instead
	if !strings.Contains(err.Error(), "KID") {
		t.Errorf("Error should mention KID, got: %v", err)
	}
}

// Test PGPToBoxPublicKey with unsupported format
func TestPGPToBoxPublicKey_UnsupportedFormat(t *testing.T) {
	testCases := []struct {
		name    string
		keyData string
		kid     string
	}{
		{"random text", "this is not a key", ""},
		{"wrong length binary", "too short", ""},
		{"empty both", "", ""},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PGPToBoxPublicKey(tc.keyData, tc.kid)
			if err == nil {
				t.Errorf("Expected error for unsupported format, got nil")
			}
		})
	}
}

// Test ValidatePublicKey with valid key
func TestValidatePublicKey_Valid(t *testing.T) {
	// Create a non-zero key
	var rawKey saltpack.RawBoxKey
	for i := range rawKey {
		rawKey[i] = byte(i + 1)
	}
	key := NewBoxPublicKey(rawKey)
	
	err := ValidatePublicKey(key)
	if err != nil {
		t.Errorf("ValidatePublicKey() failed for valid key: %v", err)
	}
}

// Test ValidatePublicKey with all-zero key
func TestValidatePublicKey_AllZeros(t *testing.T) {
	var rawKey saltpack.RawBoxKey // All zeros by default
	key := NewBoxPublicKey(rawKey)
	
	err := ValidatePublicKey(key)
	if err == nil {
		t.Error("Expected error for all-zero key, got nil")
	}
	
	if !strings.Contains(err.Error(), "all zeros") {
		t.Errorf("Expected 'all zeros' error, got: %v", err)
	}
}

// Test FormatKeyID
func TestFormatKeyID(t *testing.T) {
	// Create a test key
	testHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	key, _ := parseHexKey(testHex)
	
	kid := FormatKeyID(key)
	
	expectedKID := "0120" + testHex
	if kid != expectedKID {
		t.Errorf("FormatKeyID() = %s, want %s", kid, expectedKID)
	}
}

// Test PublicKeyToHex
func TestPublicKeyToHex(t *testing.T) {
	testHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	key, _ := parseHexKey(testHex)
	
	actualHex := PublicKeyToHex(key)
	
	if actualHex != testHex {
		t.Errorf("PublicKeyToHex() = %s, want %s", actualHex, testHex)
	}
}

// Test ConvertPublicKeys with valid input
func TestConvertPublicKeys_Valid(t *testing.T) {
	users := []UserPublicKey{
		{
			Username:  "alice",
			KeyID:     "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			PublicKey: "",
		},
		{
			Username:  "bob",
			KeyID:     "01201234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			PublicKey: "",
		},
	}
	
	results, err := ConvertPublicKeys(users)
	if err != nil {
		t.Fatalf("ConvertPublicKeys() failed: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	// Verify usernames
	if results[0].Username != "alice" {
		t.Errorf("Expected username 'alice', got %s", results[0].Username)
	}
	if results[1].Username != "bob" {
		t.Errorf("Expected username 'bob', got %s", results[1].Username)
	}
}

// Test ConvertPublicKeys with empty input
func TestConvertPublicKeys_Empty(t *testing.T) {
	_, err := ConvertPublicKeys([]UserPublicKey{})
	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

// Test ConvertPublicKeys with invalid key
func TestConvertPublicKeys_InvalidKey(t *testing.T) {
	users := []UserPublicKey{
		{
			Username:  "alice",
			KeyID:     "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			PublicKey: "",
		},
		{
			Username:  "bob",
			KeyID:     "invalid-kid",
			PublicKey: "",
		},
	}
	
	results, err := ConvertPublicKeys(users)
	
	// Should get an error
	if err == nil {
		t.Error("Expected error for invalid key, got nil")
	}
	
	// Should still return valid results for alice
	if len(results) != 1 {
		t.Errorf("Expected 1 valid result, got %d", len(results))
	}
	
	if len(results) > 0 && results[0].Username != "alice" {
		t.Errorf("Expected valid result for 'alice', got %s", results[0].Username)
	}
}

// Test ConvertPublicKeys with all-zero key
func TestConvertPublicKeys_AllZeroKey(t *testing.T) {
	// All zeros = invalid key
	users := []UserPublicKey{
		{
			Username:  "alice",
			KeyID:     "01200000000000000000000000000000000000000000000000000000000000000000",
			PublicKey: "",
		},
	}
	
	results, err := ConvertPublicKeys(users)
	
	// Should get an error
	if err == nil {
		t.Error("Expected error for all-zero key, got nil")
	}
	
	// Should have no valid results
	if len(results) != 0 {
		t.Errorf("Expected 0 valid results, got %d", len(results))
	}
}

// Test round-trip: KID -> Key -> KID
func TestRoundTrip_KIDToKeyToKID(t *testing.T) {
	originalKID := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	// KID -> Key
	key, err := extractKeyFromKID(originalKID)
	if err != nil {
		t.Fatalf("extractKeyFromKID() failed: %v", err)
	}
	
	// Key -> KID
	reconstructedKID := FormatKeyID(key)
	
	if reconstructedKID != originalKID {
		t.Errorf("Round-trip failed: original %s, reconstructed %s", originalKID, reconstructedKID)
	}
}

// Test KID with whitespace (should be trimmed)
func TestExtractKeyFromKID_Whitespace(t *testing.T) {
	testKID := "  0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  "
	
	key, err := extractKeyFromKID(testKID)
	if err != nil {
		t.Fatalf("extractKeyFromKID() failed with whitespace: %v", err)
	}
	
	// Verify the key is valid
	err = ValidatePublicKey(key)
	if err != nil {
		t.Errorf("Key validation failed: %v", err)
	}
}

// Test hex key with whitespace (should be trimmed)
func TestParseHexKey_Whitespace(t *testing.T) {
	testHex := "  abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  "
	
	key, err := parseHexKey(testHex)
	if err != nil {
		t.Fatalf("parseHexKey() failed with whitespace: %v", err)
	}
	
	expectedHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	actualHex := PublicKeyToHex(key)
	
	if actualHex != expectedHex {
		t.Errorf("Key mismatch: expected %s, got %s", expectedHex, actualHex)
	}
}

// Test PGPToBoxPublicKey prioritizes KID over keyData
func TestPGPToBoxPublicKey_KIDPriority(t *testing.T) {
	validKID := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	invalidKeyData := "this is invalid"
	
	// Should succeed because KID is tried first
	key, err := PGPToBoxPublicKey(invalidKeyData, validKID)
	if err != nil {
		t.Fatalf("PGPToBoxPublicKey() should prioritize KID: %v", err)
	}
	
	// Verify the key comes from the KID
	expectedHex := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	actualHex := PublicKeyToHex(key)
	
	if actualHex != expectedHex {
		t.Errorf("Key should come from KID: expected %s, got %s", expectedHex, actualHex)
	}
}

// Benchmark key conversion
func BenchmarkExtractKeyFromKID(b *testing.B) {
	testKID := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractKeyFromKID(testKID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPGPToBoxPublicKey(b *testing.B) {
	testKID := "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := PGPToBoxPublicKey("", testKID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertPublicKeys(b *testing.B) {
	users := []UserPublicKey{
		{
			Username: "alice",
			KeyID:    "0120abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			Username: "bob",
			KeyID:    "01201234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			Username: "charlie",
			KeyID:    "0120fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ConvertPublicKeys(users)
		if err != nil {
			b.Fatal(err)
		}
	}
}
