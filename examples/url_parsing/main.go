package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-keybase-encryption/keybase"
)

func main() {
	fmt.Println("=== Keybase URL Parsing Examples ===")
	fmt.Println()

	// Example 1: Basic single recipient
	fmt.Println("Example 1: Basic single recipient")
	config1, err := keybase.ParseURL("keybase://alice")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Recipients: %v\n", config1.Recipients)
	fmt.Printf("Format: %s\n", config1.Format)
	fmt.Printf("Cache TTL: %s\n", config1.CacheTTL)
	fmt.Printf("Config: %s\n\n", config1.String())

	// Example 2: Multiple recipients
	fmt.Println("Example 2: Multiple recipients")
	config2, err := keybase.ParseURL("keybase://alice,bob,charlie")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Recipients: %v\n", config2.Recipients)
	fmt.Printf("Format: %s\n\n", config2.Format)

	// Example 3: With format parameter
	fmt.Println("Example 3: With PGP format")
	config3, err := keybase.ParseURL("keybase://alice,bob?format=pgp")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Recipients: %v\n", config3.Recipients)
	fmt.Printf("Format: %s\n\n", config3.Format)

	// Example 4: With custom cache TTL
	fmt.Println("Example 4: Custom cache TTL (1 hour)")
	config4, err := keybase.ParseURL("keybase://alice?cache_ttl=3600")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Recipients: %v\n", config4.Recipients)
	fmt.Printf("Cache TTL: %s\n\n", config4.CacheTTL)

	// Example 5: With verify proofs enabled
	fmt.Println("Example 5: With identity proof verification")
	config5, err := keybase.ParseURL("keybase://alice?verify_proofs=true")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Recipients: %v\n", config5.Recipients)
	fmt.Printf("Verify Proofs: %t\n\n", config5.VerifyProofs)

	// Example 6: All parameters
	fmt.Println("Example 6: All parameters specified")
	config6, err := keybase.ParseURL("keybase://alice,bob,charlie?format=pgp&cache_ttl=7200&verify_proofs=true")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Config: %s\n\n", config6.String())

	// Example 7: Round-trip (config -> URL -> config)
	fmt.Println("Example 7: Round-trip conversion")
	originalConfig := &keybase.Config{
		Recipients:   []string{"alice", "bob"},
		Format:       keybase.FormatSaltpack,
		CacheTTL:     12 * 60 * 60 * 1000000000, // 12 hours in nanoseconds
		VerifyProofs: true,
	}
	url := originalConfig.ToURL()
	fmt.Printf("Generated URL: %s\n", url)
	
	parsedConfig, err := keybase.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Parsed Config: %s\n\n", parsedConfig.String())

	// Example 8: Error handling - invalid format
	fmt.Println("Example 8: Error handling - invalid format")
	_, err = keybase.ParseURL("keybase://alice?format=aes")
	if err != nil {
		fmt.Printf("Error (expected): %v\n\n", err)
	}

	// Example 9: Error handling - invalid username
	fmt.Println("Example 9: Error handling - invalid username")
	_, err = keybase.ParseURL("keybase://alice@example.com")
	if err != nil {
		fmt.Printf("Error (expected): %v\n\n", err)
	}

	// Example 10: Error handling - invalid scheme
	fmt.Println("Example 10: Error handling - invalid scheme")
	_, err = keybase.ParseURL("https://alice,bob")
	if err != nil {
		fmt.Printf("Error (expected): %v\n\n", err)
	}

	fmt.Println("=== All examples completed successfully ===")
}
