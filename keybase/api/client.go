package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
			// Calculate delay with exponential backoff
			delay := c.RetryDelay * time.Duration(1<<uint(attempt-1))
			
			// If the previous error was a rate limit error with RetryAfter, use that instead
			if apiErr, ok := err.(*APIError); ok && apiErr.IsRateLimitError() && apiErr.RetryAfter > 0 {
				delay = apiErr.RetryAfter
			}
			
			select {
			case <-ctx.Done():
				return nil, wrapContextError(ctx.Err())
			case <-time.After(delay):
			}
		}
		
		response, err = c.doLookup(ctx, fullURL)
		if err == nil {
			break
		}
		
		// Don't retry on client errors (4xx) except rate limiting (429)
		if apiErr, ok := err.(*APIError); ok {
			if !apiErr.IsTemporary() && !apiErr.IsRateLimitError() {
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

// classifyHTTPError classifies HTTP client errors (network, timeout, etc.)
func classifyHTTPError(err error) *APIError {
	// Check for context errors first
	if errors.Is(err, context.DeadlineExceeded) {
		return &APIError{
			Message:    "request timed out while connecting to Keybase API",
			StatusCode: 0,
			Kind:       ErrorKindTimeout,
			Temporary:  true,
			Underlying: err,
		}
	}
	
	if errors.Is(err, context.Canceled) {
		return &APIError{
			Message:    "request was cancelled",
			StatusCode: 0,
			Kind:       ErrorKindTimeout,
			Temporary:  false,
			Underlying: err,
		}
	}
	
	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &APIError{
				Message:    fmt.Sprintf("network timeout while connecting to Keybase API: %v", netErr),
				StatusCode: 0,
				Kind:       ErrorKindTimeout,
				Temporary:  true,
				Underlying: err,
			}
		}
		return &APIError{
			Message:    fmt.Sprintf("network error while connecting to Keybase API: %v", netErr),
			StatusCode: 0,
			Kind:       ErrorKindNetwork,
			Temporary:  true,
			Underlying: err,
		}
	}
	
	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &APIError{
			Message:    fmt.Sprintf("DNS lookup failed for Keybase API: %v", dnsErr),
			StatusCode: 0,
			Kind:       ErrorKindNetwork,
			Temporary:  dnsErr.Temporary(),
			Underlying: err,
		}
	}
	
	// Check for connection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return &APIError{
			Message:    fmt.Sprintf("failed to connect to Keybase API: %v", opErr),
			StatusCode: 0,
			Kind:       ErrorKindNetwork,
			Temporary:  opErr.Temporary(),
			Underlying: err,
		}
	}
	
	// Generic HTTP error
	return &APIError{
		Message:    fmt.Sprintf("HTTP request failed: %v", err),
		StatusCode: 0,
		Kind:       ErrorKindNetwork,
		Temporary:  true,
		Underlying: err,
	}
}

// classifyHTTPStatusError classifies errors based on HTTP status code
func classifyHTTPStatusError(resp *http.Response, body []byte) *APIError {
	statusCode := resp.StatusCode
	bodyStr := string(body)
	
	// Truncate body for error message if too long
	if len(bodyStr) > 200 {
		bodyStr = bodyStr[:200] + "..."
	}
	
	// Handle rate limiting (429) with Retry-After header
	if statusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &APIError{
			Message:    fmt.Sprintf("rate limited by Keybase API (retry after %s): %s", retryAfter, bodyStr),
			StatusCode: statusCode,
			Kind:       ErrorKindRateLimit,
			Temporary:  true,
			RetryAfter: retryAfter,
		}
	}
	
	// Handle client errors (4xx)
	if statusCode >= 400 && statusCode < 500 {
		kind := ErrorKindInvalidInput
		message := fmt.Sprintf("Keybase API rejected request: %s", bodyStr)
		
		if statusCode == http.StatusNotFound {
			kind = ErrorKindNotFound
			message = fmt.Sprintf("user not found or endpoint does not exist: %s", bodyStr)
		} else if statusCode == http.StatusBadRequest {
			kind = ErrorKindInvalidInput
			message = fmt.Sprintf("invalid request to Keybase API: %s", bodyStr)
		} else if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			kind = ErrorKindInvalidInput
			message = fmt.Sprintf("authentication or authorization failed: %s", bodyStr)
		}
		
		return &APIError{
			Message:    message,
			StatusCode: statusCode,
			Kind:       kind,
			Temporary:  false,
		}
	}
	
	// Handle server errors (5xx)
	if statusCode >= 500 {
		return &APIError{
			Message:    fmt.Sprintf("Keybase API server error (status %d): %s", statusCode, bodyStr),
			StatusCode: statusCode,
			Kind:       ErrorKindServerError,
			Temporary:  true, // Server errors are usually temporary
		}
	}
	
	// Unknown status code
	return &APIError{
		Message:    fmt.Sprintf("unexpected HTTP status %d: %s", statusCode, bodyStr),
		StatusCode: statusCode,
		Kind:       ErrorKindUnknown,
		Temporary:  false,
	}
}

