package crypto

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/keybase/saltpack"
)

// TestEncryptionWithOneRecipient tests encryption with a single recipient
func TestEncryptionWithOneRecipient(t *testing.T) {
	// Generate keys
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
	
	plaintext := []byte("Secret message for one recipient")
	
	// Encrypt
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	if len(ciphertext) == 0 {
		t.Fatal("Encrypt() returned empty ciphertext")
	}
	
	// Create keyring with recipient's key
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey)
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	// Decrypt
	decrypted, info, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Verify plaintext
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted plaintext doesn't match.\nGot:  %s\nWant: %s", 
			string(decrypted), string(plaintext))
	}
	
	if info == nil {
		t.Error("Decrypt() returned nil MessageKeyInfo")
	}
	
	t.Logf("Successfully encrypted and decrypted message with 1 recipient")
}

// TestEncryptionWithFiveRecipients tests encryption with 5 recipients
func TestEncryptionWithFiveRecipients(t *testing.T) {
	// Generate sender key
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	// Generate 5 recipient keys
	numRecipients := 5
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
	
	plaintext := []byte("Secret message for five recipients")
	
	// Encrypt for all recipients
	ciphertext, err := enc.Encrypt(plaintext, publicKeys)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	if len(ciphertext) == 0 {
		t.Fatal("Encrypt() returned empty ciphertext")
	}
	
	// Verify each recipient can decrypt independently
	for i, recipient := range recipients {
		t.Run(fmt.Sprintf("Recipient_%d", i+1), func(t *testing.T) {
			// Create keyring with only this recipient's key
			keyring := NewSimpleKeyring()
			keyring.AddKeyPair(recipient)
			keyring.AddPublicKey(sender.PublicKey)
			
			// Create decryptor
			dec, err := NewDecryptor(&DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}
			
			// Decrypt
			decrypted, info, err := dec.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Recipient %d failed to decrypt: %v", i+1, err)
			}
			
			// Verify plaintext
			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("Recipient %d decrypted wrong plaintext", i+1)
			}
			
			if info == nil {
				t.Errorf("Recipient %d got nil MessageKeyInfo", i+1)
			}
		})
	}
	
	t.Logf("Successfully encrypted and decrypted message with 5 recipients")
}

// TestEncryptionWithTenRecipients tests encryption with 10 recipients
func TestEncryptionWithTenRecipients(t *testing.T) {
	// Generate sender key
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	// Generate 10 recipient keys
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
	
	plaintext := []byte("Secret message for ten recipients")
	
	// Encrypt for all recipients
	ciphertext, err := enc.Encrypt(plaintext, publicKeys)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	if len(ciphertext) == 0 {
		t.Fatal("Encrypt() returned empty ciphertext")
	}
	
	// Verify each recipient can decrypt independently
	successCount := 0
	for i, recipient := range recipients {
		t.Run(fmt.Sprintf("Recipient_%d", i+1), func(t *testing.T) {
			// Create keyring with only this recipient's key
			keyring := NewSimpleKeyring()
			keyring.AddKeyPair(recipient)
			keyring.AddPublicKey(sender.PublicKey)
			
			// Create decryptor
			dec, err := NewDecryptor(&DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}
			
			// Decrypt
			decrypted, info, err := dec.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Recipient %d failed to decrypt: %v", i+1, err)
			}
			
			// Verify plaintext
			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("Recipient %d decrypted wrong plaintext", i+1)
			}
			
			if info == nil {
				t.Errorf("Recipient %d got nil MessageKeyInfo", i+1)
			}
			
			successCount++
		})
	}
	
	if successCount != numRecipients {
		t.Errorf("Only %d/%d recipients could decrypt successfully", successCount, numRecipients)
	}
	
	t.Logf("Successfully encrypted and decrypted message with 10 recipients")
}

