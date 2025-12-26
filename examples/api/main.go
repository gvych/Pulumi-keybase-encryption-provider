package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
)

func main() {
	fmt.Println("=== Keybase API Client Example ===")
	fmt.Println()

	// Example 1: Create client with default configuration
	fmt.Println("Example 1: Creating API client with default configuration")
	client := api.NewClient(nil)
	fmt.Printf("  Base URL: %s\n", client.BaseURL)
	fmt.Printf("  Max Retries: %d\n", client.MaxRetries)
	fmt.Printf("  Retry Delay: %v\n\n", client.RetryDelay)

	// Example 2: Create client with custom configuration
	fmt.Println("Example 2: Creating API client with custom configuration")
	customConfig := &api.ClientConfig{
		BaseURL:    api.DefaultAPIEndpoint,
		Timeout:    15 * time.Second,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	}
	customClient := api.NewClient(customConfig)
	fmt.Printf("  Timeout: %v\n", customConfig.Timeout)
	fmt.Printf("  Max Retries: %d\n", customClient.MaxRetries)
	fmt.Printf("  Retry Delay: %v\n\n", customClient.RetryDelay)

	// Example 3: Username validation
	fmt.Println("Example 3: Username validation")
	validUsernames := []string{"alice", "bob_123", "Charlie"}
	for _, username := range validUsernames {
		if err := api.ValidateUsername(username); err != nil {
			fmt.Printf("  ✗ %s: %v\n", username, err)
		} else {
			fmt.Printf("  ✓ %s: valid\n", username)
		}
	}

	invalidUsernames := []string{"alice-bob", "alice@bob", "alice bob", ""}
	for _, username := range invalidUsernames {
		if err := api.ValidateUsername(username); err != nil {
			fmt.Printf("  ✗ %s: %v\n", username, err)
		} else {
			fmt.Printf("  ✓ %s: valid\n", username)
		}
	}
	fmt.Println()

	// Example 4: Fetching public keys (commented out by default)
	fmt.Println("Example 4: Fetching public keys from Keybase API")
	fmt.Println("  Note: Uncomment the code below to test with real Keybase users")
	fmt.Println("  Replace 'chris' with an actual Keybase username")
	fmt.Println()
	
	/*
	// Add "log" to imports when uncommenting
	ctx := context.Background()
	
	// Fetch single user
	keys, err := client.LookupUsers(ctx, []string{"chris"})
	if err != nil {
		log.Printf("  Error fetching user: %v\n", err)
	} else {
		fmt.Printf("  Fetched public key for user: %s\n", keys[0].Username)
		fmt.Printf("    Key ID: %s\n", keys[0].KeyID)
		fmt.Printf("    Public Key (first 100 chars): %s...\n", keys[0].PublicKey[:100])
	}
	fmt.Println()
	*/

	// Example 5: Batch user lookup (commented out by default)
	fmt.Println("Example 5: Batch user lookup")
	fmt.Println("  Note: Uncomment to test fetching multiple users in a single API call")
	fmt.Println("  This is more efficient than making separate requests")
	fmt.Println()
	
	/*
	// Add "log" to imports when uncommenting
	ctx := context.Background()
	usernames := []string{"chris", "max", "malgorithms"}
	
	keys, err := client.LookupUsers(ctx, usernames)
	if err != nil {
		log.Printf("  Error fetching users: %v\n", err)
	} else {
		fmt.Printf("  Fetched %d public keys:\n", len(keys))
		for _, key := range keys {
			fmt.Printf("    - %s (Key ID: %s)\n", key.Username, key.KeyID[:20]+"...")
		}
	}
	fmt.Println()
	*/

	// Example 6: Error handling
	fmt.Println("Example 6: Error handling")
	fmt.Println("  Testing with invalid username (should fail validation):")
	ctx := context.Background()
	_, err := client.LookupUsers(ctx, []string{"invalid@username"})
	if err != nil {
		fmt.Printf("  ✓ Expected error: %v\n", err)
	}
	
	fmt.Println("\n  Testing with empty username list (should fail):")
	_, err = client.LookupUsers(ctx, []string{})
	if err != nil {
		fmt.Printf("  ✓ Expected error: %v\n", err)
	}
	fmt.Println()

	// Example 7: Context with timeout
	fmt.Println("Example 7: Using context with timeout")
	fmt.Println("  Note: Uncomment to test timeout behavior")
	fmt.Println()
	
	/*
	// Create context with very short timeout to demonstrate timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	_, err := client.LookupUsers(ctx, []string{"chris"})
	if err != nil {
		fmt.Printf("  Expected timeout error: %v\n", err)
	}
	fmt.Println()
	*/

	// Example 8: API Error types
	fmt.Println("Example 8: API Error types")
	fmt.Println("  The client provides detailed error information:")
	fmt.Println("    - APIError with status code and temporary flag")
	fmt.Println("    - Automatic retry for temporary errors (5xx, 429)")
	fmt.Println("    - No retry for permanent errors (4xx except 429)")
	fmt.Println("    - Exponential backoff for retries")
	fmt.Println()

	// Example 9: Best practices
	fmt.Println("Example 9: Best practices")
	fmt.Println("  ✓ Batch multiple users in a single request")
	fmt.Println("  ✓ Use context for cancellation and timeouts")
	fmt.Println("  ✓ Validate usernames before API calls")
	fmt.Println("  ✓ Handle APIError types for detailed error information")
	fmt.Println("  ✓ Use caching layer (see cache package) to reduce API calls")
	fmt.Println("  ✓ Configure retries based on your reliability requirements")
	fmt.Println()

	fmt.Println("=== Example Complete ===")
	fmt.Println("\nTo test with real Keybase API:")
	fmt.Println("1. Uncomment the API call examples above")
	fmt.Println("2. Replace placeholder usernames with real Keybase users")
	fmt.Println("3. Run: go run main.go")
}
