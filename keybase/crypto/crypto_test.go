package crypto

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/keybase/saltpack"
)

// TestNewEncryptor tests creating a new encryptor
func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name    string
		config  *EncryptorConfig
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "empty config uses defaults",
			config:  &EncryptorConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncryptor(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && enc == nil {
				t.Error("NewEncryptor() returned nil encryptor")
			}
			// Skip version check since we can't reliably check if it's nil
		})
	}
}

// TestNewDecryptor tests creating a new decryptor
func TestNewDecryptor(t *testing.T) {
	keyring := NewSimpleKeyring()
	
	tests := []struct {
		name    string
		config  *DecryptorConfig
		wantErr bool
	}{
		{
			name:    "nil config returns error",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "nil keyring returns error",
			config:  &DecryptorConfig{Keyring: nil},
			wantErr: true,
		},
		{
			name:    "valid keyring succeeds",
			config:  &DecryptorConfig{Keyring: keyring},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := NewDecryptor(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDecryptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && dec == nil {
				t.Error("NewDecryptor() returned nil decryptor")
			}
		})
	}
}

// TestEncryptDecrypt tests basic encryption and decryption
func TestEncryptDecrypt(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient1 key: %v", err)
	}
	
	recipient2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient2 key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Create keyring with recipient keys and sender public key
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient1)
	keyring.AddKeyPair(recipient2)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	tests := []struct {
		name       string
		plaintext  []byte
		receivers  []saltpack.BoxPublicKey
		wantErr    bool
		wantErrEnc bool
		wantErrDec bool
	}{
		{
			name:      "single recipient",
			plaintext: []byte("Hello, World!"),
			receivers: []saltpack.BoxPublicKey{recipient1.PublicKey},
			wantErr:   false,
		},
		{
			name:      "multiple recipients",
			plaintext: []byte("Secret message for team"),
			receivers: []saltpack.BoxPublicKey{recipient1.PublicKey, recipient2.PublicKey},
			wantErr:   false,
		},
		{
			name:       "empty plaintext",
			plaintext:  []byte{},
			receivers:  []saltpack.BoxPublicKey{recipient1.PublicKey},
			wantErrEnc: true,
		},
		{
			name:       "no receivers",
			plaintext:  []byte("test"),
			receivers:  []saltpack.BoxPublicKey{},
			wantErrEnc: true,
		},
		{
			name:      "large message",
			plaintext: bytes.Repeat([]byte("A"), 10000),
			receivers: []saltpack.BoxPublicKey{recipient1.PublicKey},
			wantErr:   false,
		},
		{
			name:      "unicode message",
			plaintext: []byte("Hello 世界 🌍"),
			receivers: []saltpack.BoxPublicKey{recipient1.PublicKey},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext, tt.receivers)
			if (err != nil) != tt.wantErrEnc {
				t.Errorf("Encrypt() error = %v, wantErrEnc %v", err, tt.wantErrEnc)
				return
			}
			if tt.wantErrEnc {
				return
			}
			
			if len(ciphertext) == 0 {
				t.Error("Encrypt() returned empty ciphertext")
				return
			}
			
			// Decrypt
			decrypted, info, err := dec.Decrypt(ciphertext)
			if (err != nil) != tt.wantErrDec {
				t.Errorf("Decrypt() error = %v, wantErrDec %v", err, tt.wantErrDec)
				return
			}
			if tt.wantErrDec {
				return
			}
			
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted plaintext doesn't match original.\nGot:  %s\nWant: %s", 
					string(decrypted), string(tt.plaintext))
			}
			
			if info == nil {
				t.Error("Decrypt() returned nil MessageKeyInfo")
			}
		})
	}
}

