package main

import (
	"context"
	"fmt"
	"log"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	fmt.Println("=== Saltpack Message Header Parsing Example ===")
	fmt.Println()

	// Generate test keys
	fmt.Println("1. Generating test keys...")
	sender, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate sender key: %v", err)
	}
	fmt.Println("   ✓ Generated sender key")

	alice, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Alice's key: %v", err)
	}
	fmt.Println("   ✓ Generated Alice's key")

	bob, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Bob's key: %v", err)
	}
	fmt.Println("   ✓ Generated Bob's key")

	charlie, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate Charlie's key: %v", err)
	}
	fmt.Println("   ✓ Generated Charlie's key")

	// Create encryptor with sender key
	fmt.Println("\n2. Creating encryptor with identified sender...")
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: sender.SecretKey,
	})
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}
	fmt.Println("   ✓ Encryptor created")

	// Encrypt message for multiple recipients
	plaintext := []byte("Secret message for the team!")
	fmt.Printf("\n3. Encrypting message for Alice, Bob, and Charlie...\n")
	fmt.Printf("   Plaintext: %s\n", plaintext)

	ciphertext, err := encryptor.EncryptArmored(plaintext, []saltpack.BoxPublicKey{
		alice.PublicKey,
		bob.PublicKey,
		charlie.PublicKey,
	})
	if err != nil {
		log.Fatalf("Failed to encrypt: %v", err)
	}
	fmt.Printf("   ✓ Encrypted (%d characters of armored ciphertext)\n", len(ciphertext))

	// Bob decrypts the message
	fmt.Println("\n4. Bob decrypting the message...")
	bobKeyring := crypto.NewSimpleKeyring()
	bobKeyring.AddKeyPair(bob)
	bobKeyring.AddPublicKey(sender.PublicKey) // Add sender's public key for verification

	bobDecryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: bobKeyring,
	})
	if err != nil {
		log.Fatalf("Failed to create Bob's decryptor: %v", err)
	}

	decrypted, messageKeyInfo, err := bobDecryptor.DecryptArmored(ciphertext)
	if err != nil {
		log.Fatalf("Bob failed to decrypt: %v", err)
	}
	fmt.Printf("   ✓ Decrypted: %s\n", decrypted)

	// Parse message header information
	fmt.Println("\n5. Parsing message header information...")
	messageInfo, err := crypto.ParseMessageKeyInfo(messageKeyInfo)
	if err != nil {
		log.Fatalf("Failed to parse message info: %v", err)
	}

	fmt.Printf("   Message Header Details:\n")
	fmt.Printf("   - Receiver KID: %s\n", crypto.FormatKeyID(messageInfo.ReceiverKID))
	if messageInfo.IsAnonymousSender {
		fmt.Printf("   - Sender: Anonymous\n")
	} else {
		fmt.Printf("   - Sender KID: %s\n", crypto.FormatKeyID(messageInfo.SenderKID))
	}

	// Verify the receiver
	fmt.Println("\n6. Verifying recipient identity...")
	if crypto.VerifyReceiver(messageKeyInfo, bob.PublicKey) {
		fmt.Println("   ✓ Confirmed: Bob decrypted this message")
	} else {
		fmt.Println("   ✗ Error: Receiver verification failed")
	}

	if crypto.VerifyReceiver(messageKeyInfo, alice.PublicKey) {
		fmt.Println("   ✗ Error: Alice should not be the decryptor")
	} else {
		fmt.Println("   ✓ Confirmed: Alice did not decrypt this message")
	}

	// Verify the sender
	fmt.Println("\n7. Verifying sender identity...")
	if crypto.VerifySender(messageKeyInfo, sender.PublicKey) {
		fmt.Println("   ✓ Confirmed: Message is from the expected sender")
	} else {
		fmt.Println("   ✗ Error: Sender verification failed")
	}

	// Test anonymous sender
	fmt.Println("\n8. Testing anonymous sender...")
	anonEncryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: nil, // Anonymous
	})
	if err != nil {
		log.Fatalf("Failed to create anonymous encryptor: %v", err)
	}

	anonCiphertext, err := anonEncryptor.EncryptArmored([]byte("Anonymous message"), []saltpack.BoxPublicKey{
		alice.PublicKey,
	})
	if err != nil {
		log.Fatalf("Failed to encrypt anonymously: %v", err)
	}

	aliceKeyring := crypto.NewSimpleKeyring()
	aliceKeyring.AddKeyPair(alice)

	aliceDecryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: aliceKeyring,
	})
	if err != nil {
		log.Fatalf("Failed to create Alice's decryptor: %v", err)
	}

	_, anonMessageKeyInfo, err := aliceDecryptor.DecryptArmored(anonCiphertext)
	if err != nil {
		log.Fatalf("Alice failed to decrypt: %v", err)
	}

	if crypto.IsAnonymousSender(anonMessageKeyInfo) {
		fmt.Println("   ✓ Confirmed: Sender is anonymous")
	} else {
		fmt.Println("   ✗ Error: Should be anonymous sender")
	}

	anonSenderKID := crypto.GetSenderKeyID(anonMessageKeyInfo)
	if anonSenderKID == nil {
		fmt.Println("   ✓ Confirmed: No sender key ID for anonymous message")
	} else {
		fmt.Println("   ✗ Error: Anonymous message should not have sender KID")
	}

	// Demonstrate high-level Keeper API
	fmt.Println("\n9. Using Keeper API with message info...")
	fmt.Println("   (Note: This example uses in-memory keys, not actual Keybase accounts)")

	// Create a mock cache manager for demonstration
	mockKeys := map[string]saltpack.BoxPublicKey{
		"alice": alice.PublicKey,
		"bob":   bob.PublicKey,
	}

	// In a real scenario, you would use:
	// keeper, err := keybase.NewKeeperFromURL("keybase://alice,bob")
	// plaintext, msgInfo, err := keeper.DecryptWithInfo(ctx, ciphertext)

	fmt.Println("   ✓ In production, use:")
	fmt.Println("     keeper, err := keybase.NewKeeperFromURL(\"keybase://alice,bob\")")
	fmt.Println("     plaintext, msgInfo, err := keeper.DecryptWithInfo(ctx, ciphertext)")

	_ = context.Background()
	_ = mockKeys

	fmt.Println()
	fmt.Println("=== Example Complete ===")
	fmt.Println()
	fmt.Println("Key Takeaways:")
	fmt.Println("• ParseMessageKeyInfo() extracts sender and receiver information")
	fmt.Println("• VerifyReceiver() confirms which recipient decrypted the message")
	fmt.Println("• VerifySender() validates the sender's identity (if not anonymous)")
	fmt.Println("• IsAnonymousSender() detects anonymous messages")
	fmt.Println("• Keeper.DecryptWithInfo() provides a high-level API for applications")
}
