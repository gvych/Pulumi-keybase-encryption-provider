# Keybase Credentials Package

This package provides credential discovery functionality for the Pulumi Keybase encryption provider. It detects whether Keybase CLI is installed, verifies login status, and provides clear error messages when Keybase is not available.

## Features

- **Keybase CLI Detection**: Automatically detects if the Keybase CLI is installed and available in PATH
- **Login Status Verification**: Checks if a Keybase user is currently logged in
- **Cross-Platform Support**: Works on Linux, macOS, and Windows
- **Clear Error Messages**: Provides user-friendly error messages when Keybase is not configured
- **Configuration Directory Discovery**: Automatically locates the Keybase configuration directory

## Usage

### Basic Credential Discovery

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
)

func main() {
	// Discover Keybase credentials
	status, err := credentials.DiscoverCredentials()
	if err != nil {
		log.Fatalf("Keybase not available: %v", err)
	}
	
	fmt.Printf("Keybase CLI: %s\n", status.CLIPath)
	fmt.Printf("Config Dir: %s\n", status.ConfigDir)
	fmt.Printf("Logged in as: %s\n", status.Username)
}
```

### Quick Availability Check

```go
// Simple boolean check
if !credentials.IsKeybaseAvailable() {
	log.Fatal("Keybase is not available")
}

// Or get detailed error message
if err := credentials.VerifyKeybaseAvailable(); err != nil {
	log.Fatalf("Keybase verification failed: %v", err)
}
```

### Get Current Username

```go
username, err := credentials.GetUsername()
if err != nil {
	log.Fatalf("Failed to get username: %v", err)
}

fmt.Printf("Current user: %s\n", username)
```

## API Reference

### `DiscoverCredentials() (*Status, error)`

Performs comprehensive credential discovery and returns detailed status information.

**Returns:**
- `*Status`: Detailed status information about Keybase installation and login
- `error`: Error if Keybase is not available or not properly configured

**Status Fields:**
- `IsInstalled` (bool): True if Keybase CLI is installed
- `IsLoggedIn` (bool): True if a user is logged in
- `Username` (string): The logged-in username (empty if not logged in)
- `ConfigDir` (string): Path to Keybase configuration directory
- `CLIPath` (string): Path to Keybase CLI binary
- `Error` (error): Any error encountered during discovery

**Example:**

```go
status, err := credentials.DiscoverCredentials()
if err != nil {
	// Handle error - Keybase not available
	fmt.Printf("Error: %v\n", err)
	return
}

fmt.Printf("Keybase is installed: %v\n", status.IsInstalled)
fmt.Printf("User is logged in: %v\n", status.IsLoggedIn)
fmt.Printf("Username: %s\n", status.Username)
```

### `VerifyKeybaseAvailable() error`

Convenience function that checks if Keybase is available and returns a user-friendly error if not.

**Returns:**
- `error`: Descriptive error if Keybase is not available, nil otherwise

**Example:**

```go
if err := credentials.VerifyKeybaseAvailable(); err != nil {
	log.Fatal(err) // Will print: "Keybase is not installed..." or "no Keybase user is logged in..."
}
```

### `GetUsername() (string, error)`

Returns the currently logged-in Keybase username.

**Returns:**
- `string`: The username
- `error`: Error if Keybase is not available or no user is logged in

**Example:**

```go
username, err := credentials.GetUsername()
if err != nil {
	log.Fatalf("Failed to get username: %v", err)
}
fmt.Printf("Logged in as: %s\n", username)
```

### `IsKeybaseAvailable() bool`

Simple boolean check for Keybase availability.

**Returns:**
- `bool`: True if Keybase is installed and a user is logged in, false otherwise

**Example:**

```go
if credentials.IsKeybaseAvailable() {
	fmt.Println("Keybase is ready to use")
} else {
	fmt.Println("Keybase is not available")
}
```

## How It Works

### Detection Process

The credential discovery process performs the following checks:

1. **CLI Detection**: Uses `exec.LookPath()` to find the `keybase` binary in PATH
2. **Config Directory Discovery**: Locates the Keybase configuration directory based on OS:
   - Linux/macOS: `~/.config/keybase`
   - Windows: `%LOCALAPPDATA%\Keybase`
3. **Login Verification**: Reads `config.json` from the configuration directory to determine logged-in user

### Configuration File

The package reads the Keybase `config.json` file to determine login status. This file is automatically maintained by Keybase and contains the current user information:

```json
{
  "current_user": "alice",
  "username": "alice"
}
```

## Error Handling

The package provides clear, actionable error messages:

### Keybase Not Installed

```
Error: Keybase CLI not found: keybase command not found in PATH
```

**Solution**: Install Keybase from https://keybase.io/download

### Not Logged In

```
Error: no Keybase user logged in: no logged-in user found in config.json: please run 'keybase login'
```

**Solution**: Run `keybase login` to authenticate

### Config Directory Not Found

```
Error: Keybase config directory not found: Keybase config directory does not exist at ~/.config/keybase: ensure Keybase is installed and has been run at least once
```

**Solution**: Run Keybase at least once to initialize the configuration directory

## Platform-Specific Behavior

### Linux and macOS

- Configuration directory: `~/.config/keybase`
- CLI binary typically located in `/usr/local/bin/keybase` or `/usr/bin/keybase`

### Windows

- Configuration directory: `%LOCALAPPDATA%\Keybase`
- CLI binary typically located in `C:\Program Files\Keybase\keybase.exe`

## Integration with Pulumi Provider

This package is used by the Pulumi Keybase encryption provider to verify that Keybase is available before attempting encryption or decryption operations.

**Example Integration:**

```go
import (
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
)

