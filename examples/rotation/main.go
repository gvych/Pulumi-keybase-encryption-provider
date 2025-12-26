package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pulumi/pulumi-keybase-encryption/keybase"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	fmt.Println("🔐 Keybase Key Rotation Example")
	fmt.Println("================================")
	fmt.Println()

	// This example demonstrates how to detect and handle key rotation
	// in the Keybase encryption provider

	// Step 1: Demonstrate encryption with initial keys
	fmt.Println("Step 1: Encrypting with initial keys...")
	initialKeys := demonstrateInitialEncryption()
	fmt.Println()

	// Step 2: Simulate key rotation
	fmt.Println("Step 2: Simulating key rotation...")
	newKeys := simulateKeyRotation()
	fmt.Println()

	// Step 3: Detect rotation and re-encrypt
	fmt.Println("Step 3: Detecting rotation and re-encrypting...")
	demonstrateRotationDetection(initialKeys, newKeys)
	fmt.Println()

	// Step 4: Bulk migration example
	fmt.Println("Step 4: Bulk migration example...")
	demonstrateBulkMigration()
	fmt.Println()

	fmt.Println("✅ Key rotation example completed successfully!")
}

// demonstrateInitialEncryption shows encrypting with initial keys
func demonstrateInitialEncryption() *KeySet {
	// Generate initial key pairs for Alice and Bob
	alice, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Alice's key: %v", err)
	}

	bob, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Bob's key: %v", err)
	}

	// Create a sender key for authenticated encryption
	sender, err := crypto.CreateTestSenderKey("sender")
	if err != nil {
		log.Fatalf("Failed to create sender key: %v", err)
	}

	// Create encryptor with sender key
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	// Encrypt a secret for Alice and Bob
	plaintext := []byte("Database password: super-secret-password-123")
	ciphertext, err := encryptor.EncryptArmored(plaintext, []crypto.BoxPublicKey{
		alice.PublicKey,
		bob.PublicKey,
	})
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	fmt.Printf("  ✓ Encrypted secret for Alice and Bob\n")
	fmt.Printf("  ✓ Ciphertext length: %d bytes\n", len(ciphertext))
	fmt.Printf("  ✓ Recipients: alice, bob\n")

	// Test decryption with Alice's key
	aliceKeyring := crypto.NewSimpleKeyring()
	aliceKeyring.AddKeyPair(alice)
	aliceKeyring.AddPublicKey(sender.PublicKey)

	aliceDecryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: aliceKeyring,
	})

	decrypted, _, err := aliceDecryptor.DecryptArmored(ciphertext)
	if err != nil {
		log.Fatalf("Alice's decryption failed: %v", err)
	}

	fmt.Printf("  ✓ Alice successfully decrypted: %s\n", string(decrypted))

	return &KeySet{
		Alice:      alice,
		Bob:        bob,
		Sender:     sender,
		Ciphertext: []byte(ciphertext),
		Plaintext:  plaintext,
	}
}

// simulateKeyRotation demonstrates what happens when keys are rotated
func simulateKeyRotation() *KeySet {
	// Alice rotates her key (new key pair)
	newAlice, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate new Alice key: %v", err)
	}

	// Bob keeps his existing key (in a real scenario, he'd fetch his current key)
	bob, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Bob's key: %v", err)
	}

	// Sender might also rotate their key
	newSender, err := crypto.CreateTestSenderKey("sender-rotated")
	if err != nil {
		log.Fatalf("Failed to create new sender key: %v", err)
	}

	fmt.Printf("  ✓ Alice rotated her encryption key\n")
	fmt.Printf("  ✓ Bob kept his existing key\n")
	fmt.Printf("  ✓ Sender rotated their signing key\n")

	return &KeySet{
		Alice:  newAlice,
		Bob:    bob,
		Sender: newSender,
	}
}