// TestAllRecipientsCanDecryptIndependently verifies that each recipient can decrypt
// without knowing about other recipients
func TestAllRecipientsCanDecryptIndependently(t *testing.T) {
	recipientCounts := []int{1, 5, 10}
	
	for _, numRecipients := range recipientCounts {
		t.Run(fmt.Sprintf("%d_recipients", numRecipients), func(t *testing.T) {
			// Generate sender key
			sender, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("Failed to generate sender key: %v", err)
			}
			
			// Generate recipient keys
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
			
			plaintext := []byte(fmt.Sprintf("Test message for %d recipients", numRecipients))
			
			// Encrypt
			ciphertext, err := enc.Encrypt(plaintext, publicKeys)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			
			// Each recipient decrypts independently
			for i, recipient := range recipients {
				// Create isolated keyring with ONLY this recipient's key
				keyring := NewSimpleKeyring()
				keyring.AddKeyPair(recipient)
				keyring.AddPublicKey(sender.PublicKey)
				
				dec, err := NewDecryptor(&DecryptorConfig{
					Keyring: keyring,
				})
				if err != nil {
					t.Fatalf("Failed to create decryptor for recipient %d: %v", i, err)
				}
				
				// Decrypt
				decrypted, _, err := dec.Decrypt(ciphertext)
				if err != nil {
					t.Errorf("Recipient %d failed to decrypt independently: %v", i, err)
					continue
				}
				
				// Verify
				if !bytes.Equal(decrypted, plaintext) {
					t.Errorf("Recipient %d got wrong plaintext", i)
				}
			}
			
			t.Logf("All %d recipients successfully decrypted independently", numRecipients)
		})
	}
}

// TestVeryLargeMessage tests encryption with very large messages
func TestVeryLargeMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large message test in short mode")
	}
	
	messageSizes := []struct {
		name string
		size int
	}{
		{"1MB", 1 * 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}
	
	for _, tc := range messageSizes {
		t.Run(tc.name, func(t *testing.T) {
			// Generate keys
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
			
			// Create large plaintext
			plaintext := make([]byte, tc.size)
			for i := 0; i < len(plaintext); i++ {
				plaintext[i] = byte(i % 256)
			}
			
			t.Logf("Testing encryption of %s message (%d bytes)", tc.name, tc.size)
			
			// Encrypt
			startEncrypt := time.Now()
			ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
			encryptDuration := time.Since(startEncrypt)
			
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			
			t.Logf("Encryption took %v for %s", encryptDuration, tc.name)
			
			// Create keyring
			keyring := NewSimpleKeyring()
			keyring.AddKeyPair(recipient)
			keyring.AddPublicKey(sender.PublicKey)
			
			// Create decryptor
			dec, err := NewDecryptor(&DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}
			
			// Decrypt
			startDecrypt := time.Now()
			decrypted, _, err := dec.Decrypt(ciphertext)
			decryptDuration := time.Since(startDecrypt)
			
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			
			t.Logf("Decryption took %v for %s", decryptDuration, tc.name)
			
			// Verify (sample check to avoid full comparison for huge files)
			if len(decrypted) != len(plaintext) {
				t.Errorf("Decrypted length mismatch: got %d, want %d", len(decrypted), len(plaintext))
			} else {
				// Check first, middle, and last chunks
				chunkSize := 1024
				checks := []int{0, len(plaintext)/2 - chunkSize/2, len(plaintext) - chunkSize}
				for _, start := range checks {
					if start >= 0 && start+chunkSize <= len(plaintext) {
						if !bytes.Equal(decrypted[start:start+chunkSize], plaintext[start:start+chunkSize]) {
							t.Errorf("Decrypted data mismatch at position %d", start)
							break
						}
					}
				}
			}
			
			t.Logf("Successfully encrypted and decrypted %s message", tc.name)
		})
	}
}

