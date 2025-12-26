package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

func main() {
	fmt.Println("=== Keybase Crypto - Ephemeral Key Generation Example ===\n")

	// Example 1: Generate a single ephemeral key pair
	fmt.Println("1. Generating a single ephemeral key pair...")
	creator := crypto.NewEphemeralKeyCreator()
	
	pair, err := creator.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}
	
	fmt.Printf("   Public key (first 8 bytes): %x...\n", pair.PublicKey.Bytes()[:8])
	fmt.Printf("   Secret key (first 8 bytes): %x...\n", pair.SecretKey.Bytes()[:8])
	
	// Clean up the secret key
	defer pair.Zero()
	fmt.Println("   ✓ Key pair generated successfully\n")

	// Example 2: Generate multiple key pairs
	fmt.Println("2. Generating 5 ephemeral key pairs...")
	pairs, err := creator.GenerateKeys(5)
	if err != nil {
		log.Fatalf("Failed to generate key pairs: %v", err)
	}
	
	for i, p := range pairs {
		fmt.Printf("   Key pair %d - Public: %x...\n", i+1, p.PublicKey.Bytes()[:8])
		defer p.Zero()
	}
	fmt.Println("   ✓ All key pairs generated successfully\n")

	// Example 3: Demonstrate key uniqueness
	fmt.Println("3. Verifying key uniqueness...")
	pair1, err := creator.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate first key: %v", err)
	}
	defer pair1.Zero()
	
	pair2, err := creator.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate second key: %v", err)
	}
	defer pair2.Zero()
	
	// Compare first few bytes to show they're different
	fmt.Printf("   Key 1 public: %x...\n", pair1.PublicKey.Bytes()[:8])
	fmt.Printf("   Key 2 public: %x...\n", pair2.PublicKey.Bytes()[:8])
	
	isUnique := false
	for i := range pair1.PublicKey {
		if pair1.PublicKey[i] != pair2.PublicKey[i] {
			isUnique = true
			break
		}
	}
	
	if isUnique {
		fmt.Println("   ✓ Keys are unique\n")
	} else {
		fmt.Println("   ✗ Warning: Keys are identical (very unlikely!)\n")
	}

	// Example 4: Demonstrate secure key zeroing
	fmt.Println("4. Demonstrating secure key cleanup...")
	tempPair, err := creator.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate temp key: %v", err)
	}
	
	fmt.Printf("   Before Zero() - Secret key (first 8 bytes): %x...\n", tempPair.SecretKey.Bytes()[:8])
	
	tempPair.Zero()
	
	allZeros := true
	for _, b := range tempPair.SecretKey {
		if b != 0 {
			allZeros = false
			break
		}
	}
	
	if allZeros {
		fmt.Println("   After Zero()  - Secret key: [all zeros]")
		fmt.Println("   ✓ Secret key successfully zeroed\n")
	} else {
		fmt.Println("   ✗ Warning: Secret key not completely zeroed\n")
	}

	fmt.Println("=== Example completed successfully ===")
}
