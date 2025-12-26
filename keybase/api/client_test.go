package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := &ClientConfig{
		BaseURL:    "https://test.keybase.io",
		Timeout:    10 * time.Second,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	}
	
	client := NewClient(config)
	
	if client.BaseURL != config.BaseURL {
		t.Errorf("BaseURL = %v, want %v", client.BaseURL, config.BaseURL)
	}
	
	if client.MaxRetries != config.MaxRetries {
		t.Errorf("MaxRetries = %v, want %v", client.MaxRetries, config.MaxRetries)
	}
	
	if client.RetryDelay != config.RetryDelay {
		t.Errorf("RetryDelay = %v, want %v", client.RetryDelay, config.RetryDelay)
	}
}

func TestNewClientNegativeMaxRetriesClamped(t *testing.T) {
	config := &ClientConfig{
		BaseURL:    "https://test.keybase.io",
		Timeout:    10 * time.Second,
		MaxRetries: -1,
		RetryDelay: 2 * time.Second,
	}

	client := NewClient(config)
	if client.MaxRetries != 0 {
		t.Errorf("MaxRetries = %v, want %v", client.MaxRetries, 0)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()
	
	if config.BaseURL != DefaultAPIEndpoint {
		t.Errorf("BaseURL = %v, want %v", config.BaseURL, DefaultAPIEndpoint)
	}
	
	if config.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", config.Timeout, DefaultTimeout)
	}
	
	if config.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %v, want %v", config.MaxRetries, DefaultMaxRetries)
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "valid lowercase",
			username: "alice",
			wantErr:  false,
		},
		{
			name:     "valid uppercase",
			username: "ALICE",
			wantErr:  false,
		},
		{
			name:     "valid with numbers",
			username: "alice123",
			wantErr:  false,
		},
		{
			name:     "valid with underscore",
			username: "alice_bob",
			wantErr:  false,
		},
		{
			name:     "empty string",
			username: "",
			wantErr:  true,
		},
		{
			name:     "contains dash",
			username: "alice-bob",
			wantErr:  true,
		},
		{
			name:     "contains space",
			username: "alice bob",
			wantErr:  true,
		},
		{
			name:     "contains special char",
			username: "alice@example",
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLookupUsersSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{
				Code: 0,
				Name: "OK",
			},
			Them: []User{
				{
					Basics: Basics{
						Username: "alice",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "test_kid_alice",
							Bundle: "test_bundle_alice",
						},
					},
				},
				{
					Basics: Basics{
						Username: "bob",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "test_kid_bob",
							Bundle: "test_bundle_bob",
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	
	keys, err := client.LookupUsers(context.Background(), []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("LookupUsers() error = %v", err)
	}
	
	if len(keys) != 2 {
		t.Fatalf("LookupUsers() returned %d keys, want 2", len(keys))
	}
	
	if keys[0].Username != "alice" {
		t.Errorf("keys[0].Username = %v, want alice", keys[0].Username)
	}
	
	if keys[1].Username != "bob" {
		t.Errorf("keys[1].Username = %v, want bob", keys[1].Username)
	}
}

func TestLookupUsersNegativeMaxRetriesDoesNotSkipRequest(t *testing.T) {
	// This test guards against the bug where MaxRetries < 0 skips the retry loop
	// entirely, leaving (response, err) as (nil, nil) and causing a nil deref.
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		response := LookupResponse{
			Status: Status{
				Code: 0,
				Name: "OK",
			},
			Them: []User{
				{
					Basics: Basics{
						Username: "alice",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "test_kid_alice",
							Bundle: "test_bundle_alice",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: -1,
		RetryDelay: 1 * time.Millisecond,
	}

	client := NewClient(config)
	keys, err := client.LookupUsers(context.Background(), []string{"alice"})
	if err != nil {
		t.Fatalf("LookupUsers() error = %v", err)
	}
	if len(keys) != 1 || keys[0].Username != "alice" {
		t.Fatalf("LookupUsers() returned %+v, want alice", keys)
	}
	if requests != 1 {
		t.Fatalf("server requests = %d, want 1", requests)
	}
}

func TestLookupUsersNotFound(t *testing.T) {
	// Create mock server that returns empty result
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{
				Code: 0,
				Name: "OK",
			},
			Them: []User{},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	
	_, err := client.LookupUsers(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Error("LookupUsers() expected error for nonexistent user, got nil")
	}
}

func TestLookupUsersServerError(t *testing.T) {
	// Create mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0, // No retries for faster test
	}
	
	client := NewClient(config)
	
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	if err == nil {
		t.Error("LookupUsers() expected error for server error, got nil")
	}
	
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
	
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %v, want %v", apiErr.StatusCode, http.StatusInternalServerError)
	}
	
	if !apiErr.IsTemporary() {
		t.Error("Server error should be temporary")
	}
}

func TestLookupUsersInvalidUsername(t *testing.T) {
	config := DefaultClientConfig()
	client := NewClient(config)
	
	_, err := client.LookupUsers(context.Background(), []string{"invalid@username"})
	if err == nil {
		t.Error("LookupUsers() expected error for invalid username, got nil")
	}
}

func TestLookupUsersEmptyList(t *testing.T) {
	config := DefaultClientConfig()
	client := NewClient(config)
	
	_, err := client.LookupUsers(context.Background(), []string{})
	if err == nil {
		t.Error("LookupUsers() expected error for empty username list, got nil")
	}
}

func TestLookupUsersContextCancellation(t *testing.T) {
	// Create mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	
	_, err := client.LookupUsers(ctx, []string{"alice"})
	if err == nil {
		t.Error("LookupUsers() expected error for cancelled context, got nil")
	}
}

func TestAPIErrorIsTemporary(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{
			name:       "500 is temporary",
			statusCode: 500,
			want:       true,
		},
		{
			name:       "503 is temporary",
			statusCode: 503,
			want:       true,
		},
		{
			name:       "429 is temporary",
			statusCode: 429,
			want:       true,
		},
		{
			name:       "404 is not temporary",
			statusCode: 404,
			want:       false,
		},
		{
			name:       "400 is not temporary",
			statusCode: 400,
			want:       false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{
				StatusCode: tt.statusCode,
				Temporary:  tt.statusCode >= 500 || tt.statusCode == 429,
			}
			
			if got := err.IsTemporary(); got != tt.want {
				t.Errorf("IsTemporary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLookupUsersMissingPublicKey(t *testing.T) {
	// Create mock server that returns user without public key
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{
				Code: 0,
				Name: "OK",
			},
			Them: []User{
				{
					Basics: Basics{
						Username: "alice",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "test_kid",
							Bundle: "", // Empty bundle
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	if err == nil {
		t.Error("LookupUsers() expected error for missing public key, got nil")
	}
}

func TestRateLimitWithRetryAfter(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			// First attempt: return 429 with Retry-After header
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limit exceeded"))
			return
		}
		// Second attempt: success
		response := LookupResponse{
			Status: Status{Code: 0, Name: "OK"},
			Them: []User{
				{
					Basics:     Basics{Username: "alice"},
					PublicKeys: PublicKeys{Primary: PrimaryKey{KID: "kid", Bundle: "bundle"}},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    10 * time.Second,
		MaxRetries: 2,
		RetryDelay: 100 * time.Millisecond,
	}

	client := NewClient(config)
	start := time.Now()
	keys, err := client.LookupUsers(context.Background(), []string{"alice"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("LookupUsers() should succeed after retry: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("Expected 1 key, got %d", len(keys))
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	// Should have waited at least 1 second for Retry-After
	if elapsed < 1*time.Second {
		t.Errorf("Expected at least 1s delay for Retry-After, got %v", elapsed)
	}
}

func TestRateLimitExhaustsRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "1") // Short retry-after for faster test
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Rate limit exceeded"))
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    10 * time.Second,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})

	if err == nil {
		t.Fatal("Expected error for exhausted retries")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if !apiErr.IsRateLimitError() {
		t.Errorf("Expected rate limit error, got %v", apiErr.Kind)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", apiErr.StatusCode)
	}
	if apiErr.RetryAfter != 1*time.Second {
		t.Errorf("Expected RetryAfter=1s, got %v", apiErr.RetryAfter)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (initial + 2 retries), got %d", attempts)
	}
}

func TestNetworkErrorClassification(t *testing.T) {
	// Use invalid URL to trigger network error
	config := &ClientConfig{
		BaseURL:    "http://invalid-host-that-does-not-exist-12345.test",
		Timeout:    2 * time.Second,
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})

	if err == nil {
		t.Fatal("Expected network error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if !apiErr.IsNetworkError() {
		t.Errorf("Expected network error, got %v", apiErr.Kind)
	}
	if !apiErr.IsTemporary() {
		t.Error("Network errors should be temporary")
	}
}

func TestTimeoutErrorClassification(t *testing.T) {
	// Create server with slow response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    50 * time.Millisecond, // Very short timeout
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if !apiErr.IsTimeout() {
		t.Errorf("Expected timeout error, got %v", apiErr.Kind)
	}
	if !apiErr.IsTemporary() {
		t.Error("Timeout errors should be temporary")
	}
}

func TestContextCancellationDuringRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    10 * time.Second,
		MaxRetries: 5,
		RetryDelay: 200 * time.Millisecond,
	}

	client := NewClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_, err := client.LookupUsers(ctx, []string{"alice"})

	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	// Should have tried at most twice before context cancelled during backoff
	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts before context cancellation, got %d", attempts)
	}
}

func TestNotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{Code: 0, Name: "OK"},
			Them:   []User{}, // Empty result
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"nonexistent"})

	if err == nil {
		t.Fatal("Expected not found error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if apiErr.Kind != ErrorKindNotFound {
		t.Errorf("Expected ErrorKindNotFound, got %v", apiErr.Kind)
	}
	if apiErr.IsTemporary() {
		t.Error("Not found errors should not be temporary")
	}
}

func TestMultipleUsersNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{Code: 0, Name: "OK"},
			Them: []User{
				{
					Basics:     Basics{Username: "alice"},
					PublicKeys: PublicKeys{Primary: PrimaryKey{KID: "kid1", Bundle: "bundle1"}},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice", "bob", "charlie"})

	if err == nil {
		t.Fatal("Expected error for missing users")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if apiErr.Kind != ErrorKindNotFound {
		t.Errorf("Expected ErrorKindNotFound, got %v", apiErr.Kind)
	}
	// Should mention multiple users
	if !strings.Contains(apiErr.Message, "bob") || !strings.Contains(apiErr.Message, "charlie") {
		t.Errorf("Error message should mention missing users: %s", apiErr.Message)
	}
}

func TestServerError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})

	if err == nil {
		t.Fatal("Expected server error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if apiErr.Kind != ErrorKindServerError {
		t.Errorf("Expected ErrorKindServerError, got %v", apiErr.Kind)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", apiErr.StatusCode)
	}
	if !apiErr.IsTemporary() {
		t.Error("Server errors should be temporary")
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})

	if err == nil {
		t.Fatal("Expected invalid response error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if apiErr.Kind != ErrorKindInvalidResponse {
		t.Errorf("Expected ErrorKindInvalidResponse, got %v", apiErr.Kind)
	}
	if apiErr.IsTemporary() {
		t.Error("Invalid response errors should not be temporary")
	}
}

func TestAPIStatusCodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{Code: 205, Name: "NOT_FOUND"}, // Keybase API error code
			Them:   []User{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})

	if err == nil {
		t.Fatal("Expected API error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected *APIError, got %T", err)
	}

	if apiErr.Kind != ErrorKindNotFound {
		t.Errorf("Expected ErrorKindNotFound for status code 205, got %v", apiErr.Kind)
	}
}

func TestErrorKindString(t *testing.T) {
	tests := []struct {
		kind ErrorKind
		want string
	}{
		{ErrorKindNetwork, "NetworkError"},
		{ErrorKindTimeout, "TimeoutError"},
		{ErrorKindRateLimit, "RateLimitError"},
		{ErrorKindNotFound, "NotFoundError"},
		{ErrorKindInvalidInput, "InvalidInputError"},
		{ErrorKindServerError, "ServerError"},
		{ErrorKindInvalidResponse, "InvalidResponseError"},
		{ErrorKindUnknown, "UnknownError"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("ErrorKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIErrorUnwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	apiErr := &APIError{
		Message:    "wrapped error",
		Underlying: underlying,
	}

	if !errors.Is(apiErr, underlying) {
		t.Error("errors.Is should find underlying error")
	}

	var unwrapped *APIError
	if !errors.As(apiErr, &unwrapped) {
		t.Error("errors.As should work with APIError")
	}
	if unwrapped != apiErr {
		t.Error("errors.As should return the same APIError")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{
			name:   "seconds format",
			header: "120",
			want:   120 * time.Second,
		},
		{
			name:   "empty header",
			header: "",
			want:   0,
		},
		{
			name:   "invalid format",
			header: "invalid",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfter(tt.header)
			if got != tt.want {
				t.Errorf("parseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPStatusErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantKind   ErrorKind
		wantTemp   bool
	}{
		{"BadRequest", http.StatusBadRequest, ErrorKindInvalidInput, false},
		{"Unauthorized", http.StatusUnauthorized, ErrorKindInvalidInput, false},
		{"Forbidden", http.StatusForbidden, ErrorKindInvalidInput, false},
		{"NotFound", http.StatusNotFound, ErrorKindNotFound, false},
		{"TooManyRequests", http.StatusTooManyRequests, ErrorKindRateLimit, true},
		{"InternalServerError", http.StatusInternalServerError, ErrorKindServerError, true},
		{"BadGateway", http.StatusBadGateway, ErrorKindServerError, true},
		{"ServiceUnavailable", http.StatusServiceUnavailable, ErrorKindServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error message"))
			}))
			defer server.Close()

			config := &ClientConfig{
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				MaxRetries: 0,
			}

			client := NewClient(config)
			_, err := client.LookupUsers(context.Background(), []string{"alice"})

			if err == nil {
				t.Fatal("Expected error")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("Expected *APIError, got %T", err)
			}

			if apiErr.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", apiErr.Kind, tt.wantKind)
			}
			if apiErr.IsTemporary() != tt.wantTemp {
				t.Errorf("IsTemporary() = %v, want %v", apiErr.IsTemporary(), tt.wantTemp)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %v, want %v", apiErr.StatusCode, tt.statusCode)
			}
		})
	}
}