// parseRetryAfter parses the Retry-After header and returns a duration
// Supports both delay-seconds and HTTP-date formats
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	
	// Try parsing as seconds
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second
	}
	
	// Try parsing as HTTP date
	if t, err := http.ParseTime(header); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}
	
	// Default: no retry after
	return 0
}

// classifyAPIStatusCode classifies Keybase API status codes
func classifyAPIStatusCode(code int) ErrorKind {
	// Keybase API status codes (from their documentation)
	switch code {
	case 0:
		return ErrorKindUnknown // Should not happen, but handle it
	case 205: // User not found
		return ErrorKindNotFound
	case 207: // Bad username
		return ErrorKindInvalidInput
	default:
		return ErrorKindUnknown
	}
}

// wrapContextError wraps context errors with appropriate classification
func wrapContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &APIError{
			Message:    "operation timed out",
			StatusCode: 0,
			Kind:       ErrorKindTimeout,
			Temporary:  true,
			Underlying: err,
		}
	}
	
	if errors.Is(err, context.Canceled) {
		return &APIError{
			Message:    "operation was cancelled",
			StatusCode: 0,
			Kind:       ErrorKindTimeout,
			Temporary:  false,
			Underlying: err,
		}
	}
	
	return err
}

// doLookup performs the actual HTTP request
func (c *Client) doLookup(ctx context.Context, url string) (*LookupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, &APIError{
			Message:    fmt.Sprintf("failed to create request: %v", err),
			StatusCode: 0,
			Kind:       ErrorKindInvalidInput,
			Temporary:  false,
			Underlying: err,
		}
	}
	
	req.Header.Set("User-Agent", "pulumi-keybase-encryption/1.0")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		// Classify the error based on its type
		return nil, classifyHTTPError(err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &APIError{
			Message:    fmt.Sprintf("failed to read response body: %v", err),
			StatusCode: 0,
			Kind:       ErrorKindNetwork,
			Temporary:  true,
			Underlying: err,
		}
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, classifyHTTPStatusError(resp, body)
	}
	
	var lookupResp LookupResponse
	if err := json.Unmarshal(body, &lookupResp); err != nil {
		return nil, &APIError{
			Message:    fmt.Sprintf("failed to parse API response: %v", err),
			StatusCode: resp.StatusCode,
			Kind:       ErrorKindInvalidResponse,
			Temporary:  false,
			Underlying: err,
		}
	}
	
	if lookupResp.Status.Code != 0 {
		return nil, &APIError{
			Message:    fmt.Sprintf("API returned error: %s (code: %d)", lookupResp.Status.Name, lookupResp.Status.Code),
			StatusCode: resp.StatusCode,
			Kind:       classifyAPIStatusCode(lookupResp.Status.Code),
			Temporary:  false,
		}
	}
	
	return &lookupResp, nil
}

