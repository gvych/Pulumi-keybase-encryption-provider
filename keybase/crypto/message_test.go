package crypto

import (
	"bytes"
	"strings"
	"testing"

	"github.com/keybase/saltpack"
)

// TestParseMessageKeyInfo tests parsing of MessageKeyInfo
func TestParseMessageKeyInfo(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor and decryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey)
	
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	// Encrypt a message
	plaintext := []byte("Test message for header parsing")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Decrypt and get MessageKeyInfo
	_, messageKeyInfo, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Parse MessageKeyInfo
	messageInfo, err := ParseMessageKeyInfo(messageKeyInfo)
	if err != nil {
		t.Fatalf("ParseMessageKeyInfo() error = %v", err)
	}
	
	// Verify parsed information
	if messageInfo == nil {
		t.Fatal("ParseMessageKeyInfo() returned nil")
	}
	
	if len(messageInfo.ReceiverKID) == 0 {
		t.Error("ReceiverKID is empty")
	}
	
	if messageInfo.ReceiverKIDHex == "" {
		t.Error("ReceiverKIDHex is empty")
	}
	
	if messageInfo.IsAnonymousSender {
		t.Error("Expected non-anonymous sender, got anonymous")
	}
	
	if len(messageInfo.SenderKID) == 0 {
		t.Error("SenderKID is empty")
	}
	
	if messageInfo.SenderKIDHex == "" {
		t.Error("SenderKIDHex is empty")
	}
	
	// Verify receiver key matches
	expectedReceiverKID := recipient.PublicKey.ToKID()
	if !bytes.Equal(messageInfo.ReceiverKID, expectedReceiverKID) {
		t.Errorf("ReceiverKID mismatch: got %x, want %x", messageInfo.ReceiverKID, expectedReceiverKID)
	}
	
	// Verify sender key matches
	expectedSenderKID := sender.PublicKey.ToKID()
	if !bytes.Equal(messageInfo.SenderKID, expectedSenderKID) {
		t.Errorf("SenderKID mismatch: got %x, want %x", messageInfo.SenderKID, expectedSenderKID)
	}
}

// TestParseMessageKeyInfoAnonymous tests parsing with anonymous sender
func TestParseMessageKeyInfoAnonymous(t *testing.T) {
	// Generate test keys
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create encryptor with nil sender (anonymous)
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: nil,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	// Encrypt a message
	plaintext := []byte("Anonymous message")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Decrypt and get MessageKeyInfo
	_, messageKeyInfo, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Parse MessageKeyInfo
	messageInfo, err := ParseMessageKeyInfo(messageKeyInfo)
	if err != nil {
		t.Fatalf("ParseMessageKeyInfo() error = %v", err)
	}
	
	// Verify anonymous sender
	if !messageInfo.IsAnonymousSender {
		t.Error("Expected anonymous sender, got non-anonymous")
	}
	
	if len(messageInfo.SenderKID) != 0 {
		t.Error("SenderKID should be empty for anonymous sender")
	}
	
	if messageInfo.SenderKIDHex != "" {
		t.Error("SenderKIDHex should be empty for anonymous sender")
	}
	
	// Receiver should still be valid
	if len(messageInfo.ReceiverKID) == 0 {
		t.Error("ReceiverKID is empty")
	}
}

// TestParseMessageKeyInfoNil tests error handling for nil input
func TestParseMessageKeyInfoNil(t *testing.T) {
	_, err := ParseMessageKeyInfo(nil)
	if err == nil {
		t.Error("ParseMessageKeyInfo(nil) should return error")
	}
}

// TestGetReceiverKeyID tests extracting receiver key ID
func TestGetReceiverKeyID(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create and encrypt
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("Test message")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Decrypt
	_, messageKeyInfo, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Get receiver key ID
	receiverKID, err := GetReceiverKeyID(messageKeyInfo)
	if err != nil {
		t.Fatalf("GetReceiverKeyID() error = %v", err)
	}
	
	// Verify
	expectedKID := recipient.PublicKey.ToKID()
	if !bytes.Equal(receiverKID, expectedKID) {
		t.Errorf("ReceiverKID mismatch: got %x, want %x", receiverKID, expectedKID)
	}
}

// TestGetReceiverKeyIDNil tests error handling for nil input
func TestGetReceiverKeyIDNil(t *testing.T) {
	_, err := GetReceiverKeyID(nil)
	if err == nil {
		t.Error("GetReceiverKeyID(nil) should return error")
	}
}

