package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// TestGenerateKeyPair tests key pair generation
func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	
	if kp == nil {
		t.Fatal("GenerateKeyPair() returned nil")
	}
	
	if kp.PublicKey == nil {
		t.Error("GenerateKeyPair() returned nil PublicKey")
	}
	
	if kp.SecretKey == nil {
		t.Error("GenerateKeyPair() returned nil SecretKey")
	}
	
	if len(kp.Identifier) != 32 {
		t.Errorf("GenerateKeyPair() identifier length = %d, want 32", len(kp.Identifier))
	}
	
	// Verify public key from secret key matches
	derivedPublicKey := kp.SecretKey.GetPublicKey()
	if !KeysEqual(derivedPublicKey, kp.PublicKey) {
		t.Error("Public key derived from secret key doesn't match stored public key")
	}
}

// TestCreatePublicKey tests creating a public key from bytes
func TestCreatePublicKey(t *testing.T) {
	tests := []struct {
		name    string
		keyBytes []byte
		wantErr bool
	}{
		{
			name:    "valid 32 bytes",
			keyBytes: make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "too short",
			keyBytes: make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "too long",
			keyBytes: make([]byte, 64),
			wantErr: true,
		},
		{
			name:    "empty",
			keyBytes: []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill with test data
			for i := range tt.keyBytes {
				tt.keyBytes[i] = byte(i)
			}
			
			key, err := CreatePublicKey(tt.keyBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePublicKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if key == nil {
					t.Error("CreatePublicKey() returned nil key")
					return
				}
				
				kid := key.ToKID()
				if !bytes.Equal(kid, tt.keyBytes) {
					t.Error("CreatePublicKey() key ID doesn't match input bytes")
				}
			}
		})
	}
}

// TestCreatePublicKeyFromHex tests creating a public key from hex string
func TestCreatePublicKeyFromHex(t *testing.T) {
	// Generate a valid 32-byte key
	validKeyBytes := make([]byte, 32)
	for i := range validKeyBytes {
		validKeyBytes[i] = byte(i)
	}
	validHex := hex.EncodeToString(validKeyBytes)
	
	tests := []struct {
		name    string
		hexKey  string
		wantErr bool
	}{
		{
			name:    "valid hex",
			hexKey:  validHex,
			wantErr: false,
		},
		{
			name:    "invalid hex characters",
			hexKey:  "zzzzzz",
			wantErr: true,
		},
		{
			name:    "wrong length",
			hexKey:  "0123456789",
			wantErr: true,
		},
		{
			name:    "empty",
			hexKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := CreatePublicKeyFromHex(tt.hexKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePublicKeyFromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && key == nil {
				t.Error("CreatePublicKeyFromHex() returned nil key")
			}
		})
	}
}

// TestCreateSecretKey tests creating a secret key from bytes
func TestCreateSecretKey(t *testing.T) {
	tests := []struct {
		name    string
		keyBytes []byte
		wantErr bool
	}{
		{
			name:    "valid 32 bytes",
			keyBytes: make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "too short",
			keyBytes: make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "too long",
			keyBytes: make([]byte, 64),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill with test data
			for i := range tt.keyBytes {
				tt.keyBytes[i] = byte(i + 1) // Avoid all zeros
			}
			
			key, err := CreateSecretKey(tt.keyBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecretKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if key == nil {
					t.Error("CreateSecretKey() returned nil key")
					return
				}
				
				// Verify we can get public key
				pubKey := key.GetPublicKey()
				if pubKey == nil {
					t.Error("CreateSecretKey() GetPublicKey() returned nil")
				}
			}
		})
	}
}

// TestCreateSecretKeyFromHex tests creating a secret key from hex string
func TestCreateSecretKeyFromHex(t *testing.T) {
	// Generate a valid 32-byte key
	validKeyBytes := make([]byte, 32)
	for i := range validKeyBytes {
		validKeyBytes[i] = byte(i + 1)
	}
	validHex := hex.EncodeToString(validKeyBytes)
	
	tests := []struct {
		name    string
		hexKey  string
		wantErr bool
	}{
		{
			name:    "valid hex",
			hexKey:  validHex,
			wantErr: false,
		},
		{
			name:    "invalid hex",
			hexKey:  "invalid",
			wantErr: true,
		},
		{
			name:    "wrong length",
			hexKey:  "0123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := CreateSecretKeyFromHex(tt.hexKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecretKeyFromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && key == nil {
				t.Error("CreateSecretKeyFromHex() returned nil key")
			}
		})
	}
}