// parseResponse extracts public keys from the API response
func (c *Client) parseResponse(response *LookupResponse, requestedUsers []string) ([]UserPublicKey, error) {
	if response == nil {
		return nil, &APIError{
			Message:    "received nil response from Keybase API",
			StatusCode: 0,
			Kind:       ErrorKindInvalidResponse,
			Temporary:  false,
		}
	}
	
	if len(response.Them) == 0 {
		// No users found - could be because they don't exist
		if len(requestedUsers) == 1 {
			return nil, &APIError{
				Message:    fmt.Sprintf("user %q not found on Keybase", requestedUsers[0]),
				StatusCode: 0,
				Kind:       ErrorKindNotFound,
				Temporary:  false,
			}
		}
		return nil, &APIError{
			Message:    fmt.Sprintf("none of the requested users were found: %s", strings.Join(requestedUsers, ", ")),
			StatusCode: 0,
			Kind:       ErrorKindNotFound,
			Temporary:  false,
		}
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
			return nil, &APIError{
				Message:    fmt.Sprintf("user %q exists but has no primary public key configured", user.Basics.Username),
				StatusCode: 0,
				Kind:       ErrorKindInvalidResponse,
				Temporary:  false,
			}
		}
		
		results = append(results, UserPublicKey{
			Username:  user.Basics.Username,
			PublicKey: user.PublicKeys.Primary.Bundle,
			KeyID:     user.PublicKeys.Primary.KID,
		})
	}
	
	// Check if all requested users were found
	var missingUsers []string
	for _, username := range requestedUsers {
		if !foundUsers[username] {
			missingUsers = append(missingUsers, username)
		}
	}
	
	if len(missingUsers) > 0 {
		if len(missingUsers) == 1 {
			return nil, &APIError{
				Message:    fmt.Sprintf("user %q not found on Keybase", missingUsers[0]),
				StatusCode: 0,
				Kind:       ErrorKindNotFound,
				Temporary:  false,
			}
		}
		return nil, &APIError{
			Message:    fmt.Sprintf("users not found on Keybase: %s", strings.Join(missingUsers, ", ")),
			StatusCode: 0,
			Kind:       ErrorKindNotFound,
			Temporary:  false,
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

// ErrorKind represents the kind of error that occurred
type ErrorKind int

const (
	// ErrorKindUnknown represents an unknown error
	ErrorKindUnknown ErrorKind = iota
	// ErrorKindNetwork represents a network connectivity error
	ErrorKindNetwork
	// ErrorKindTimeout represents a timeout error
	ErrorKindTimeout
	// ErrorKindRateLimit represents a rate limiting error (429)
	ErrorKindRateLimit
	// ErrorKindNotFound represents a user not found error (404)
	ErrorKindNotFound
	// ErrorKindInvalidInput represents invalid input (400)
	ErrorKindInvalidInput
	// ErrorKindServerError represents a server error (5xx)
	ErrorKindServerError
	// ErrorKindInvalidResponse represents an error parsing the API response
	ErrorKindInvalidResponse
)

// String returns a string representation of the ErrorKind
func (k ErrorKind) String() string {
	switch k {
	case ErrorKindNetwork:
		return "NetworkError"
	case ErrorKindTimeout:
		return "TimeoutError"
	case ErrorKindRateLimit:
		return "RateLimitError"
	case ErrorKindNotFound:
		return "NotFoundError"
	case ErrorKindInvalidInput:
		return "InvalidInputError"
	case ErrorKindServerError:
		return "ServerError"
	case ErrorKindInvalidResponse:
		return "InvalidResponseError"
	default:
		return "UnknownError"
	}
}

// APIError represents an API error with detailed classification
type APIError struct {
	// Message is the human-readable error message
	Message string
	// StatusCode is the HTTP status code (0 if not an HTTP error)
	StatusCode int
	// Kind is the classification of the error
	Kind ErrorKind
	// Temporary indicates if the error is temporary and can be retried
	Temporary bool
	// RetryAfter is the duration to wait before retrying (for rate limiting)
	RetryAfter time.Duration
	// Underlying is the underlying error that caused this error
	Underlying error
}

func (e *APIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s (HTTP %d): %s", e.Kind, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

// IsTemporary returns true if the error is temporary and can be retried
func (e *APIError) IsTemporary() bool {
	return e.Temporary
}

// Unwrap returns the underlying error for errors.Is and errors.As support
func (e *APIError) Unwrap() error {
	return e.Underlying
}

// IsNetworkError returns true if the error is a network connectivity error
func (e *APIError) IsNetworkError() bool {
	return e.Kind == ErrorKindNetwork
}

// IsTimeout returns true if the error is a timeout error
func (e *APIError) IsTimeout() bool {
	return e.Kind == ErrorKindTimeout
}

// IsRateLimitError returns true if the error is a rate limiting error
func (e *APIError) IsRateLimitError() bool {
	return e.Kind == ErrorKindRateLimit
}