// TestGetSenderKeyID tests extracting sender key ID
func TestGetSenderKeyID(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Test with non-anonymous sender
	t.Run("non-anonymous sender", func(t *testing.T) {
		enc, err := NewEncryptor(&EncryptorConfig{
			SenderKey: sender.SecretKey,
		})
		if err != nil {
			t.Fatalf("Failed to create encryptor: %v", err)
		}
		
		keyring := NewSimpleKeyring()
		keyring.AddKeyPair(recipient)
		keyring.AddPublicKey(sender.PublicKey)
		
		dec, err := NewDecryptor(&DecryptorConfig{
			Keyring: keyring,
		})
		if err != nil {
			t.Fatalf("Failed to create decryptor: %v", err)
		}
		
		plaintext := []byte("Test message")
		ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		
		_, messageKeyInfo, err := dec.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}
		
		senderKID := GetSenderKeyID(messageKeyInfo)
		if senderKID == nil {
			t.Error("GetSenderKeyID() returned nil for non-anonymous sender")
		}
		
		expectedKID := sender.PublicKey.ToKID()
		if !bytes.Equal(senderKID, expectedKID) {
			t.Errorf("SenderKID mismatch: got %x, want %x", senderKID, expectedKID)
		}
	})
	
	// Test with anonymous sender
	t.Run("anonymous sender", func(t *testing.T) {
		enc, err := NewEncryptor(&EncryptorConfig{
			SenderKey: nil,
		})
		if err != nil {
			t.Fatalf("Failed to create encryptor: %v", err)
		}
		
		keyring := NewSimpleKeyring()
		keyring.AddKeyPair(recipient)
		
		dec, err := NewDecryptor(&DecryptorConfig{
			Keyring: keyring,
		})
		if err != nil {
			t.Fatalf("Failed to create decryptor: %v", err)
		}
		
		plaintext := []byte("Anonymous message")
		ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		
		_, messageKeyInfo, err := dec.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}
		
		senderKID := GetSenderKeyID(messageKeyInfo)
		if senderKID != nil {
			t.Error("GetSenderKeyID() should return nil for anonymous sender")
		}
	})
}