// TestSimpleKeyring tests the SimpleKeyring implementation
func TestSimpleKeyring(t *testing.T) {
	keyring := NewSimpleKeyring()
	if keyring == nil {
		t.Fatal("NewSimpleKeyring() returned nil")
	}
	
	// Generate test key pairs
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}
	
	kp2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}
	
	// Add keys
	keyring.AddKey(kp1.SecretKey)
	keyring.AddKeyPair(kp2)
	
	// Lookup by key ID
	kid1 := kp1.PublicKey.ToKID()
	kid2 := kp2.PublicKey.ToKID()
	
	t.Run("lookup existing key", func(t *testing.T) {
		index, key := keyring.LookupBoxSecretKey([][]byte{kid1})
		if index != 0 {
			t.Errorf("LookupBoxSecretKey() index = %d, want 0", index)
		}
		if key == nil {
			t.Error("LookupBoxSecretKey() returned nil key")
		}
	})
	
	t.Run("lookup multiple keys", func(t *testing.T) {
		index, key := keyring.LookupBoxSecretKey([][]byte{kid1, kid2})
		if index < 0 || index > 1 {
			t.Errorf("LookupBoxSecretKey() index = %d, want 0 or 1", index)
		}
		if key == nil {
			t.Error("LookupBoxSecretKey() returned nil key")
		}
	})
	
	t.Run("lookup non-existent key", func(t *testing.T) {
		fakeKID := make([]byte, 32)
		index, key := keyring.LookupBoxSecretKey([][]byte{fakeKID})
		if index != -1 {
			t.Errorf("LookupBoxSecretKey() index = %d, want -1", index)
		}
		if key != nil {
			t.Error("LookupBoxSecretKey() should return nil for non-existent key")
		}
	})
	
	t.Run("lookup public key", func(t *testing.T) {
		pubKey := keyring.LookupBoxPublicKey(kid1)
		if pubKey == nil {
			t.Error("LookupBoxPublicKey() returned nil")
		}
		if !KeysEqual(pubKey, kp1.PublicKey) {
			t.Error("LookupBoxPublicKey() returned wrong key")
		}
	})
	
	t.Run("get all secret keys", func(t *testing.T) {
		secKeys := keyring.GetAllBoxSecretKeys()
		if len(secKeys) != 2 {
			t.Errorf("GetAllBoxSecretKeys() returned %d keys, want 2", len(secKeys))
		}
	})
	
	t.Run("import secret key", func(t *testing.T) {
		rawKey := make([]byte, 32)
		for i := range rawKey {
			rawKey[i] = byte(i + 1)
		}
		
		importedKey := keyring.ImportBoxSecretKey(rawKey)
		if importedKey == nil {
			t.Error("ImportBoxSecretKey() returned nil")
		}
	})
	
	t.Run("import invalid key", func(t *testing.T) {
		invalidKey := make([]byte, 16) // Too short
		importedKey := keyring.ImportBoxSecretKey(invalidKey)
		if importedKey != nil {
			t.Error("ImportBoxSecretKey() should return nil for invalid key")
		}
	})
}

