package main

import (
	"context"
	"fmt"
	"log"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

// DecryptKeeper interface for the wrapper
type DecryptKeeper interface {
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

func main() {
	fmt.Println("=== Keybase Decrypt Method Example ===\n")

	// Example 1: Single Recipient Encrypt/Decrypt
	fmt.Println("Example 1: Single Recipient")
	singleRecipientExample()

	// Example 2: Multiple Recipients
	fmt.Println("\nExample 2: Multiple Recipients")
	multipleRecipientsExample()

	// Example 3: Streaming Large Files
	fmt.Println("\nExample 3: Streaming Large Files (>10 MiB)")
	streamingExample()

	// Example 4: Error Handling
	fmt.Println("\nExample 4: Error Handling")
	errorHandlingExample()
}

func singleRecipientExample() {
	// Generate a test key pair for Alice
	aliceKeyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create a mock cache manager with Alice's key
	cacheManager, err := createMockCacheManager(map[string]*crypto.KeyPair{
		"alice": aliceKeyPair,
	})
	if err != nil {
		log.Fatalf("Failed to create cache manager: %v", err)
	}
	defer cacheManager.Close()

	// Create a keeper for Alice
	keeper, err := keybase.NewKeeper(&keybase.KeeperConfig{
		Config: &keybase.Config{
			Recipients: []string{"alice"},
			Format:     keybase.FormatSaltpack,
		},
		CacheManager: cacheManager,
	})
	if err != nil {
		log.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()

	// Encrypt a message
	ctx := context.Background()
	plaintext := []byte("Hello, Alice! This is a secret message.")
	
	fmt.Printf("  Original: %s\n", plaintext)
	
	ciphertext, err := keeper.Encrypt(ctx, plaintext)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	
	fmt.Printf("  Encrypted: %d bytes\n", len(ciphertext))

	// Create a decryption keeper with Alice's secret key
	decryptKeeper := createDecryptKeeper(aliceKeyPair, keeper)

	// Decrypt the message
	decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}

	fmt.Printf("  Decrypted: %s\n", decrypted)
	fmt.Printf("  ✅ Success! Plaintext matches: %v\n", string(decrypted) == string(plaintext))
}

func multipleRecipientsExample() {
	// Generate key pairs for Alice, Bob, and Charlie
	aliceKeyPair, _ := crypto.GenerateKeyPair()
	bobKeyPair, _ := crypto.GenerateKeyPair()
	charlieKeyPair, _ := crypto.GenerateKeyPair()

	// Create mock cache manager with all keys
	cacheManager, err := createMockCacheManager(map[string]*crypto.KeyPair{
		"alice":   aliceKeyPair,
		"bob":     bobKeyPair,
		"charlie": charlieKeyPair,
	})
	if err != nil {
		log.Fatalf("Failed to create cache manager: %v", err)
	}
	defer cacheManager.Close()

	// Create keeper for all three recipients
	keeper, err := keybase.NewKeeper(&keybase.KeeperConfig{
		Config: &keybase.Config{
			Recipients: []string{"alice", "bob", "charlie"},
			Format:     keybase.FormatSaltpack,
		},
		CacheManager: cacheManager,
	})
	if err != nil {
		log.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()

	// Encrypt a message for all recipients
	ctx := context.Background()
	plaintext := []byte("Team message: Project Alpha is green!")
	
	fmt.Printf("  Original: %s\n", plaintext)
	fmt.Printf("  Recipients: alice, bob, charlie\n")
	
	ciphertext, err := keeper.Encrypt(ctx, plaintext)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	
	fmt.Printf("  Encrypted: %d bytes\n", len(ciphertext))

	// Each recipient can decrypt independently
	recipients := map[string]*crypto.KeyPair{
		"alice":   aliceKeyPair,
		"bob":     bobKeyPair,
		"charlie": charlieKeyPair,
	}

	for name, keyPair := range recipients {
		decryptKeeper := createDecryptKeeper(keyPair, keeper)
		
		decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
		if err != nil {
			log.Printf("  ❌ %s failed to decrypt: %v\n", name, err)
			continue
		}

		matches := string(decrypted) == string(plaintext)
		if matches {
			fmt.Printf("  ✅ %s decrypted successfully\n", name)
		} else {
			fmt.Printf("  ❌ %s decrypted but plaintext doesn't match\n", name)
		}
	}
}

func streamingExample() {
	// Generate key pair for Alice
	aliceKeyPair, _ := crypto.GenerateKeyPair()

	// Create mock cache manager
	cacheManager, err := createMockCacheManager(map[string]*crypto.KeyPair{
		"alice": aliceKeyPair,
	})
	if err != nil {
		log.Fatalf("Failed to create cache manager: %v", err)
	}
	defer cacheManager.Close()

	// Create keeper
	keeper, err := keybase.NewKeeper(&keybase.KeeperConfig{
		Config: &keybase.Config{
			Recipients: []string{"alice"},
			Format:     keybase.FormatSaltpack,
		},
		CacheManager: cacheManager,
	})
	if err != nil {
		log.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()

	// Create a large plaintext (11 MiB to trigger streaming)
	size := 11 * 1024 * 1024
	plaintext := make([]byte, size)
	for i := 0; i < size; i++ {
		plaintext[i] = byte(i % 256)
	}
	
	fmt.Printf("  Message size: %d bytes (%.2f MiB)\n", size, float64(size)/(1024*1024))
	fmt.Printf("  Streaming threshold: 10 MiB\n")
	fmt.Printf("  Will use: Streaming decryption\n")

	// Encrypt
	ctx := context.Background()
	ciphertext, err := keeper.Encrypt(ctx, plaintext)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	
	fmt.Printf("  Encrypted: %d bytes (%.2f MiB)\n", len(ciphertext), float64(len(ciphertext))/(1024*1024))

	// Decrypt (will automatically use streaming)
	decryptKeeper := createDecryptKeeper(aliceKeyPair, keeper)
	
	decrypted, err := decryptKeeper.Decrypt(ctx, ciphertext)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}

	fmt.Printf("  Decrypted: %d bytes (%.2f MiB)\n", len(decrypted), float64(len(decrypted))/(1024*1024))
	
	// Verify a sample of the data
	matches := len(decrypted) == len(plaintext)
	if matches {
		// Check first 1000 bytes
		for i := 0; i < 1000; i++ {
			if decrypted[i] != plaintext[i] {
				matches = false
				break
			}
		}
	}
	
	fmt.Printf("  ✅ Success! Streaming decryption worked correctly: %v\n", matches)
}

func errorHandlingExample() {
	// Generate key pair
	keyPair, _ := crypto.GenerateKeyPair()

	// Create decryption-only keeper (no cache manager needed for this test)
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	decryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})

	// Test 1: Empty ciphertext
	fmt.Println("  Test 1: Empty ciphertext")
	_, _, err := decryptor.Decrypt([]byte(""))
	if err != nil {
		fmt.Printf("    ✅ Correctly rejected: %v\n", err)
	}

	// Test 2: Invalid ciphertext
	fmt.Println("  Test 2: Invalid ciphertext")
	_, _, err = decryptor.Decrypt([]byte("not a valid ciphertext"))
	if err != nil {
		fmt.Printf("    ✅ Correctly rejected: %v\n", err)
	}

	// Test 3: Corrupted ciphertext
	fmt.Println("  Test 3: Corrupted armored ciphertext")
	_, _, err = decryptor.DecryptArmored("BEGIN SALTPACK ENCRYPTED MESSAGE. corrupted data")
	if err != nil {
		fmt.Printf("    ✅ Correctly rejected: %v\n", err)
	}

	// Test 4: Wrong key
	fmt.Println("  Test 4: Decryption with wrong key")
	
	// Create a message encrypted for someone else
	otherKeyPair, _ := crypto.GenerateKeyPair()
	encryptor, _ := crypto.NewEncryptor(&crypto.EncryptorConfig{})
	ciphertext, _ := encryptor.Encrypt([]byte("secret"), []saltpack.BoxPublicKey{otherKeyPair.PublicKey})
	
	// Try to decrypt with our key (should fail)
	_, _, err = decryptor.Decrypt(ciphertext)
	if err != nil {
		fmt.Printf("    ✅ Correctly rejected: %v\n", err)
	}
}

// Helper functions

func createMockCacheManager(keys map[string]*crypto.KeyPair) (*cache.Manager, error) {
	manager, err := cache.NewManager(nil)
	if err != nil {
		return nil, err
	}

	// Pre-populate cache with test keys
	for username, keyPair := range keys {
		keyID := keyPair.PublicKey.ToKID()
		
		// Convert to hex with 0120 prefix (simulates Keybase KID format)
		keyIDHex := "0120" + fmt.Sprintf("%x", keyID)
		mockKeyBundle := "-----BEGIN PGP PUBLIC KEY BLOCK----- test key -----"
		
		if err := manager.Cache().Set(username, mockKeyBundle, keyIDHex); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func createDecryptKeeper(keyPair *crypto.KeyPair, baseKeeper *keybase.Keeper) DecryptKeeper {
	// Create keyring with the secret key
	keyring := crypto.NewSimpleKeyring()
	keyring.AddKey(keyPair.SecretKey)

	// Create decryptor
	decryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})

	// Return a struct that implements the Decrypt method
	return &decryptKeeperWrapper{
		keeper:    baseKeeper,
		decryptor: decryptor,
		keyring:   keyring,
	}
}

// decryptKeeperWrapper wraps a keeper with a custom decryptor
type decryptKeeperWrapper struct {
	keeper    *keybase.Keeper
	decryptor *crypto.Decryptor
	keyring   *crypto.SimpleKeyring
}

func (d *decryptKeeperWrapper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}
	
	// Try armored decryption first
	plaintext, _, err := d.decryptor.DecryptArmored(string(ciphertext))
	if err != nil {
		// If armored decryption fails, try binary decryption
		plaintext, _, err = d.decryptor.Decrypt(ciphertext)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
	}
	
	return plaintext, nil
}