func NewKeybaseProvider(config *Config) (*Provider, error) {
	// Verify Keybase is available
	if err := credentials.VerifyKeybaseAvailable(); err != nil {
		return nil, fmt.Errorf("Keybase provider requires Keybase to be installed and configured: %w", err)
	}
	
	// Get current username for sender identity
	username, err := credentials.GetUsername()
	if err != nil {
		return nil, err
	}
	
	// Initialize cache manager and other components
	manager, err := cache.NewManager(nil)
	if err != nil {
		return nil, err
	}
	
	return &Provider{
		username: username,
		manager:  manager,
	}, nil
}
```

## Testing

Run tests with:

```bash
go test -v ./keybase/credentials/...
```

Run tests with coverage:

```bash
go test -v -cover ./keybase/credentials/...
```

### Test Coverage

The test suite includes:

- CLI detection tests
- Config directory discovery tests
- Login status verification tests
- Error handling tests
- Cross-platform behavior tests
- Mock configuration tests

**Note**: Some tests may be skipped if Keybase is not installed on the test machine. This is expected behavior and allows the test suite to pass in CI environments without Keybase.

## Security Considerations

- **Read-Only Operations**: This package only reads Keybase configuration files and never modifies them
- **No Sensitive Data**: Only public configuration data is read (username, paths)
- **No CLI Execution**: The package checks for CLI presence but doesn't execute it, avoiding potential command injection risks
- **File Permissions**: Respects Keybase's file permission settings

## Limitations

- **Requires Local Keybase**: This package requires Keybase to be installed locally and does not support remote or cloud-based Keybase services
- **No Auto-Installation**: The package will not automatically install Keybase if it's not present
- **Single User**: Only detects the currently logged-in user, not multiple user configurations

## Future Enhancements

Potential future improvements:

- Support for Keybase service status checks
- Detection of Keybase daemon/service running status
- Support for multiple Keybase user profiles
- Integration with Keybase API for remote credential verification
- Caching of credential status to reduce filesystem checks

## Contributing

When contributing to this package:

1. Ensure all tests pass: `go test -v ./keybase/credentials/...`
2. Add tests for new functionality
3. Update this README with new features
4. Follow Go standard formatting: `go fmt`
5. Ensure cross-platform compatibility

## License

This package is part of the Pulumi Keybase encryption provider and follows Pulumi's licensing terms.

## Related Packages

- `keybase/api`: Keybase REST API client for public key lookup
- `keybase/cache`: Public key caching with TTL support
- `keybase/crypto`: Encryption/decryption operations (coming in Phase 2)

## Support

For issues and questions:

- GitHub Issues: Report bugs and feature requests
- Documentation: See main project README
- Community: Pulumi Community Slack
