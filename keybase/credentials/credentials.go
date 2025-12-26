package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Status represents the Keybase credential status
type Status struct {
	// IsInstalled indicates if Keybase CLI is installed
	IsInstalled bool
	
	// IsLoggedIn indicates if a user is logged in
	IsLoggedIn bool
	
	// Username is the logged-in username (empty if not logged in)
	Username string
	
	// ConfigDir is the Keybase configuration directory
	ConfigDir string
	
	// CLIPath is the path to the Keybase CLI binary
	CLIPath string
	
	// Error contains any error encountered during discovery
	Error error
}

// DiscoverCredentials detects Keybase CLI installation and login status
// It performs the following checks:
// 1. Detects if Keybase CLI is installed and in PATH
// 2. Verifies Keybase user is logged in
// 3. Reads authentication status from Keybase directory
// 4. Returns detailed status with clear errors if Keybase is not available
func DiscoverCredentials() (*Status, error) {
	status := &Status{}
	
	// Step 1: Check if Keybase CLI is installed
	cliPath, err := findKeybaseCLI()
	if err != nil {
		status.Error = fmt.Errorf("Keybase CLI not found: %w", err)
		return status, status.Error
	}
	
	status.IsInstalled = true
	status.CLIPath = cliPath
	
	// Step 2: Find Keybase config directory
	configDir, err := getKeybaseConfigDir()
	if err != nil {
		status.Error = fmt.Errorf("Keybase config directory not found: %w", err)
		return status, status.Error
	}
	
	status.ConfigDir = configDir
	
	// Step 3: Check if user is logged in
	username, err := getLoggedInUser(configDir)
	if err != nil {
		status.Error = fmt.Errorf("no Keybase user logged in: %w", err)
		return status, status.Error
	}
	
	status.IsLoggedIn = true
	status.Username = username
	
	return status, nil
}

// findKeybaseCLI attempts to locate the Keybase CLI binary
func findKeybaseCLI() (string, error) {
	// Try to find keybase in PATH
	path, err := exec.LookPath("keybase")
	if err != nil {
		return "", fmt.Errorf("keybase command not found in PATH: %w", err)
	}
	
	return path, nil
}

// getKeybaseConfigDir returns the Keybase configuration directory
func getKeybaseConfigDir() (string, error) {
	var configDir string
	
	// Determine config directory based on OS
	switch runtime.GOOS {
	case "linux", "darwin": // Linux and macOS
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config", "keybase")
		
	case "windows":
		// Windows uses a different location
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "Keybase")
		
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	
	// Verify directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return "", fmt.Errorf("Keybase config directory does not exist at %s: ensure Keybase is installed and has been run at least once", configDir)
	}
	
	return configDir, nil
}

// getLoggedInUser determines the currently logged-in Keybase user
func getLoggedInUser(configDir string) (string, error) {
	// Check for config.json which contains current user info
	configFile := filepath.Join(configDir, "config.json")
	
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("config.json not found: Keybase may not be configured or no user is logged in")
		}
		return "", fmt.Errorf("failed to read config.json: %w", err)
	}
	
	// Parse config.json to extract username
	var config struct {
		CurrentUser string `json:"current_user"`
		Username    string `json:"username"` // Alternative field name
	}
	
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse config.json: %w", err)
	}
	
	// Try current_user first, then username
	username := config.CurrentUser
	if username == "" {
		username = config.Username
	}
	
	if username == "" {
		return "", fmt.Errorf("no logged-in user found in config.json: please run 'keybase login'")
	}
	
	return username, nil
}

// VerifyKeybaseAvailable is a convenience function that checks if Keybase is available
// and returns a user-friendly error if not
func VerifyKeybaseAvailable() error {
	status, err := DiscoverCredentials()
	if err != nil {
		return err
	}
	
	if !status.IsInstalled {
		return fmt.Errorf("Keybase is not installed. Please install Keybase from https://keybase.io/download")
	}
	
	if !status.IsLoggedIn {
		return fmt.Errorf("no Keybase user is logged in. Please run 'keybase login' to authenticate")
	}
	
	return nil
}

// GetUsername returns the currently logged-in Keybase username
// Returns an error if Keybase is not available or no user is logged in
func GetUsername() (string, error) {
	status, err := DiscoverCredentials()
	if err != nil {
		return "", err
	}
	
	if !status.IsLoggedIn {
		return "", fmt.Errorf("no Keybase user logged in")
	}
	
	return status.Username, nil
}

// IsKeybaseAvailable returns true if Keybase is installed and a user is logged in
func IsKeybaseAvailable() bool {
	status, err := DiscoverCredentials()
	if err != nil {
		return false
	}
	
	return status.IsInstalled && status.IsLoggedIn
}
