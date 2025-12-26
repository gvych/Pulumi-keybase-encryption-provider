package main

import (
	"fmt"
	"log"
	"time"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
)

func main() {
	fmt.Println("=== Keybase Keyring Loading Example ===")
	fmt.Println()

	// Step 1: Check if Keybase is available
	fmt.Println("Step 1: Checking Keybase availability...")
	if err := credentials.VerifyKeybaseAvailable(); err != nil {
		fmt.Printf("  ⚠️  Keybase not available: %v\n", err)
		fmt.Println()
		fmt.Println("This example requires Keybase to be installed and configured.")
		fmt.Println("Install from: https://keybase.io/download")
		fmt.Println()
		fmt.Println("Falling back to demo mode with generated keys...")
		runDemoMode()
		return
	}
	fmt.Println("  ✓ Keybase is available")
	fmt.Println()

	// Step 2: Get current username
	fmt.Println("Step 2: Getting current username...")
	username, err := credentials.GetUsername()
	if err != nil {
		log.Fatalf("  ✗ Failed to get username: %v", err)
	}
	fmt.Printf("  ✓ Current user: %s\n", username)
	fmt.Println()

	// Step 3: Create keyring loader with custom TTL
	fmt.Println("Step 3: Creating keyring loader...")
	loader, err := crypto.NewKeyringLoader(&crypto.KeyringLoaderConfig{
		TTL: 30 * time.Minute,
	})
	if err != nil {
		log.Fatalf("  ✗ Failed to create keyring loader: %v", err)
	}
	fmt.Println("  ✓ Keyring loader created (TTL: 30 minutes)")
	fmt.Println()

	// Step 4: Load keyring for current user
	fmt.Println("Step 4: Loading keyring for current user...")
	keyring, err := loader.LoadKeyring()
	if err != nil {
		log.Fatalf("  ✗ Failed to load keyring: %v", err)
	}
	fmt.Println("  ✓ Keyring loaded successfully")
	fmt.Println()

	// Step 5: Check cache statistics
	fmt.Println("Step 5: Cache statistics...")
	stats := loader.GetCacheStats()
	fmt.Printf("  Total cached keys: %d\n", stats.TotalCached)
	fmt.Printf("  Valid keys: %d\n", stats.ValidCount)
	fmt.Printf("  Expired keys: %d\n", stats.ExpiredCount)
	fmt.Printf("  Cache TTL: %v\n", stats.TTL)
	fmt.Println()

	// Step 6: Load sender key for encryption
	fmt.Println("Step 6: Loading sender key...")
	senderKey, err := crypto.LoadSenderKey(nil)
	if err != nil {
		log.Fatalf("  ✗ Failed to load sender key: %v", err)
	}
	fmt.Printf("  ✓ Sender key loaded for: %s\n", senderKey.Username)
	fmt.Println()

	// Step 7: Test encryption and decryption
	fmt.Println("Step 7: Testing encryption and decryption...")
	
	// Create encryptor
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: senderKey.SecretKey,
	})
	if err != nil {
		log.Fatalf("  ✗ Failed to create encryptor: %v", err)
	}

	// Encrypt a message (encrypt to ourselves for testing)
	plaintext := []byte("Hello from Keybase keyring example!")
	receivers := []saltpack.BoxPublicKey{senderKey.PublicKey}
	
	ciphertext, err := encryptor.EncryptArmored(plaintext, receivers)
	if err != nil {
		log.Fatalf("  ✗ Encryption failed: %v", err)
	}
	fmt.Println("  ✓ Message encrypted")

	// Create decryptor with loaded keyring
	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		log.Fatalf("  ✗ Failed to create decryptor: %v", err)
	}

	// Decrypt the message
	decrypted, info, err := decryptor.DecryptArmored(ciphertext)
	if err != nil {
		log.Fatalf("  ✗ Decryption failed: %v", err)
	}
	fmt.Println("  ✓ Message decrypted")
	fmt.Printf("  Original:  %s\n", string(plaintext))
	fmt.Printf("  Decrypted: %s\n", string(decrypted))
	
	if !info.SenderIsAnon {
		fmt.Printf("  Sender verified: %s\n", crypto.KeyIDToHex(info.SenderKey.ToKID()))
	}
	fmt.Println()

	// Step 8: Demonstrate cache reuse
	fmt.Println("Step 8: Demonstrating cache reuse...")
	start := time.Now()
	keyring2, err := loader.LoadKeyring()
	duration := time.Since(start)
	if err != nil {
		log.Fatalf("  ✗ Failed to load keyring (second time): %v", err)
	}
	fmt.Printf("  ✓ Keyring loaded from cache in %v\n", duration)
	
	if keyring2 != nil {
		fmt.Println("  ✓ Cache working correctly")
	}
	fmt.Println()

	// Step 9: Cache management
	fmt.Println("Step 9: Cache management...")
	
	// Get cached users
	cachedUsers := loader.GetCachedUsers()
	fmt.Printf("  Cached users: %v\n", cachedUsers)
	
	// Cleanup expired keys (shouldn't find any yet)
	removed := loader.CleanupExpiredKeys()
	fmt.Printf("  Removed %d expired keys\n", removed)
	
	// Demonstrate invalidation
	loader.InvalidateCacheForUser(username)
	fmt.Printf("  ✓ Invalidated cache for user: %s\n", username)
	
	stats = loader.GetCacheStats()
	fmt.Printf("  Cached keys after invalidation: %d\n", stats.TotalCached)
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
}

// runDemoMode runs the example in demo mode with generated keys
// This is used when Keybase is not available
func runDemoMode() {
	fmt.Println()
	fmt.Println("=== Running in Demo Mode ===")
	fmt.Println()

	// Generate test keys
	fmt.Println("Step 1: Generating test keys...")
	alice, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("  ✗ Failed to generate keys: %v", err)
	}
	fmt.Println("  ✓ Test keys generated")
	fmt.Println()

	// Create a keyring
	fmt.Println("Step 2: Creating keyring...")
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKeyPair(alice)
	fmt.Println("  ✓ Keyring created")
	fmt.Println()

	// Create encryptor and decryptor
	fmt.Println("Step 3: Setting up encryption...")
	encryptor, _ := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: alice.SecretKey,
	})
	
	decryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	fmt.Println("  ✓ Encryptor and decryptor ready")
	fmt.Println()

	// Encrypt and decrypt
	fmt.Println("Step 4: Testing encryption/decryption...")
	plaintext := []byte("Demo message with generated keys")
	receivers := []saltpack.BoxPublicKey{alice.PublicKey}
	
	ciphertext, err := encryptor.EncryptArmored(plaintext, receivers)
	if err != nil {
		log.Fatalf("  ✗ Encryption failed: %v", err)
	}
	fmt.Println("  ✓ Message encrypted")

	decrypted, _, err := decryptor.DecryptArmored(ciphertext)
	if err != nil {
		log.Fatalf("  ✗ Decryption failed: %v", err)
	}
	fmt.Println("  ✓ Message decrypted")
	fmt.Printf("  Original:  %s\n", string(plaintext))
	fmt.Printf("  Decrypted: %s\n", string(decrypted))
	fmt.Println()

	fmt.Println("=== Demo completed successfully! ===")
	fmt.Println()
	fmt.Println("To use real Keybase keys:")
	fmt.Println("  1. Install Keybase: https://keybase.io/download")
	fmt.Println("  2. Run: keybase login")
	fmt.Println("  3. Run: keybase pgp gen (if you don't have keys)")
	fmt.Println("  4. Run this example again")
}
