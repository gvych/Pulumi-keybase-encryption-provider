package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultAPIEndpoint is the Keybase API endpoint
	DefaultAPIEndpoint = "https://keybase.io/_/api/1.0"
	
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
	
	// DefaultMaxRetries is the default number of retries for API calls
	DefaultMaxRetries = 3
	
	// DefaultRetryDelay is the initial delay between retries
	DefaultRetryDelay = 1 * time.Second
)

// Client represents a Keybase API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	MaxRetries int
	RetryDelay time.Duration
}

// ClientConfig holds configuration for the API client
type ClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultClientConfig returns the default API client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:    DefaultAPIEndpoint,
		Timeout:    DefaultTimeout,
		MaxRetries: DefaultMaxRetries,
		RetryDelay: DefaultRetryDelay,
	}
}

// NewClient creates a new Keybase API client
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultAPIEndpoint
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	maxRetries := config.MaxRetries
	if maxRetries < 0 {
		// Negative values would skip the retry loop entirely, leading to a nil
		// response and potential nil dereference later.
		maxRetries = 0
	}

	retryDelay := config.RetryDelay
	if retryDelay <= 0 {
		retryDelay = DefaultRetryDelay
	}
	
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
	}
}

// UserPublicKey represents a user's public key information
type UserPublicKey struct {
	Username  string
	PublicKey string
	KeyID     string
}

// LookupUsers fetches public keys for multiple users
func (c *Client) LookupUsers(ctx context.Context, usernames []string) ([]UserPublicKey, error) {
	if len(usernames) == 0 {
		return nil, fmt.Errorf("no usernames provided")
	}
	
	// Validate usernames
	for _, username := range usernames {
		if err := ValidateUsername(username); err != nil {
			return nil, fmt.Errorf("invalid username %q: %w", username, err)
		}
	}
	
	// Build request URL
	reqURL := fmt.Sprintf("%s/user/lookup.json", c.BaseURL)
	params := url.Values{}
	params.Set("usernames", strings.Join(usernames, ","))
	params.Set("fields", "public_keys")
	
	fullURL := fmt.Sprintf("%s?%s", reqURL, params.Encode())
	
	// Make API call with retries
	var response *LookupResponse
	var err error
	
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := c.RetryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		
		response, err = c.doLookup(ctx, fullURL)
		if err == nil {
			break
		}
		
		// Don't retry on client errors (4xx)
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			if apiErr.StatusCode != 429 { // Except rate limiting
				break
			}
		}
	}
	
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("lookup succeeded but response was nil")
	}
	
	// Parse response
	return c.parseResponse(response, usernames)
}

// doLookup performs the actual HTTP request
func (c *Client) doLookup(ctx context.Context, url string) (*LookupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "pulumi-keybase-encryption/1.0")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &APIError{
			Message:    fmt.Sprintf("HTTP request failed: %v", err),
			StatusCode: 0,
			Temporary:  true,
		}
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			Message:    fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)),
			StatusCode: resp.StatusCode,
			Temporary:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}
	
	var lookupResp LookupResponse
	if err := json.Unmarshal(body, &lookupResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}
	
	if lookupResp.Status.Code != 0 {
		return nil, fmt.Errorf("API error: %s", lookupResp.Status.Name)
	}
	
	return &lookupResp, nil
}

// parseResponse extracts public keys from the API response
func (c *Client) parseResponse(response *LookupResponse, requestedUsers []string) ([]UserPublicKey, error) {
	if response == nil {
		return nil, fmt.Errorf("nil response")
	}
	if len(response.Them) == 0 {
		return nil, fmt.Errorf("no users found in response")
	}
	
	var results []UserPublicKey
	foundUsers := make(map[string]bool)
	
	for _, user := range response.Them {
		if user.Basics.Username == "" {
			continue
		}
		
		foundUsers[user.Basics.Username] = true
		
		// Extract primary public key
		if user.PublicKeys.Primary.Bundle == "" {
			return nil, fmt.Errorf("user %q has no primary public key", user.Basics.Username)
		}
		
		results = append(results, UserPublicKey{
			Username:  user.Basics.Username,
			PublicKey: user.PublicKeys.Primary.Bundle,
			KeyID:     user.PublicKeys.Primary.KID,
		})
	}
	
	// Check if all requested users were found
	for _, username := range requestedUsers {
		if !foundUsers[username] {
			return nil, fmt.Errorf("user not found: %s", username)
		}
	}
	
	return results, nil
}

// ValidateUsername validates a Keybase username
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	
	// Keybase usernames are alphanumeric + underscore only
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
			 (r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("username contains invalid character: %c", r)
		}
	}
	
	return nil
}

// LookupResponse represents the API response for user lookup
type LookupResponse struct {
	Status Status   `json:"status"`
	Them   []User   `json:"them"`
}

// Status represents the API response status
type Status struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// User represents a Keybase user
type User struct {
	Basics     Basics     `json:"basics"`
	PublicKeys PublicKeys `json:"public_keys"`
}

// Basics contains basic user information
type Basics struct {
	Username string `json:"username"`
}

// PublicKeys contains user's public keys
type PublicKeys struct {
	Primary PrimaryKey `json:"primary"`
}

// PrimaryKey represents the primary public key
type PrimaryKey struct {
	KID    string `json:"kid"`
	Bundle string `json:"bundle"`
}

// APIError represents an API error
type APIError struct {
	Message    string
	StatusCode int
	Temporary  bool
}

func (e *APIError) Error() string {
	return e.Message
}

// IsTemporary returns true if the error is temporary and can be retried
func (e *APIError) IsTemporary() bool {
	return e.Temporary
}