// TestVeryLargeMessageStreaming tests streaming encryption for very large messages
func TestVeryLargeMessageStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large message streaming test in short mode")
	}
	
	// Test 100MB with streaming (more memory efficient)
	messageSize := 100 * 1024 * 1024
	
	// Generate keys
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
	
	// Create large plaintext
	plaintext := make([]byte, messageSize)
	for i := 0; i < len(plaintext); i++ {
		plaintext[i] = byte(i % 256)
	}
	
	t.Logf("Testing streaming encryption of 100MB message")
	
	// Encrypt using streaming
	plaintextReader := bytes.NewReader(plaintext)
	var ciphertextBuf bytes.Buffer
	
	startEncrypt := time.Now()
	err = enc.EncryptStream(plaintextReader, &ciphertextBuf, []saltpack.BoxPublicKey{recipient.PublicKey})
	encryptDuration := time.Since(startEncrypt)
	
	if err != nil {
		t.Fatalf("EncryptStream() error = %v", err)
	}
	
	t.Logf("Streaming encryption took %v", encryptDuration)
	
	// Create keyring
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey)
	
	// Create decryptor
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	// Decrypt using streaming
	ciphertextReader := bytes.NewReader(ciphertextBuf.Bytes())
	var decryptedBuf bytes.Buffer
	
	startDecrypt := time.Now()
	_, err = dec.DecryptStream(ciphertextReader, &decryptedBuf)
	decryptDuration := time.Since(startDecrypt)
	
	if err != nil {
		t.Fatalf("DecryptStream() error = %v", err)
	}
	
	t.Logf("Streaming decryption took %v", decryptDuration)
	
	// Verify length
	if decryptedBuf.Len() != len(plaintext) {
		t.Errorf("Decrypted length mismatch: got %d, want %d", decryptedBuf.Len(), len(plaintext))
	}
	
	t.Logf("Successfully encrypted and decrypted 100MB message using streaming")
}

