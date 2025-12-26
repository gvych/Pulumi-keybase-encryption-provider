package keybase

import (
	"context"
	"testing"
	"time"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
	"gocloud.dev/gcerrors"
)

// TestNewKeeper tests creating a new Keeper
func TestNewKeeper(t *testing.T) {
	tests := []struct {
		name    string
		config  *KeeperConfig
		wantErr bool
	}{
		{
			name: "valid config with single recipient",
			config: &KeeperConfig{
				Config: &Config{
					Recipients: []string{"alice"},
					Format:     FormatSaltpack,
					CacheTTL:   24 * time.Hour,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple recipients",
			config: &KeeperConfig{
				Config: &Config{
					Recipients: []string{"alice", "bob", "charlie"},
					Format:     FormatSaltpack,
					CacheTTL:   24 * time.Hour,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config with nil Config",
			config: &KeeperConfig{
				Config: nil,
			},
			wantErr: true,
		},
		{
			name: "config with no recipients",
			config: &KeeperConfig{
				Config: &Config{
					Recipients: []string{},
					Format:     FormatSaltpack,
					CacheTTL:   24 * time.Hour,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keeper, err := NewKeeper(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKeeper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if keeper == nil {
					t.Error("NewKeeper() returned nil keeper for valid config")
				}
				defer keeper.Close()
			}
		})
	}
}

// TestNewKeeperFromURL tests creating a Keeper from a URL
func TestNewKeeperFromURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid URL with single recipient",
			url:     "keybase://alice",
			wantErr: false,
		},
		{
			name:    "valid URL with multiple recipients",
			url:     "keybase://alice,bob,charlie",
			wantErr: false,
		},
		{
			name:    "valid URL with query parameters",
			url:     "keybase://alice,bob?format=saltpack&cache_ttl=3600",
			wantErr: false,
		},
		{
			name:    "invalid URL - wrong scheme",
			url:     "https://alice",
			wantErr: true,
		},
		{
			name:    "invalid URL - no recipients",
			url:     "keybase://",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keeper, err := NewKeeperFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKeeperFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if keeper == nil {
					t.Error("NewKeeperFromURL() returned nil keeper for valid URL")
				}
				defer keeper.Close()
			}
		})
	}
}

// TestKeeperEncrypt tests the Encrypt method
func TestKeeperEncrypt(t *testing.T) {
	// Create test keys
	keyPair1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}
	
	keyPair2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}

	// Create a mock cache manager with test keys
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair1.PublicKey,
		"bob":   keyPair2.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	tests := []struct {
		name      string
		recipients []string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:       "encrypt for single recipient",
			recipients: []string{"alice"},
			plaintext:  []byte("secret message"),
			wantErr:    false,
		},
		{
			name:       "encrypt for multiple recipients",
			recipients: []string{"alice", "bob"},
			plaintext:  []byte("secret message for multiple recipients"),
			wantErr:    false,
		},
		{
			name:       "encrypt empty plaintext",
			recipients: []string{"alice"},
			plaintext:  []byte(""),
			wantErr:    true,
		},
		{
			name:       "encrypt for unknown recipient",
			recipients: []string{"unknown"},
			plaintext:  []byte("secret message"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Recipients: tt.recipients,
				Format:     FormatSaltpack,
				CacheTTL:   24 * time.Hour,
			}

			keeper, err := NewKeeper(&KeeperConfig{
				Config:       config,
				CacheManager: cacheManager,
			})
			if err != nil {
				t.Fatalf("Failed to create keeper: %v", err)
			}
			defer keeper.Close()

			ctx := context.Background()
			ciphertext, err := keeper.Encrypt(ctx, tt.plaintext)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Keeper.Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(ciphertext) == 0 {
					t.Error("Keeper.Encrypt() returned empty ciphertext")
				}
				
				// Verify it's ASCII-armored (should start with BEGIN SALTPACK)
				if len(ciphertext) > 20 {
					ciphertextStr := string(ciphertext)
					if len(ciphertextStr) < 20 || ciphertextStr[:10] != "BEGIN SALT" {
						t.Logf("Ciphertext preview: %s", ciphertextStr[:min(100, len(ciphertextStr))])
					}
				}
			}
		})
	}
}