// TestValidatePublicKey tests public key validation
func TestValidatePublicKey(t *testing.T) {
	validKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	tests := []struct {
		name    string
		key     BoxPublicKey
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     validKey.PublicKey,
			wantErr: false,
		},
		{
			name:    "nil key",
			key:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	
	t.Run("all zeros key", func(t *testing.T) {
		zeroKey, _ := CreatePublicKey(make([]byte, 32))
		err := ValidatePublicKey(zeroKey)
		if err == nil {
			t.Error("ValidatePublicKey() should reject all-zero key")
		}
	})
}

// TestValidateSecretKey tests secret key validation
func TestValidateSecretKey(t *testing.T) {
	validKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	tests := []struct {
		name    string
		key     BoxSecretKey
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     validKey.SecretKey,
			wantErr: false,
		},
		{
			name:    "nil key",
			key:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecretKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSecretKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	
	t.Run("all zeros key", func(t *testing.T) {
		zeroKey, _ := CreateSecretKey(make([]byte, 32))
		err := ValidateSecretKey(zeroKey)
		if err == nil {
			t.Error("ValidateSecretKey() should reject all-zero key")
		}
	})
}

// TestKeysEqual tests key equality checking
func TestKeysEqual(t *testing.T) {
	key1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}
	
	key2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}
	
	tests := []struct {
		name  string
		key1  BoxPublicKey
		key2  BoxPublicKey
		want  bool
	}{
		{
			name: "same key",
			key1: key1.PublicKey,
			key2: key1.PublicKey,
			want: true,
		},
		{
			name: "different keys",
			key1: key1.PublicKey,
			key2: key2.PublicKey,
			want: false,
		},
		{
			name: "both nil",
			key1: nil,
			key2: nil,
			want: true,
		},
		{
			name: "first nil",
			key1: nil,
			key2: key1.PublicKey,
			want: false,
		},
		{
			name: "second nil",
			key1: key1.PublicKey,
			key2: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeysEqual(tt.key1, tt.key2); got != tt.want {
				t.Errorf("KeysEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseKeybasePublicKey tests parsing Keybase public keys
func TestParseKeybasePublicKey(t *testing.T) {
	tests := []struct {
		name      string
		keyBundle string
		wantErr   bool
	}{
		{
			name:      "PGP key bundle",
			keyBundle: "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest\n-----END PGP PUBLIC KEY BLOCK-----",
			wantErr:   true, // PGP conversion not yet implemented
		},
		{
			name:      "raw hex key (valid length)",
			keyBundle: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr:   false,
		},
		{
			name:      "invalid format",
			keyBundle: "not a valid key",
			wantErr:   true,
		},
		{
			name:      "empty",
			keyBundle: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseKeybasePublicKey(tt.keyBundle)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeybasePublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseKeybaseKeyID tests parsing Keybase key IDs
func TestParseKeybaseKeyID(t *testing.T) {
	tests := []struct {
		name    string
		kid     string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid hex with prefix",
			kid:     "0x0123456789abcdef",
			wantErr: false,
			wantLen: 8,
		},
		{
			name:    "valid hex without prefix",
			kid:     "0123456789abcdef",
			wantErr: false,
			wantLen: 8,
		},
		{
			name:    "invalid hex",
			kid:     "zzzz",
			wantErr: true,
		},
		{
			name:    "empty",
			kid:     "",
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyID, err := ParseKeybaseKeyID(tt.kid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeybaseKeyID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(keyID) != tt.wantLen {
				t.Errorf("ParseKeybaseKeyID() returned key ID of length %d, want %d", 
					len(keyID), tt.wantLen)
			}
		})
	}
}

// TestPrecompute tests key precomputation
func TestPrecompute(t *testing.T) {
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}
	
	kp2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}
	
	// Precompute shared key
	sharedKey := kp1.SecretKey.Precompute(kp2.PublicKey)
	if sharedKey == nil {
		t.Fatal("Precompute() returned nil")
	}
	
	// Test encryption with precomputed key
	var nonce [24]byte
	message := []byte("test message")
	
	encrypted := sharedKey.Box(nonce, message)
	if len(encrypted) == 0 {
		t.Error("Box() returned empty ciphertext")
	}
	
	// Decrypt with the other side's precomputed key
	sharedKey2 := kp2.SecretKey.Precompute(kp1.PublicKey)
	decrypted, err := sharedKey2.Unbox(nonce, encrypted)
	if err != nil {
		t.Errorf("Unbox() error = %v", err)
	}
	
	if !bytes.Equal(decrypted, message) {
		t.Errorf("Decrypted message doesn't match.\nGot:  %s\nWant: %s", 
			string(decrypted), string(message))
	}
}

// TestCreateEphemeralKey tests ephemeral key creation
func TestCreateEphemeralKey(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	
	ephemeralKey, err := kp.PublicKey.CreateEphemeralKey()
	if err != nil {
		t.Fatalf("CreateEphemeralKey() error = %v", err)
	}
	
	if ephemeralKey == nil {
		t.Fatal("CreateEphemeralKey() returned nil")
	}
	
	// Verify we can get public key from ephemeral key
	ephemeralPubKey := ephemeralKey.GetPublicKey()
	if ephemeralPubKey == nil {
		t.Error("Ephemeral key has no public key")
	}
	
	// Ephemeral key should be different from original
	if KeysEqual(ephemeralPubKey, kp.PublicKey) {
		t.Error("Ephemeral key should be different from original key")
	}
}

// TestNaclBoxPublicKeyMethods tests all methods of naclBoxPublicKey
func TestNaclBoxPublicKeyMethods(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	
	pubKey := kp.PublicKey
	
	t.Run("ToKID", func(t *testing.T) {
		kid := pubKey.ToKID()
		if len(kid) != 32 {
			t.Errorf("ToKID() returned %d bytes, want 32", len(kid))
		}
	})
	
	t.Run("ToRawBoxKeyPointer", func(t *testing.T) {
		rawKey := pubKey.ToRawBoxKeyPointer()
		if rawKey == nil {
			t.Error("ToRawBoxKeyPointer() returned nil")
		}
	})
	
	t.Run("HideIdentity", func(t *testing.T) {
		if pubKey.HideIdentity() {
			t.Error("HideIdentity() should return false")
		}
	})
}

// TestNaclBoxSecretKeyMethods tests all methods of naclBoxSecretKey
func TestNaclBoxSecretKeyMethods(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	
	secretKey := kp.SecretKey
	
	t.Run("GetPublicKey", func(t *testing.T) {
		pubKey := secretKey.GetPublicKey()
		if pubKey == nil {
			t.Error("GetPublicKey() returned nil")
		}
		if !KeysEqual(pubKey, kp.PublicKey) {
			t.Error("GetPublicKey() returned different key")
		}
	})
}

// BenchmarkGenerateKeyPair benchmarks key pair generation
func BenchmarkGenerateKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateKeyPair()
	}
}

// BenchmarkPrecompute benchmarks key precomputation
func BenchmarkPrecompute(b *testing.B) {
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kp1.SecretKey.Precompute(kp2.PublicKey)
	}
}