// BenchmarkEncryptMultipleRecipients benchmarks encryption with different numbers of recipients
func BenchmarkEncryptMultipleRecipients(b *testing.B) {
	recipientCounts := []int{1, 5, 10}
	
	for _, numRecipients := range recipientCounts {
		b.Run(fmt.Sprintf("%d_recipients", numRecipients), func(b *testing.B) {
			// Generate sender key
			sender, err := GenerateKeyPair()
			if err != nil {
				b.Fatalf("Failed to generate sender key: %v", err)
			}
			
			// Generate recipient keys
			publicKeys := make([]saltpack.BoxPublicKey, numRecipients)
			for i := 0; i < numRecipients; i++ {
				recipient, err := GenerateKeyPair()
				if err != nil {
					b.Fatalf("Failed to generate recipient key: %v", err)
				}
				publicKeys[i] = recipient.PublicKey
			}
			
			// Create encryptor
			enc, err := NewEncryptor(&EncryptorConfig{
				SenderKey: sender.SecretKey,
			})
			if err != nil {
				b.Fatalf("Failed to create encryptor: %v", err)
			}
			
			plaintext := []byte("Benchmark message for multiple recipients")
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := enc.Encrypt(plaintext, publicKeys)
				if err != nil {
					b.Fatalf("Encrypt() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkEncryptionLatency benchmarks encryption latency to verify <500ms target
func BenchmarkEncryptionLatency(b *testing.B) {
	// Generate sender key
	sender, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate sender key: %v", err)
	}
	
	// Generate 10 recipient keys (target scenario)
	numRecipients := 10
	publicKeys := make([]saltpack.BoxPublicKey, numRecipients)
	for i := 0; i < numRecipients; i++ {
		recipient, err := GenerateKeyPair()
		if err != nil {
			b.Fatalf("Failed to generate recipient key: %v", err)
		}
		publicKeys[i] = recipient.PublicKey
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		b.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Test with typical Pulumi secret size (1KB)
	plaintext := bytes.Repeat([]byte("A"), 1024)
	
	b.ResetTimer()
	
	var totalDuration time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := enc.Encrypt(plaintext, publicKeys)
		duration := time.Since(start)
		totalDuration += duration
		
		if err != nil {
			b.Fatalf("Encrypt() error = %v", err)
		}
	}
	
	avgDuration := totalDuration / time.Duration(b.N)
	b.ReportMetric(float64(avgDuration.Microseconds())/1000.0, "ms/op")
	
	// Check if we meet the <500ms target
	if avgDuration > 500*time.Millisecond {
		b.Logf("WARNING: Average latency %v exceeds 500ms target", avgDuration)
	} else {
		b.Logf("SUCCESS: Average latency %v is within 500ms target", avgDuration)
	}
}

// BenchmarkEncryptionLatencyWithContext benchmarks encryption with context
func BenchmarkEncryptionLatencyWithContext(b *testing.B) {
	// Generate sender key
	sender, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate sender key: %v", err)
	}
	
	// Generate 10 recipient keys
	numRecipients := 10
	publicKeys := make([]saltpack.BoxPublicKey, numRecipients)
	for i := 0; i < numRecipients; i++ {
		recipient, err := GenerateKeyPair()
		if err != nil {
			b.Fatalf("Failed to generate recipient key: %v", err)
		}
		publicKeys[i] = recipient.PublicKey
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		b.Fatalf("Failed to create encryptor: %v", err)
	}
	
	plaintext := bytes.Repeat([]byte("A"), 1024)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enc.EncryptWithContext(ctx, plaintext, publicKeys)
		if err != nil {
			b.Fatalf("EncryptWithContext() error = %v", err)
		}
	}
}

// BenchmarkDecryptMultipleRecipients benchmarks decryption with messages encrypted for multiple recipients
func BenchmarkDecryptMultipleRecipients(b *testing.B) {
	recipientCounts := []int{1, 5, 10}
	
	for _, numRecipients := range recipientCounts {
		b.Run(fmt.Sprintf("%d_recipients", numRecipients), func(b *testing.B) {
			// Generate sender key
			sender, err := GenerateKeyPair()
			if err != nil {
				b.Fatalf("Failed to generate sender key: %v", err)
			}
			
			// Generate recipient keys
			publicKeys := make([]saltpack.BoxPublicKey, numRecipients)
			var testRecipient *KeyPair
			
			for i := 0; i < numRecipients; i++ {
				recipient, err := GenerateKeyPair()
				if err != nil {
					b.Fatalf("Failed to generate recipient key: %v", err)
				}
				publicKeys[i] = recipient.PublicKey
				if i == 0 {
					testRecipient = recipient
				}
			}
			
			// Create encryptor and encrypt
			enc, err := NewEncryptor(&EncryptorConfig{
				SenderKey: sender.SecretKey,
			})
			if err != nil {
				b.Fatalf("Failed to create encryptor: %v", err)
			}
			
			plaintext := []byte("Benchmark message")
			ciphertext, err := enc.Encrypt(plaintext, publicKeys)
			if err != nil {
				b.Fatalf("Encrypt() error = %v", err)
			}
			
			// Create keyring with first recipient's key
			keyring := NewSimpleKeyring()
			keyring.AddKeyPair(testRecipient)
			keyring.AddPublicKey(sender.PublicKey)
			
			// Create decryptor
			dec, err := NewDecryptor(&DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				b.Fatalf("Failed to create decryptor: %v", err)
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := dec.Decrypt(ciphertext)
				if err != nil {
					b.Fatalf("Decrypt() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkEncryptLargeMessage benchmarks encryption of large messages
func BenchmarkEncryptLargeMessage(b *testing.B) {
	messageSizes := []int{
		1 * 1024,      // 1KB
		10 * 1024,     // 10KB
		100 * 1024,    // 100KB
		1024 * 1024,   // 1MB
	}
	
	for _, size := range messageSizes {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			// Generate keys
			sender, err := GenerateKeyPair()
			if err != nil {
				b.Fatalf("Failed to generate sender key: %v", err)
			}
			
			recipient, err := GenerateKeyPair()
			if err != nil {
				b.Fatalf("Failed to generate recipient key: %v", err)
			}
			
			// Create encryptor
			enc, err := NewEncryptor(&EncryptorConfig{
				SenderKey: sender.SecretKey,
			})
			if err != nil {
				b.Fatalf("Failed to create encryptor: %v", err)
			}
			
			plaintext := bytes.Repeat([]byte("A"), size)
			
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
				if err != nil {
					b.Fatalf("Encrypt() error = %v", err)
				}
			}
		})
	}
}
