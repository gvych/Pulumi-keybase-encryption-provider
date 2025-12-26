package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	fmt.Println("=== Saltpack Encryption Example ===")
	fmt.Println()

	// Example 1: Generate key pairs for sender and recipients
	fmt.Println("Example 1: Generating key pairs")
	fmt.Println("  Generating sender key pair...")
	sender, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate sender key: %v", err)
	}
	fmt.Println("  ✓ Sender key pair generated")
	
	fmt.Println("  Generating recipient key pairs...")
	alice, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Alice's key: %v", err)
	}
	fmt.Println("  ✓ Alice's key pair generated")
	
	bob, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Bob's key: %v", err)
	}
	fmt.Println("  ✓ Bob's key pair generated")
	
	charlie, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Charlie's key: %v", err)
	}
	fmt.Println("  ✓ Charlie's key pair generated")
	fmt.Println()

	// Example 2: Create encryptor and decryptor
	fmt.Println("Example 2: Creating encryptor and decryptor")
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}
	fmt.Println("  ✓ Encryptor created")
	
	// Create keyring for Alice
	aliceKeyring := crypto.NewSimpleKeyring()
	aliceKeyring.AddKeyPair(alice)
	aliceKeyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification
	
	aliceDecryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: aliceKeyring,
	})
	if err != nil {
		log.Fatalf("Failed to create Alice's decryptor: %v", err)
	}
	fmt.Println("  ✓ Alice's decryptor created")
	fmt.Println()

	// Example 3: Single recipient encryption
	fmt.Println("Example 3: Single recipient encryption")
	message1 := []byte("Hello Alice! This is a secret message just for you.")
	fmt.Printf("  Original message: %s\n", string(message1))
	
	ciphertext1, err := encryptor.Encrypt(message1, []saltpack.BoxPublicKey{alice.PublicKey})
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	fmt.Printf("  ✓ Encrypted (%d bytes)\n", len(ciphertext1))
	
	decrypted1, info1, err := aliceDecryptor.Decrypt(ciphertext1)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}
	fmt.Printf("  ✓ Decrypted: %s\n", string(decrypted1))
	
	if info1 != nil {
		fmt.Printf("  ℹ Decryption successful\n")
	}
	fmt.Println()

	// Example 4: Multiple recipient encryption
	fmt.Println("Example 4: Multiple recipient encryption (team message)")
	teamMessage := []byte("Team update: Our Q4 metrics look great! 🎉")
	fmt.Printf("  Original message: %s\n", string(teamMessage))
	
	// Encrypt for all three recipients
	recipients := []saltpack.BoxPublicKey{
		alice.PublicKey,
		bob.PublicKey,
		charlie.PublicKey,
	}
	
	teamCiphertext, err := encryptor.Encrypt(teamMessage, recipients)
	if err != nil {
		log.Fatalf("Team encryption failed: %v", err)
	}
	fmt.Printf("  ✓ Encrypted for 3 recipients (%d bytes)\n", len(teamCiphertext))
	
	// Each recipient can decrypt
	fmt.Println("  Testing decryption by each recipient:")
	
	// Alice decrypts
	aliceDecrypted, _, err := aliceDecryptor.Decrypt(teamCiphertext)
	if err != nil {
		log.Fatalf("Alice's decryption failed: %v", err)
	}
	fmt.Printf("    ✓ Alice: %s\n", string(aliceDecrypted))
	
	// Bob decrypts
	bobKeyring := crypto.NewSimpleKeyring()
	bobKeyring.AddKeyPair(bob)
	bobKeyring.AddPublicKey(sender.PublicKey)
	bobDecryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: bobKeyring,
	})
	bobDecrypted, _, err := bobDecryptor.Decrypt(teamCiphertext)
	if err != nil {
		log.Fatalf("Bob's decryption failed: %v", err)
	}
	fmt.Printf("    ✓ Bob: %s\n", string(bobDecrypted))
	
	// Charlie decrypts
	charlieKeyring := crypto.NewSimpleKeyring()
	charlieKeyring.AddKeyPair(charlie)
	charlieKeyring.AddPublicKey(sender.PublicKey)
	charlieDecryptor, _ := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: charlieKeyring,
	})
	charlieDecrypted, _, err := charlieDecryptor.Decrypt(teamCiphertext)
	if err != nil {
		log.Fatalf("Charlie's decryption failed: %v", err)
	}
	fmt.Printf("    ✓ Charlie: %s\n", string(charlieDecrypted))
	fmt.Println()

	// Example 5: ASCII-armored encryption
	fmt.Println("Example 5: ASCII-armored encryption")
	armoredMessage := []byte("This message will be ASCII-armored for easy storage.")
	fmt.Printf("  Original message: %s\n", string(armoredMessage))
	
	armoredCiphertext, err := encryptor.EncryptArmored(armoredMessage, []saltpack.BoxPublicKey{alice.PublicKey})
	if err != nil {
		log.Fatalf("Armored encryption failed: %v", err)
	}
	fmt.Println("  ✓ Encrypted and armored")
	fmt.Printf("  Armored ciphertext (first 100 chars):\n  %s...\n", armoredCiphertext[:min(100, len(armoredCiphertext))])
	
	armoredDecrypted, _, err := aliceDecryptor.DecryptArmored(armoredCiphertext)
	if err != nil {
		log.Fatalf("Armored decryption failed: %v", err)
	}
	fmt.Printf("  ✓ Decrypted: %s\n", string(armoredDecrypted))
	fmt.Println()

	// Example 6: Streaming encryption (for large data)
	fmt.Println("Example 6: Streaming encryption")
	largeMessage := bytes.Repeat([]byte("This is a large message. "), 100)
	fmt.Printf("  Original message size: %d bytes\n", len(largeMessage))
	
	plaintextReader := bytes.NewReader(largeMessage)
	var ciphertextBuf bytes.Buffer
	
	err = encryptor.EncryptStream(plaintextReader, &ciphertextBuf, []saltpack.BoxPublicKey{alice.PublicKey})
	if err != nil {
		log.Fatalf("Stream encryption failed: %v", err)
	}
	fmt.Printf("  ✓ Encrypted via streaming (%d bytes)\n", ciphertextBuf.Len())
	
	ciphertextReader := bytes.NewReader(ciphertextBuf.Bytes())
	var decryptedBuf bytes.Buffer
	
	_, err = aliceDecryptor.DecryptStream(ciphertextReader, &decryptedBuf)
	if err != nil {
		log.Fatalf("Stream decryption failed: %v", err)
	}
	fmt.Printf("  ✓ Decrypted via streaming (%d bytes)\n", decryptedBuf.Len())
	
	if bytes.Equal(decryptedBuf.Bytes(), largeMessage) {
		fmt.Println("  ✓ Stream encryption/decryption successful!")
	}
	fmt.Println()

	// Example 7: Key validation
	fmt.Println("Example 7: Key validation")
	
	if err := crypto.ValidatePublicKey(alice.PublicKey); err != nil {
		fmt.Printf("  ✗ Alice's public key invalid: %v\n", err)
	} else {
		fmt.Println("  ✓ Alice's public key is valid")
	}
	
	if err := crypto.ValidateSecretKey(alice.SecretKey); err != nil {
		fmt.Printf("  ✗ Alice's secret key invalid: %v\n", err)
	} else {
		fmt.Println("  ✓ Alice's secret key is valid")
	}
	fmt.Println()

	// Example 8: Key comparison
	fmt.Println("Example 8: Key comparison")
	
	if crypto.KeysEqual(alice.PublicKey, alice.PublicKey) {
		fmt.Println("  ✓ Alice's key equals itself")
	}
	
	if !crypto.KeysEqual(alice.PublicKey, bob.PublicKey) {
		fmt.Println("  ✓ Alice's key != Bob's key")
	}
	fmt.Println()

	// Example 9: Error handling - decryption with wrong key
	fmt.Println("Example 9: Error handling - decryption with wrong key")
	secretMessage := []byte("This is only for Alice")
	secretCiphertext, _ := encryptor.Encrypt(secretMessage, []saltpack.BoxPublicKey{alice.PublicKey})
	
	// Try to decrypt with Bob's key (should fail)
	_, _, err = bobDecryptor.Decrypt(secretCiphertext)
	if err != nil {
		fmt.Printf("  ✓ Expected error: Bob cannot decrypt Alice's message\n")
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Println("  ✗ Bob should not be able to decrypt!")
	}
	fmt.Println()

	// Example 10: Performance comparison
	fmt.Println("Example 10: Performance comparison")
	testMessage := []byte("Performance test message")
	
	// Binary encryption
	binaryCiphertext, _ := encryptor.Encrypt(testMessage, []saltpack.BoxPublicKey{alice.PublicKey})
	fmt.Printf("  Binary ciphertext size: %d bytes\n", len(binaryCiphertext))
	
	// Armored encryption
	armoredCipher, _ := encryptor.EncryptArmored(testMessage, []saltpack.BoxPublicKey{alice.PublicKey})
	fmt.Printf("  Armored ciphertext size: %d bytes\n", len(armoredCipher))
	fmt.Printf("  Overhead: %.1f%%\n", float64(len(armoredCipher)-len(binaryCiphertext))/float64(len(binaryCiphertext))*100)
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Println("✓ Saltpack encryption supports:")
	fmt.Println("  • Single and multiple recipients")
	fmt.Println("  • Binary and ASCII-armored formats")
	fmt.Println("  • Streaming for large messages")
	fmt.Println("  • Modern cryptography (NaCl/libsodium)")
	fmt.Println("  • Recipient privacy (no visible recipient list)")
	fmt.Println()
	fmt.Println("Key advantages:")
	fmt.Println("  • Each recipient can decrypt independently")
	fmt.Println("  • Recipient list is encrypted (privacy)")
	fmt.Println("  • Efficient multi-recipient handling")
	fmt.Println("  • Compatible with Keybase ecosystem")
	fmt.Println()
	fmt.Println("=== Example Complete ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
