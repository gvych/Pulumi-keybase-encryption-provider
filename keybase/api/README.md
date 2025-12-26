# Keybase API Client

This package provides a Go client for the Keybase REST API, specifically for fetching user public keys.

## Features

- **Batch user lookup**: Fetch multiple users in a single API call
- **Automatic retries**: Exponential backoff for transient errors
- **Context support**: Cancellable requests
- **Rate limiting handling**: Automatic retry on 429 responses
- **Username validation**: Client-side validation before API calls
- **Detailed errors**: Clear error messages with status codes

## Usage

### Creating a Client

```go
import "github.com/pulumi/pulumi-keybase-encryption/keybase/api"

// Use default configuration
client := api.NewClient(nil)

// Custom configuration
config := &api.ClientConfig{
    BaseURL:    "https://keybase.io/_/api/1.0",
    Timeout:    15 * time.Second,
    MaxRetries: 5,
    RetryDelay: 2 * time.Second,
}
client := api.NewClient(config)
```

### Fetching Public Keys

```go
ctx := context.Background()

// Fetch single user
keys, err := client.LookupUsers(ctx, []string{"alice"})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Username: %s\n", keys[0].Username)
fmt.Printf("Key ID: %s\n", keys[0].KeyID)
fmt.Printf("Public Key: %s\n", keys[0].PublicKey)

// Fetch multiple users (more efficient)
keys, err := client.LookupUsers(ctx, []string{"alice", "bob", "charlie"})
```

### Validating Usernames

```go
// Valid usernames
api.ValidateUsername("alice")        // OK
api.ValidateUsername("alice_bob")    // OK
api.ValidateUsername("alice123")     // OK

// Invalid usernames
api.ValidateUsername("alice-bob")    // Error: invalid character
api.ValidateUsername("alice@bob")    // Error: invalid character
api.ValidateUsername("")             // Error: empty username
```

## API Reference

### Types

#### `UserPublicKey`

```go
type UserPublicKey struct {
    Username  string // Keybase username
    PublicKey string // PGP public key bundle
    KeyID     string // Key identifier
}
```

#### `ClientConfig`

```go
type ClientConfig struct {
    BaseURL    string        // API endpoint
    Timeout    time.Duration // HTTP timeout
    MaxRetries int           // Max retry attempts
    RetryDelay time.Duration // Initial retry delay
}
```

### Functions

#### `NewClient(config *ClientConfig) *Client`

Creates a new Keybase API client. If `config` is nil, uses default configuration.

#### `LookupUsers(ctx context.Context, usernames []string) ([]UserPublicKey, error)`

Fetches public keys for multiple users from the Keybase API.

**Parameters:**
- `ctx`: Context for cancellation
- `usernames`: List of Keybase usernames

**Returns:**
- Slice of `UserPublicKey` in same order as input
- Error if any user not found or API error

#### `ValidateUsername(username string) error`

Validates a Keybase username format.

**Rules:**
- Alphanumeric characters (a-z, A-Z, 0-9)
- Underscores (_)
- Cannot be empty

## Error Handling

### APIError

```go
type APIError struct {
    Message    string
    StatusCode int
    Temporary  bool
}
```

#### Error Types

- **Network errors**: `Temporary=true`, can be retried
- **404 Not Found**: User doesn't exist
- **429 Rate Limit**: Temporary, automatically retried
- **500 Server Error**: Temporary, automatically retried
- **400 Bad Request**: Permanent, not retried

#### Checking Error Types

```go
if apiErr, ok := err.(*api.APIError); ok {
    if apiErr.StatusCode == 404 {
        fmt.Println("User not found")
    } else if apiErr.IsTemporary() {
        fmt.Println("Temporary error, try again")
    }
}
```

## Retry Behavior

The client automatically retries on:
- Network timeouts
- 5xx server errors
- 429 rate limiting

Retry strategy:
- Exponential backoff: 1s, 2s, 4s, 8s...
- Maximum retries: Configurable (default: 3)
- Non-retryable: 4xx errors (except 429)

## Context Cancellation

```go
// Request with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

keys, err := client.LookupUsers(ctx, []string{"alice"})
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("Request timed out")
    }
}
```

## API Endpoints

### User Lookup

**Endpoint:** `GET /user/lookup.json`

**Query Parameters:**
- `usernames`: Comma-separated list of usernames
- `fields`: Fields to include (always set to `public_keys`)

**Example:**
```
GET https://keybase.io/_/api/1.0/user/lookup.json?usernames=alice,bob&fields=public_keys
```

**Response:**
```json
{
  "status": {
    "code": 0,
    "name": "OK"
  },
  "them": [
    {
      "basics": {
        "username": "alice"
      },
      "public_keys": {
        "primary": {
          "kid": "0120abc123...",
          "bundle": "-----BEGIN PGP PUBLIC KEY BLOCK-----..."
        }
      }
    }
  ]
}
```

## Rate Limiting

Keybase API has rate limits. The client handles this by:

1. Detecting 429 status code
2. Automatically retrying with exponential backoff
3. Respecting `Retry-After` header (if present)

**Best Practices:**
- Batch multiple users in single request
- Use caching (see `cache` package)
- Don't make unnecessary API calls

## Testing

Run API client tests:

```bash
go test -v ./keybase/api/...
```

With coverage:

```bash
go test -cover ./keybase/api/...
```

## Configuration

### Default Configuration

```go
BaseURL:    "https://keybase.io/_/api/1.0"
Timeout:    30 * time.Second
MaxRetries: 3
RetryDelay: 1 * time.Second
```

### Custom Configuration Example

```go
config := &api.ClientConfig{
    BaseURL:    "https://custom.keybase.io/api",
    Timeout:    60 * time.Second,
    MaxRetries: 5,
    RetryDelay: 2 * time.Second,
}
client := api.NewClient(config)
```

## Performance

- **Batch lookups**: Fetch multiple users in one request
- **Connection reuse**: HTTP client reuses connections
- **Timeout handling**: Configurable timeouts prevent hangs
- **Retry efficiency**: Exponential backoff reduces server load

## Security

- **HTTPS only**: All requests use HTTPS
- **Input validation**: Usernames validated before API calls
- **No credentials**: Public API, no authentication required
- **User-Agent**: Identifies requests as from this client

## Examples

See the [examples directory](../../examples/) for complete usage examples.

## Related

- [Cache Package](../cache/README.md) - Automatic caching of API results
- [Keybase API Documentation](https://keybase.io/docs/api/1.0) - Official API docs