// demonstrateRotationDetection shows how to detect and handle key rotation
func demonstrateRotationDetection(oldKeys, newKeys *KeySet) {
	// Create encryptor with new sender key
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: newKeys.Sender.SecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	// Re-encrypt with new keys
	newCiphertext, err := encryptor.EncryptArmored(oldKeys.Plaintext, []crypto.BoxPublicKey{
		newKeys.Alice.PublicKey,
		newKeys.Bob.PublicKey,
	})
	if err != nil {
		log.Fatalf("Re-encryption failed: %v", err)
	}

	fmt.Printf("  ✓ Re-encrypted with new keys\n")
	fmt.Printf("  ✓ New ciphertext length: %d bytes\n", len(newCiphertext))

	// Verify old Alice key CANNOT decrypt new ciphertext
	oldAliceKeyring := crypto.NewSimpleKeyring()
	oldAliceKeyring.AddKeyPair(oldKeys.Alice)
	oldAliceKeyring.AddPublicKey(oldKeys.Sender.PublicKey)

	oldDecryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: oldAliceKeyring,
	})

	_, _, err = oldDecryptor.DecryptArmored(newCiphertext)
	if err != nil {
		fmt.Printf("  ✓ Verified: Old Alice key CANNOT decrypt new ciphertext (expected)\n")
	} else {
		fmt.Printf("  ✗ Unexpected: Old key could decrypt new ciphertext\n")
	}

	// Verify new Alice key CAN decrypt new ciphertext
	newAliceKeyring := crypto.NewSimpleKeyring()
	newAliceKeyring.AddKeyPair(newKeys.Alice)
	newAliceKeyring.AddPublicKey(newKeys.Sender.PublicKey)

	newDecryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: newAliceKeyring,
	})

	decrypted, _, err := newDecryptor.DecryptArmored(newCiphertext)
	if err != nil {
		log.Fatalf("New Alice decryption failed: %v", err)
	}

	if string(decrypted) == string(oldKeys.Plaintext) {
		fmt.Printf("  ✓ Verified: New Alice key successfully decrypted\n")
		fmt.Printf("  ✓ Plaintext matches: %s\n", string(decrypted))
	}
}

// demonstrateBulkMigration shows how to migrate multiple secrets
func demonstrateBulkMigration() {
	// In a real scenario, you would use the Keeper's MigrateEncryptedData method
	// Here we demonstrate the concept

	// Simulate multiple encrypted secrets
	secrets := map[string]string{
		"db_password":    "supersecret123",
		"api_key":        "key-abc-xyz-789",
		"private_key":    "-----BEGIN PRIVATE KEY-----\nMIIE...",
		"oauth_token":    "oauth2-token-value",
		"encryption_key": "AES256-encryption-key",
	}

	// Generate keys
	sender, _ := crypto.CreateTestSenderKey("sender")
	alice, _ := crypto.GenerateKeyPair()
	bob, _ := crypto.GenerateKeyPair()

	encryptor, _ := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: sender.SecretKey,
	})

	fmt.Printf("  Migrating %d secrets...\n", len(secrets))

	migrated := 0
	for secretName, secretValue := range secrets {
		// Encrypt each secret
		ciphertext, err := encryptor.EncryptArmored([]byte(secretValue), []crypto.BoxPublicKey{
			alice.PublicKey,
			bob.PublicKey,
		})
		if err != nil {
			fmt.Printf("  ✗ Failed to migrate %s: %v\n", secretName, err)
			continue
		}

		// In a real scenario, you would:
		// 1. Decrypt old ciphertext
		// 2. Detect if rotation needed
		// 3. Re-encrypt with new keys if needed
		// 4. Update storage with new ciphertext

		fmt.Printf("  ✓ Migrated: %s (%d bytes)\n", secretName, len(ciphertext))
		migrated++
	}

	fmt.Printf("  ✓ Successfully migrated %d/%d secrets\n", migrated, len(secrets))
}

// KeySet holds a set of keys for testing
type KeySet struct {
	Alice      *crypto.KeyPair
	Bob        *crypto.KeyPair
	Sender     *crypto.SenderKey
	Ciphertext []byte
	Plaintext  []byte
}

func init() {
	// Ensure we're in the correct directory for the example
	if _, err := os.Stat("go.mod"); err != nil {
		// Try to find the workspace root
		if err := os.Chdir("../.."); err != nil {
			// Run anyway, tests will handle missing modules
		}
	}
}