// TestEncryptDecryptArmored tests ASCII-armored encryption and decryption
func TestEncryptDecryptArmored(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Create keyring
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("This is a secret message!")
	
	// Encrypt (armored)
	armoredCiphertext, err := enc.EncryptArmored(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("EncryptArmored() error = %v", err)
	}
	
	if len(armoredCiphertext) == 0 {
		t.Fatal("EncryptArmored() returned empty string")
	}
	
	// Check that it's ASCII (printable)
	if !isPrintableASCII(armoredCiphertext) {
		t.Error("EncryptArmored() returned non-ASCII string")
	}
	
	// Decrypt (armored)
	decrypted, info, err := dec.DecryptArmored(armoredCiphertext)
	if err != nil {
		t.Fatalf("DecryptArmored() error = %v", err)
	}
	
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted plaintext doesn't match original.\nGot:  %s\nWant: %s", 
			string(decrypted), string(plaintext))
	}
	
	if info == nil {
		t.Error("DecryptArmored() returned nil MessageKeyInfo")
	}
}

// TestEncryptDecryptStream tests streaming encryption and decryption
func TestEncryptDecryptStream(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Create keyring
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("This is a streaming test message!")
	
	// Encrypt (streaming)
	plaintextReader := bytes.NewReader(plaintext)
	var ciphertextBuf bytes.Buffer
	
	err = enc.EncryptStream(plaintextReader, &ciphertextBuf, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("EncryptStream() error = %v", err)
	}
	
	if ciphertextBuf.Len() == 0 {
		t.Fatal("EncryptStream() produced empty ciphertext")
	}
	
	// Decrypt (streaming)
	ciphertextReader := bytes.NewReader(ciphertextBuf.Bytes())
	var decryptedBuf bytes.Buffer
	
	info, err := dec.DecryptStream(ciphertextReader, &decryptedBuf)
	if err != nil {
		t.Fatalf("DecryptStream() error = %v", err)
	}
	
	if !bytes.Equal(decryptedBuf.Bytes(), plaintext) {
		t.Errorf("Decrypted plaintext doesn't match original.\nGot:  %s\nWant: %s", 
			decryptedBuf.String(), string(plaintext))
	}
	
	if info == nil {
		t.Error("DecryptStream() returned nil MessageKeyInfo")
	}
}

// TestEncryptDecryptStreamArmored tests armored streaming encryption and decryption
func TestEncryptDecryptStreamArmored(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Create keyring
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("This is an armored streaming test message!")
	
	// Encrypt (streaming, armored)
	plaintextReader := bytes.NewReader(plaintext)
	var armoredCiphertextBuf bytes.Buffer
	
	err = enc.EncryptStreamArmored(plaintextReader, &armoredCiphertextBuf, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("EncryptStreamArmored() error = %v", err)
	}
	
	if armoredCiphertextBuf.Len() == 0 {
		t.Fatal("EncryptStreamArmored() produced empty ciphertext")
	}
	
	// Check that it's ASCII
	if !isPrintableASCII(armoredCiphertextBuf.String()) {
		t.Error("EncryptStreamArmored() produced non-ASCII output")
	}
	
	// Decrypt (streaming, armored)
	armoredCiphertextReader := bytes.NewReader(armoredCiphertextBuf.Bytes())
	var decryptedBuf bytes.Buffer
	
	info, err := dec.DecryptStreamArmored(armoredCiphertextReader, &decryptedBuf)
	if err != nil {
		t.Fatalf("DecryptStreamArmored() error = %v", err)
	}
	
	if !bytes.Equal(decryptedBuf.Bytes(), plaintext) {
		t.Errorf("Decrypted plaintext doesn't match original.\nGot:  %s\nWant: %s", 
			decryptedBuf.String(), string(plaintext))
	}
	
	if info == nil {
		t.Error("DecryptStreamArmored() returned nil MessageKeyInfo")
	}
}

// TestEncryptWithContext tests context support
func TestEncryptWithContext(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	plaintext := []byte("Context test message")
	
	t.Run("valid context", func(t *testing.T) {
		ctx := context.Background()
		ciphertext, err := enc.EncryptWithContext(ctx, plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
		if err != nil {
			t.Errorf("EncryptWithContext() error = %v", err)
		}
		if len(ciphertext) == 0 {
			t.Error("EncryptWithContext() returned empty ciphertext")
		}
	})
	
	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		_, err := enc.EncryptWithContext(ctx, plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
		if err == nil {
			t.Error("EncryptWithContext() should fail with cancelled context")
		}
	})
	
	t.Run("expired context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		time.Sleep(10 * time.Millisecond) // Ensure timeout
		
		_, err := enc.EncryptWithContext(ctx, plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
		if err == nil {
			t.Error("EncryptWithContext() should fail with expired context")
		}
	})
}