// TestIsAnonymousSender tests checking for anonymous sender
func TestIsAnonymousSender(t *testing.T) {
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	tests := []struct {
		name      string
		senderKey saltpack.BoxSecretKey
		want      bool
	}{
		{
			name:      "anonymous sender",
			senderKey: nil,
			want:      true,
		},
		{
			name:      "non-anonymous sender",
			senderKey: sender.SecretKey,
			want:      false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncryptor(&EncryptorConfig{
				SenderKey: tt.senderKey,
			})
			if err != nil {
				t.Fatalf("Failed to create encryptor: %v", err)
			}
			
			keyring := NewSimpleKeyring()
			keyring.AddKeyPair(recipient)
			if tt.senderKey != nil {
				keyring.AddPublicKey(sender.PublicKey)
			}
			
			dec, err := NewDecryptor(&DecryptorConfig{
				Keyring: keyring,
			})
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}
			
			plaintext := []byte("Test message")
			ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			
			_, messageKeyInfo, err := dec.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			
			got := IsAnonymousSender(messageKeyInfo)
			if got != tt.want {
				t.Errorf("IsAnonymousSender() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVerifySender tests verifying sender identity
func TestVerifySender(t *testing.T) {
	// Generate test keys
	sender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	
	wrongSender, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate wrong sender key: %v", err)
	}
	
	recipient, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}
	
	// Create and encrypt
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey)
	
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("Test message")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	_, messageKeyInfo, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Test verification
	if !VerifySender(messageKeyInfo, sender.PublicKey) {
		t.Error("VerifySender() should return true for correct sender")
	}
	
	if VerifySender(messageKeyInfo, wrongSender.PublicKey) {
		t.Error("VerifySender() should return false for wrong sender")
	}
	
	if VerifySender(messageKeyInfo, nil) {
		t.Error("VerifySender() should return false for nil key")
	}
}

// TestVerifyReceiver tests verifying receiver identity
func TestVerifyReceiver(t *testing.T) {
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
	
	// Create and encrypt
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	keyring := NewSimpleKeyring()
	keyring.AddKeyPair(recipient)
	keyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	dec, err := NewDecryptor(&DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}
	
	plaintext := []byte("Test message")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{recipient.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	_, messageKeyInfo, err := dec.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	// Test verification
	if !VerifyReceiver(messageKeyInfo, recipient.PublicKey) {
		t.Error("VerifyReceiver() should return true for correct receiver")
	}
	
	if VerifyReceiver(messageKeyInfo, wrongRecipient.PublicKey) {
		t.Error("VerifyReceiver() should return false for wrong receiver")
	}
	
	if VerifyReceiver(messageKeyInfo, nil) {
		t.Error("VerifyReceiver() should return false for nil key")
	}
}

// TestMultipleRecipientsKeyInfo tests that correct receiver is identified with multiple recipients
func TestMultipleRecipientsKeyInfo(t *testing.T) {
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
	
	recipient3, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient3 key: %v", err)
	}
	
	// Create encryptor
	enc, err := NewEncryptor(&EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	
	// Encrypt for all three recipients
	plaintext := []byte("Message for all team members")
	ciphertext, err := enc.Encrypt(plaintext, []saltpack.BoxPublicKey{
		recipient1.PublicKey,
		recipient2.PublicKey,
		recipient3.PublicKey,
	})
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Each recipient decrypts and verifies they are the receiver
	recipients := []*KeyPair{recipient1, recipient2, recipient3}
	
	for i, recip := range recipients {
		keyring := NewSimpleKeyring()
		keyring.AddKeyPair(recip)
		keyring.AddPublicKey(sender.PublicKey)
		
		dec, err := NewDecryptor(&DecryptorConfig{
			Keyring: keyring,
		})
		if err != nil {
			t.Fatalf("Failed to create decryptor for recipient %d: %v", i, err)
		}
		
		decrypted, messageKeyInfo, err := dec.Decrypt(ciphertext)
		if err != nil {
			t.Errorf("Recipient %d failed to decrypt: %v", i, err)
			continue
		}
		
		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("Recipient %d decrypted wrong plaintext", i)
		}
		
		// Verify this recipient is the one who decrypted
		if !VerifyReceiver(messageKeyInfo, recip.PublicKey) {
			t.Errorf("Recipient %d: VerifyReceiver failed for own key", i)
		}
		
		// Verify other recipients are not the decryptor
		for j, otherRecip := range recipients {
			if i != j {
				if VerifyReceiver(messageKeyInfo, otherRecip.PublicKey) {
					t.Errorf("Recipient %d: VerifyReceiver incorrectly returned true for recipient %d's key", i, j)
				}
			}
		}
		
		// Verify sender
		if !VerifySender(messageKeyInfo, sender.PublicKey) {
			t.Errorf("Recipient %d: VerifySender failed", i)
		}
	}
}

// TestFormatKeyID tests key ID formatting
func TestFormatKeyID(t *testing.T) {
	tests := []struct {
		name string
		kid  []byte
		want string
	}{
		{
			name: "empty key ID",
			kid:  []byte{},
			want: "<empty>",
		},
		{
			name: "short key ID",
			kid:  []byte{0x01, 0x02, 0x03, 0x04},
			want: "01020304",
		},
		{
			name: "32-byte key ID",
			kid:  bytes.Repeat([]byte{0xAB}, 32),
			want: strings.Repeat("ab", 32),
		},
		{
			name: "long key ID (truncated)",
			kid:  bytes.Repeat([]byte{0xCD}, 40),
			want: "cdcdcdcd...cdcdcdcd",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatKeyID(tt.kid)
			if got != tt.want {
				t.Errorf("FormatKeyID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMessageInfoString tests string representation of MessageInfo
func TestMessageInfoString(t *testing.T) {
	// Test nil
	if str := MessageInfoString(nil); str != "<nil MessageInfo>" {
		t.Errorf("MessageInfoString(nil) = %v, want %v", str, "<nil MessageInfo>")
	}
	
	// Test with data
	info := &MessageInfo{
		ReceiverKID:       []byte{0x01, 0x02, 0x03, 0x04},
		ReceiverKIDHex:    "01020304",
		SenderKID:         []byte{0x05, 0x06, 0x07, 0x08},
		SenderKIDHex:      "05060708",
		IsAnonymousSender: false,
		ReceiverIndex:     2,
	}
	
	str := MessageInfoString(info)
	
	// Check that key components are in the string
	if !strings.Contains(str, "01020304") {
		t.Error("MessageInfoString should contain receiver KID")
	}
	
	if !strings.Contains(str, "05060708") {
		t.Error("MessageInfoString should contain sender KID")
	}
	
	if !strings.Contains(str, "ReceiverIndex: 2") {
		t.Error("MessageInfoString should contain receiver index")
	}
	
	// Test anonymous sender
	infoAnon := &MessageInfo{
		ReceiverKID:       []byte{0x01, 0x02, 0x03, 0x04},
		IsAnonymousSender: true,
		ReceiverIndex:     -1,
	}
	
	strAnon := MessageInfoString(infoAnon)
	
	if !strings.Contains(strAnon, "<anonymous>") {
		t.Error("MessageInfoString should contain <anonymous> for anonymous sender")
	}
	
	if !strings.Contains(strAnon, "<unknown>") {
		t.Error("MessageInfoString should contain <unknown> for unknown receiver index")
	}
}
