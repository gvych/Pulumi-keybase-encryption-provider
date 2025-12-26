package main

import (
	"fmt"
	"log"
	"os"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	fmt.Println("=== Keybase Sender Key Example ===\n")

	// Step 1: Verify Keybase is available
	fmt.Println("Step 1: Verifying Keybase installation...")
	if err := credentials.VerifyKeybaseAvailable(); err != nil {
		// Keybase is not available - demonstrate with test keys instead
		fmt.Printf("Warning: Keybase not available (%v)\n", err)
		fmt.Println("Falling back to test keys for demonstration...\n")
		demonstrateWithTestKeys()
		return
	}

	// Step 2: Discover credentials
	fmt.Println("Step 2: Discovering Keybase credentials...")
	status, err := credentials.DiscoverCredentials()
	if err != nil {
		log.Fatalf("Failed to discover credentials: %v", err)
	}

	fmt.Printf("  ✓ Keybase CLI: %s\n", status.CLIPath)
	fmt.Printf("  ✓ Config directory: %s\n", status.ConfigDir)
	fmt.Printf("  ✓ Logged in as: %s\n\n", status.Username)

	// Step 3: Load sender key
	fmt.Println("Step 3: Loading sender key from Keybase...")
	senderKey, err := crypto.LoadSenderKey(nil)
	if err != nil {
		// If loading from Keybase fails, fall back to test keys
		fmt.Printf("Warning: Failed to load sender key (%v)\n", err)
		fmt.Println("Falling back to test keys for demonstration...\n")
		demonstrateWithTestKeys()
		return
	}

	fmt.Printf("  ✓ Loaded key for user: %s\n", senderKey.Username)
	fmt.Printf("  ✓ Key ID: %s\n\n", crypto.KeyIDToHex(senderKey.KeyID)[:16]+"...")

	// Step 4: Validate sender key
	fmt.Println("Step 4: Validating sender key...")
	if err := crypto.ValidateSenderKey(senderKey); err != nil {
		log.Fatalf("Invalid sender key: %v", err)
	}
	fmt.Println("  ✓ Sender key is valid\n")

	// Step 5: Generate a recipient key (simulating another user)
	fmt.Println("Step 5: Generating recipient key...")
	recipientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate recipient key: %v", err)
	}
	fmt.Println("  ✓ Recipient key generated\n")

	// Step 6: Create encryptor with sender key
	fmt.Println("Step 6: Creating encryptor with sender key...")
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: senderKey.SecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}
	fmt.Println("  ✓ Encryptor created\n")

	// Step 7: Encrypt a message
	fmt.Println("Step 7: Encrypting message...")
	plaintext := []byte("This is a secret message authenticated by my Keybase identity!")
	ciphertext, err := encryptor.EncryptArmored(plaintext, []saltpack.BoxPublicKey{recipientKey.PublicKey})
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	fmt.Printf("  ✓ Message encrypted (%d bytes)\n", len(ciphertext))
	fmt.Printf("  Ciphertext preview: %s...\n\n", ciphertext[:60])

	// Step 8: Create keyring with recipient's secret key and sender's public key
	fmt.Println("Step 8: Setting up recipient keyring...")
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKeyPair(recipientKey)
	keyring.AddPublicKey(senderKey.PublicKey) // For sender verification
	fmt.Println("  ✓ Keyring configured\n")

	// Step 9: Create decryptor
	fmt.Println("Step 9: Creating decryptor...")
	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		log.Fatalf("Failed to create decryptor: %v", err)
	}
	fmt.Println("  ✓ Decryptor created\n")

	// Step 10: Decrypt and verify
	fmt.Println("Step 10: Decrypting and verifying message...")
	decrypted, info, err := decryptor.DecryptArmored(ciphertext)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}

	fmt.Printf("  ✓ Message decrypted\n")
	fmt.Printf("  ✓ Sender authenticated: %v\n", !info.SenderIsAnon)
	fmt.Printf("  Decrypted message: \"%s\"\n\n", string(decrypted))

	// Verify the message matches
	if string(decrypted) != string(plaintext) {
		log.Fatal("ERROR: Decrypted message does not match original!")
	}

	fmt.Println("=== Success! ===")
	fmt.Println("The message was encrypted with your Keybase identity and")
	fmt.Println("successfully decrypted and verified by the recipient.")
}

// demonstrateWithTestKeys shows how sender keys work using test keys
// when Keybase is not available
func demonstrateWithTestKeys() {
	fmt.Println("=== Using Test Keys for Demonstration ===\n")

	// Create a test sender key
	fmt.Println("Creating test sender key...")
	senderKey, err := crypto.CreateTestSenderKey("alice")
	if err != nil {
		log.Fatalf("Failed to create test sender key: %v", err)
	}
	fmt.Printf("  ✓ Test sender key created for user: %s\n\n", senderKey.Username)

	// Validate the test key
	fmt.Println("Validating test sender key...")
	if err := crypto.ValidateSenderKey(senderKey); err != nil {
		log.Fatalf("Invalid test sender key: %v", err)
	}
	fmt.Println("  ✓ Test sender key is valid\n")

	// Generate recipient key
	fmt.Println("Generating recipient key...")
	recipientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate recipient key: %v", err)
	}
	fmt.Println("  ✓ Recipient key generated\n")

	// Create encryptor with test sender key
	fmt.Println("Creating encryptor with test sender key...")
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: senderKey.SecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}
	fmt.Println("  ✓ Encryptor created\n")

	// Encrypt a message
	fmt.Println("Encrypting message...")
	plaintext := []byte("This is a test message with sender authentication!")
	ciphertext, err := encryptor.EncryptArmored(plaintext, []saltpack.BoxPublicKey{recipientKey.PublicKey})
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	fmt.Printf("  ✓ Message encrypted (%d bytes)\n\n", len(ciphertext))

	// Create keyring
	fmt.Println("Setting up recipient keyring...")
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKeyPair(recipientKey)
	keyring.AddPublicKey(senderKey.PublicKey)
	fmt.Println("  ✓ Keyring configured\n")

	// Create decryptor
	fmt.Println("Creating decryptor...")
	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		log.Fatalf("Failed to create decryptor: %v", err)
	}
	fmt.Println("  ✓ Decryptor created\n")

	// Decrypt and verify
	fmt.Println("Decrypting and verifying message...")
	decrypted, info, err := decryptor.DecryptArmored(ciphertext)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}

	fmt.Printf("  ✓ Message decrypted\n")
	fmt.Printf("  ✓ Sender authenticated: %v\n", !info.SenderIsAnon)
	fmt.Printf("  Decrypted message: \"%s\"\n\n", string(decrypted))

	// Save test key to demonstrate loading
	fmt.Println("Testing key save and load...")
	tempDir, err := os.MkdirTemp("", "keybase-test-*")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := crypto.SaveSenderKeyForTesting(senderKey, tempDir); err != nil {
		log.Fatalf("Failed to save test key: %v", err)
	}
	fmt.Printf("  ✓ Key saved to: %s\n", tempDir)

	// Load the key back
	loadedKey, err := crypto.LoadSenderKey(&crypto.SenderKeyConfig{
		Username:  "alice",
		ConfigDir: tempDir,
	})
	if err != nil {
		log.Fatalf("Failed to load key: %v", err)
	}
	fmt.Printf("  ✓ Key loaded for user: %s\n\n", loadedKey.Username)

	fmt.Println("=== Test Demonstration Complete ===")
	fmt.Println("In a real scenario, sender keys would be loaded from")
	fmt.Println("your Keybase installation at ~/.config/keybase/")
}
