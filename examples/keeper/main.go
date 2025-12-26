package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pulumi/pulumi-keybase-encryption/keybase"
)

func main() {
	fmt.Println("=== Keybase Keeper Example ===")
	fmt.Println()

	// Example 1: Create keeper from URL
	fmt.Println("Example 1: Creating Keeper from URL")
	fmt.Println("Note: This requires valid Keybase users. Uncomment to test with real users.")
	/*
	keeper, err := keybase.NewKeeperFromURL("keybase://alice,bob,charlie")
	if err != nil {
		log.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()

	// Encrypt a secret
	ctx := context.Background()
	plaintext := []byte("This is a secret message")
	
	fmt.Println("  Encrypting message for alice, bob, and charlie...")
	ciphertext, err := keeper.Encrypt(ctx, plaintext)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	
	fmt.Printf("  Encrypted message length: %d bytes\n", len(ciphertext))
	fmt.Printf("  Ciphertext preview: %s...\n", string(ciphertext[:min(50, len(ciphertext))]))
	
	// Any of the recipients can decrypt
	fmt.Println("\n  Decrypting message...")
	decrypted, err := keeper.Decrypt(ctx, ciphertext)
	if err != nil {
		log.Fatalf("Decryption failed: %v", err)
	}
	
	fmt.Printf("  Decrypted: %s\n", string(decrypted))
	*/

	// Example 2: Create keeper from config
	fmt.Println("\nExample 2: Creating Keeper from Config")
	config := &keybase.Config{
		Recipients: []string{"alice", "bob", "charlie"},
		Format:     keybase.FormatSaltpack,
		CacheTTL:   24 * 3600, // 24 hours in seconds
	}
	
	fmt.Printf("  Recipients: %v\n", config.Recipients)
	fmt.Printf("  Format: %s\n", config.Format)
	fmt.Printf("  Cache TTL: %v\n", config.CacheTTL)

	keeper, err := keybase.NewKeeper(&keybase.KeeperConfig{
		Config: config,
	})
	if err != nil {
		log.Fatalf("Failed to create keeper: %v", err)
	}
	defer keeper.Close()
	
	fmt.Println("  Keeper created successfully!")

	// Example 3: Demonstrate multiple recipient encryption
	fmt.Println("\nExample 3: Multiple Recipient Encryption")
	fmt.Println("  In this example, a secret is encrypted for multiple recipients.")
	fmt.Println("  Each recipient can decrypt the message independently with their private key.")
	fmt.Println("  The message is encrypted once, but each recipient gets their own")
	fmt.Println("  encrypted copy of the session key in the message header.")
	fmt.Println()
	fmt.Println("  Benefits:")
	fmt.Println("    • Efficient: Only one encryption operation")
	fmt.Println("    • Secure: No recipient enumeration attacks")
	fmt.Println("    • Flexible: Recipients can be added/removed")
	fmt.Println("    • Team-friendly: Perfect for shared secrets in teams")

	// Example 4: URL round-trip
	fmt.Println("\nExample 4: URL Configuration")
	url := config.ToURL()
	fmt.Printf("  Config as URL: %s\n", url)
	
	// Parse URL back to config
	parsedConfig, err := keybase.ParseURL(url)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	
	fmt.Printf("  Parsed recipients: %v\n", parsedConfig.Recipients)
	fmt.Printf("  Parsed format: %s\n", parsedConfig.Format)

	// Example 5: Error handling
	fmt.Println("\nExample 5: Error Handling")
	fmt.Println("  The Keeper provides detailed error information:")
	
	// Try to encrypt empty data (will fail)
	ctx := context.Background()
	_, err = keeper.Encrypt(ctx, []byte(""))
	if err != nil {
		fmt.Printf("  ✓ Empty plaintext rejected: %v\n", err)
	}
	
	// Try to decrypt invalid data (will fail)
	_, err = keeper.Decrypt(ctx, []byte("invalid ciphertext"))
	if err != nil {
		fmt.Printf("  ✓ Invalid ciphertext rejected: %v\n", err)
	}

	// Example 6: Integration with Pulumi
	fmt.Println("\nExample 6: Pulumi Integration")
	fmt.Println("  To use this keeper with Pulumi, add to your Pulumi.<stack>.yaml:")
	fmt.Println()
	fmt.Println("  config:")
	fmt.Println("    pulumi:secretsprovider: keybase://alice,bob,charlie")
	fmt.Println()
	fmt.Println("  Then set secrets:")
	fmt.Println("    $ pulumi config set myapp:apiKey \"secret-value\" --secret")
	fmt.Println()
	fmt.Println("  The secret will be encrypted for all specified recipients!")

	fmt.Println("\n=== Example Complete ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