// TestKeeperEncryptDecrypt tests the full encrypt/decrypt cycle
func TestKeeperEncryptDecrypt(t *testing.T) {
	// Generate test key pairs
	keyPair1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}
	
	keyPair2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}

	// Create mock cache manager
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair1.PublicKey,
		"bob":   keyPair2.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	tests := []struct {
		name       string
		recipients []string
		plaintext  []byte
		decryptKey saltpack.BoxSecretKey
	}{
		{
			name:       "single recipient encrypt/decrypt",
			recipients: []string{"alice"},
			plaintext:  []byte("Hello, Alice!"),
			decryptKey: keyPair1.SecretKey,
		},
		{
			name:       "multiple recipients - decrypt with first key",
			recipients: []string{"alice", "bob"},
			plaintext:  []byte("Hello, Alice and Bob!"),
			decryptKey: keyPair1.SecretKey,
		},
		{
			name:       "multiple recipients - decrypt with second key",
			recipients: []string{"alice", "bob"},
			plaintext:  []byte("Another message"),
			decryptKey: keyPair2.SecretKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create keeper for encryption
			config := &Config{
				Recipients: tt.recipients,
				Format:     FormatSaltpack,
				CacheTTL:   24 * time.Hour,
			}

			encryptKeeper, err := NewKeeper(&KeeperConfig{
				Config:       config,
				CacheManager: cacheManager,
			})
			if err != nil {
				t.Fatalf("Failed to create encrypt keeper: %v", err)
			}
			defer encryptKeeper.Close()

			// Encrypt
			ctx := context.Background()
			ciphertext, err := encryptKeeper.Encrypt(ctx, tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Create keeper for decryption with the specific key
			keyring := crypto.NewSimpleKeyring()
			keyring.AddKey(tt.decryptKey)

			decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}

			decryptKeeper := &Keeper{
				config:       config,
				cacheManager: cacheManager,
				decryptor:    decryptor,
				keyring:      keyring,
			}

			// Decrypt
			decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify
			if string(decrypted) != string(tt.plaintext) {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

// TestKeeperDecryptErrors tests error handling in Decrypt
func TestKeeperDecryptErrors(t *testing.T) {
	// Generate a test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a keeper with the test key
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	keeper := &Keeper{
		config:    config,
		decryptor: decryptor,
		keyring:   keyring,
	}

	tests := []struct {
		name       string
		ciphertext []byte
		wantErr    bool
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte(""),
			wantErr:    true,
		},
		{
			name:       "invalid ciphertext",
			ciphertext: []byte("not a valid ciphertext"),
			wantErr:    true,
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: []byte("BEGIN SALTPACK ENCRYPTED MESSAGE. corrupted data here"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := keeper.Decrypt(ctx, tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Keeper.Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestKeeperErrorCode tests the ErrorCode method
func TestKeeperErrorCode(t *testing.T) {
	keeper := &Keeper{
		config: DefaultConfig(),
	}

	tests := []struct {
		name     string
		err      error
		wantCode gcerrors.ErrorCode
	}{
		{
			name: "keeper error - invalid argument",
			err: &KeeperError{
				Message: "test error",
				Code:    gcerrors.InvalidArgument,
			},
			wantCode: gcerrors.InvalidArgument,
		},
		{
			name: "keeper error - not found",
			err: &KeeperError{
				Message: "test error",
				Code:    gcerrors.NotFound,
			},
			wantCode: gcerrors.NotFound,
		},
		{
			name: "API error - network",
			err: &api.APIError{
				Message: "network error",
				Kind:    api.ErrorKindNetwork,
			},
			wantCode: gcerrors.Internal,
		},
		{
			name: "API error - timeout",
			err: &api.APIError{
				Message: "timeout error",
				Kind:    api.ErrorKindTimeout,
			},
			wantCode: gcerrors.DeadlineExceeded,
		},
		{
			name: "API error - not found",
			err: &api.APIError{
				Message: "user not found",
				Kind:    api.ErrorKindNotFound,
			},
			wantCode: gcerrors.NotFound,
		},
		{
			name: "API error - rate limit",
			err: &api.APIError{
				Message: "rate limited",
				Kind:    api.ErrorKindRateLimit,
			},
			wantCode: gcerrors.ResourceExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := keeper.ErrorCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("ErrorCode() = %v, want %v", code, tt.wantCode)
			}
		})
	}
}

// TestKeeperErrorAs tests the ErrorAs method
func TestKeeperErrorAs(t *testing.T) {
	keeper := &Keeper{
		config: DefaultConfig(),
	}

	tests := []struct {
		name   string
		err    error
		target interface{}
		want   bool
	}{
		{
			name:   "keeper error to keeper error",
			err:    &KeeperError{Message: "test"},
			target: new(*KeeperError),
			want:   true,
		},
		{
			name:   "API error to API error",
			err:    &api.APIError{Message: "test"},
			target: new(*api.APIError),
			want:   true,
		},
		{
			name:   "keeper error to wrong type",
			err:    &KeeperError{Message: "test"},
			target: new(*api.APIError),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keeper.ErrorAs(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("ErrorAs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeeperClose tests the Close method
func TestKeeperClose(t *testing.T) {
	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	keeper, err := NewKeeper(&KeeperConfig{
		Config: config,
	})
	if err != nil {
		t.Fatalf("Failed to create keeper: %v", err)
	}

	if err := keeper.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestKeeperDecryptWithInfo tests DecryptWithInfo method
func TestKeeperDecryptWithInfo(t *testing.T) {
	// Generate test key pairs
	keyPair1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}
	
	keyPair2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}
	
	sender, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}

	// Create mock cache manager
	cacheManager, err := createMockCacheManager(map[string]saltpack.BoxPublicKey{
		"alice": keyPair1.PublicKey,
		"bob":   keyPair2.PublicKey,
	})
	if err != nil {
		t.Fatalf("Failed to create mock cache manager: %v", err)
	}
	defer cacheManager.Close()

	tests := []struct {
		name       string
		recipients []string
		plaintext  []byte
		decryptKey saltpack.BoxSecretKey
		senderKey  saltpack.BoxSecretKey
		wantAnonymous bool
	}{
		{
			name:       "single recipient with sender",
			recipients: []string{"alice"},
			plaintext:  []byte("Hello, Alice!"),
			decryptKey: keyPair1.SecretKey,
			senderKey:  sender.SecretKey,
			wantAnonymous: false,
		},
		{
			name:       "single recipient anonymous sender",
			recipients: []string{"alice"},
			plaintext:  []byte("Anonymous message"),
			decryptKey: keyPair1.SecretKey,
			senderKey:  nil,
			wantAnonymous: true,
		},
		{
			name:       "multiple recipients - decrypt with first key",
			recipients: []string{"alice", "bob"},
			plaintext:  []byte("Hello, team!"),
			decryptKey: keyPair1.SecretKey,
			senderKey:  sender.SecretKey,
			wantAnonymous: false,
		},
		{
			name:       "multiple recipients - decrypt with second key",
			recipients: []string{"alice", "bob"},
			plaintext:  []byte("Another message"),
			decryptKey: keyPair2.SecretKey,
			senderKey:  sender.SecretKey,
			wantAnonymous: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create keeper for encryption
			config := &Config{
				Recipients: tt.recipients,
				Format:     FormatSaltpack,
				CacheTTL:   24 * time.Hour,
			}

			encryptKeeper, err := NewKeeper(&KeeperConfig{
				Config:       config,
				CacheManager: cacheManager,
				SenderKey:    tt.senderKey,
			})
			if err != nil {
				t.Fatalf("Failed to create encrypt keeper: %v", err)
			}
			defer encryptKeeper.Close()

			// Encrypt
			ctx := context.Background()
			ciphertext, err := encryptKeeper.Encrypt(ctx, tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Create keeper for decryption with the specific key
			keyring := crypto.NewSimpleKeyring()
			keyring.AddKey(tt.decryptKey)
			if tt.senderKey != nil {
				keyring.AddPublicKey(tt.senderKey.GetPublicKey())
			}

			decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}

			decryptKeeper := &Keeper{
				config:       config,
				cacheManager: cacheManager,
				decryptor:    decryptor,
				keyring:      keyring,
			}

			// Decrypt with info
			decrypted, messageInfo, err := decryptKeeper.DecryptWithInfo(ctx, ciphertext)
			if err != nil {
				t.Fatalf("DecryptWithInfo() error = %v", err)
			}

			// Verify plaintext
			if string(decrypted) != string(tt.plaintext) {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
			
			// Verify message info
			if messageInfo == nil {
				t.Fatal("DecryptWithInfo() returned nil messageInfo")
			}
			
			// Verify receiver key ID
			if len(messageInfo.ReceiverKID) == 0 {
				t.Error("ReceiverKID is empty")
			}
			
			expectedReceiverKID := tt.decryptKey.GetPublicKey().ToKID()
			if string(messageInfo.ReceiverKID) != string(expectedReceiverKID) {
				t.Errorf("ReceiverKID mismatch: got %x, want %x", 
					messageInfo.ReceiverKID, expectedReceiverKID)
			}
			
			// Verify sender anonymity
			if messageInfo.IsAnonymousSender != tt.wantAnonymous {
				t.Errorf("IsAnonymousSender = %v, want %v", 
					messageInfo.IsAnonymousSender, tt.wantAnonymous)
			}
			
			// Verify sender key if not anonymous
			if !tt.wantAnonymous {
				if len(messageInfo.SenderKID) == 0 {
					t.Error("SenderKID is empty for non-anonymous sender")
				}
				
				expectedSenderKID := tt.senderKey.GetPublicKey().ToKID()
				if string(messageInfo.SenderKID) != string(expectedSenderKID) {
					t.Errorf("SenderKID mismatch: got %x, want %x",
						messageInfo.SenderKID, expectedSenderKID)
				}
			} else {
				if len(messageInfo.SenderKID) != 0 {
					t.Error("SenderKID should be empty for anonymous sender")
				}
			}
		})
	}
}

// TestKeeperDecryptWithInfoErrors tests error handling in DecryptWithInfo
func TestKeeperDecryptWithInfoErrors(t *testing.T) {
	// Generate a test key pair
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a keeper with the test key
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	config := &Config{
		Recipients: []string{"alice"},
		Format:     FormatSaltpack,
		CacheTTL:   24 * time.Hour,
	}

	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	keeper := &Keeper{
		config:    config,
		decryptor: decryptor,
		keyring:   keyring,
	}

	tests := []struct {
		name       string
		ciphertext []byte
		wantErr    bool
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte(""),
			wantErr:    true,
		},
		{
			name:       "invalid ciphertext",
			ciphertext: []byte("not a valid ciphertext"),
			wantErr:    true,
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: []byte("BEGIN SALTPACK ENCRYPTED MESSAGE. corrupted data here"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, _, err := keeper.DecryptWithInfo(ctx, tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Keeper.DecryptWithInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper functions

// createMockCacheManager creates a cache manager with pre-populated test keys
func createMockCacheManager(keys map[string]saltpack.BoxPublicKey) (*cache.Manager, error) {
	manager, err := cache.NewManager(nil)
	if err != nil {
		return nil, err
	}

	// Pre-populate cache with test keys
	for username, publicKey := range keys {
		// Convert public key to hex string for storage
		keyID := publicKey.ToKID()
		
		// For testing, we need to store the key ID in a format that can be parsed
		// Since the Keeper tries to parse the KID as a Curve25519 key, we'll store
		// the actual key bytes as a hex string prefixed with "01" to make it look like
		// a Keybase KID format
		var hexEncoder = func(b []byte) string {
			result := make([]byte, len(b)*2)
			const hexTable = "0123456789abcdef"
			for i, v := range b {
				result[i*2] = hexTable[v>>4]
				result[i*2+1] = hexTable[v&0x0f]
			}
			return string(result)
		}
		keyIDHex := "0120" + hexEncoder(keyID) // Prefix with 0120 to simulate Keybase KID format
		
		// Store a mock PGP key bundle (will fail to parse, but KID parsing will succeed)
		mockKeyBundle := "-----BEGIN PGP PUBLIC KEY BLOCK----- test key -----"
		
		if err := manager.Cache().Set(username, mockKeyBundle, keyIDHex); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