// TestDecryptWithContext tests context support for decryption
func TestDecryptWithContext(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Create keyring
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("Context test message")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	
	t.Run("valid context", func(t *testing.T) {
		ctx := context.Background()
		decrypted, info, err := dec.DecryptWithContext(ctx, ciphertext)
		if err != nil {
			t.Errorf("DecryptWithContext() error = %v", err)
		}
		if !bytes.Equal(decrypted, plaintext) {
			t.Error("DecryptWithContext() returned wrong plaintext")
		}
		if info == nil {
			t.Error("DecryptWithContext() returned nil info")
		}
	})
	
	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		_, _, err := dec.DecryptWithContext(ctx, ciphertext)
		if err == nil {
			t.Error("DecryptWithContext() should fail with cancelled context")
		}
	})
}

// TestMultipleRecipients tests encryption for multiple recipients
func TestMultipleRecipients(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	numRecipients := 10
	recipients := make([]*KeyPair, numRecipients)
	publicKeys := make([]saltpack.BoxPublicKey, numRecipients)
	
	for i := 0; i < numRecipients; i++ {
		recipients[i], err = GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate recipient %d key: %v", i, err)
		}
		publicKeys[i] = recipients[i].PublicKey
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	plaintext := []byte("Message for all team members")
	
	// Encrypt for all recipients
	ciphertext, err := enc.Encrypt(plaintext, publicKeys)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Each recipient should be able to decrypt
	for i, recipient := range recipients {
		keyring := NewSimpleKeyring()
		keyring.AddKeyPair(recipient)
		keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
		
		dec, err := NewDecryptor(&DecryptorConfig{
			Keyring: keyring,
		})
		if err != nil {
			t.Fatalf("Failed to create decryptor for recipient %d: %v", i, err)
		}
		
		decrypted, info, err := dec.Decrypt(ciphertext)
		if err != nil {
			t.Errorf("Recipient %d failed to decrypt: %v", i, err)
			continue
		}
		
		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("Recipient %d decrypted wrong plaintext", i)
		}
		
		if info == nil {
			t.Errorf("Recipient %d got nil MessageKeyInfo", i)
		}
	}
}

// TestDecryptionWithWrongKey tests that decryption fails with wrong key
func TestDecryptionWithWrongKey(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	wrongRecipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate wrong recipient key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	plaintext := []byte("Secret message")
	
	// Encrypt for recipient
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Try to decrypt with wrong key
	wrongKeyring := NewSimpleKeyring()
	wrongKeyring.AddKeyPair(wrongRecipient)
	
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: wrongKeyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	_, _, err = dec.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

// isPrintableASCII checks if a string contains only printable ASCII characters
func isPrintableASCII(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			if r != '\n' && r != '\r' && r != '\t' {
				return false
			}
		}
	}
	return true
}

// BenchmarkEncrypt benchmarks encryption performance
func BenchmarkEncrypt(b *testing.B) {
	sender, _ := GenerateKeyPair()
	recipient, _ := GenerateKeyPair()
	
	enc, _ := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	
	plaintext := []byte("Benchmark message")
	receivers := []saltpack.BoxPublicKey{recipient.PublicKey}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encrypt(plaintext, receivers)
	}
}

// BenchmarkDecrypt benchmarks decryption performance
func BenchmarkDecrypt(b *testing.B) {
	sender, _ := GenerateKeyPair()
	recipient, _ := GenerateKeyPair()
	
	enc, _ := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	
	dec, _ := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	
	plaintext := []byte("Benchmark message")
	ciphertext, _ := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = dec.Decrypt(ciphertext)
	}
}
