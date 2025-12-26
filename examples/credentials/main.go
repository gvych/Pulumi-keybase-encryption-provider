package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
)

func main() {
	fmt.Println("=== Keybase Credentials Discovery Example ===")

	// Example 1: Full credential discovery
	fmt.Println("Example 1: Full Credential Discovery")
	status, err := credentials.DiscoverCredentials()
	if err != nil {
		fmt.Printf("  ❌ Credential discovery failed: %v\n", err)
		fmt.Println("  💡 Tip: Install Keybase from https://keybase.io/download")
		fmt.Println("  💡 Tip: Run 'keybase login' to authenticate")
		
		// Show partial status even on error
		if status != nil {
			fmt.Printf("  Partial Status:\n")
			fmt.Printf("    - Keybase Installed: %v\n", status.IsInstalled)
			fmt.Printf("    - User Logged In: %v\n", status.IsLoggedIn)
			if status.CLIPath != "" {
				fmt.Printf("    - CLI Path: %s\n", status.CLIPath)
			}
			if status.ConfigDir != "" {
				fmt.Printf("    - Config Directory: %s\n", status.ConfigDir)
			}
		}
	} else {
		fmt.Println("  ✅ Keybase is available and configured!")
		fmt.Printf("    - CLI Path: %s\n", status.CLIPath)
		fmt.Printf("    - Config Directory: %s\n", status.ConfigDir)
		fmt.Printf("    - Logged in as: %s\n", status.Username)
		fmt.Printf("    - Installation Status: Installed=%v, LoggedIn=%v\n", 
			status.IsInstalled, status.IsLoggedIn)
	}

	// Example 2: Quick availability check
	fmt.Println("\nExample 2: Quick Availability Check")
	if credentials.IsKeybaseAvailable() {
		fmt.Println("  ✅ Keybase is ready to use")
	} else {
		fmt.Println("  ❌ Keybase is not available")
	}

	// Example 3: Verify with detailed error
	fmt.Println("\nExample 3: Verify Keybase with Detailed Error")
	if err := credentials.VerifyKeybaseAvailable(); err != nil {
		fmt.Printf("  ❌ Verification failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Keybase verification successful")
	}

	// Example 4: Get current username
	fmt.Println("\nExample 4: Get Current Username")
	username, err := credentials.GetUsername()
	if err != nil {
		fmt.Printf("  ❌ Failed to get username: %v\n", err)
	} else {
		fmt.Printf("  ✅ Current user: %s\n", username)
	}

	// Example 5: Use in application initialization
	fmt.Println("\nExample 5: Application Initialization Pattern")
	if err := initializeKeybaseProvider(); err != nil {
		fmt.Printf("  ❌ Provider initialization failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Keybase provider initialized successfully")
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\n📝 Note: If Keybase is not installed, some examples will show errors.")
	fmt.Println("   This is expected behavior demonstrating error handling.")
}

// initializeKeybaseProvider demonstrates how to use credential discovery
// in an application initialization function
func initializeKeybaseProvider() error {
	// Verify Keybase is available before proceeding
	if err := credentials.VerifyKeybaseAvailable(); err != nil {
		return fmt.Errorf("Keybase provider requires Keybase to be installed and configured: %w", err)
	}

	// Get current username for logging/auditing
	username, err := credentials.GetUsername()
	if err != nil {
		return fmt.Errorf("failed to get Keybase username: %w", err)
	}

	// Simulate provider initialization
	log.Printf("Initializing Keybase provider for user: %s", username)

	// In a real application, you would:
	// 1. Initialize the cache manager
	// 2. Set up encryption/decryption handlers
	// 3. Configure recipient lists
	// 4. etc.

	return nil
}
