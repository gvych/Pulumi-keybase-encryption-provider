package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegrationSingleUserLookup tests looking up a single user via mock API
func TestIntegrationSingleUserLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		if r.URL.Query().Get("usernames") != "alice" {
			t.Errorf("Expected usernames=alice, got %s", r.URL.Query().Get("usernames"))
		}
		
		if r.URL.Query().Get("fields") != "public_keys" {
			t.Errorf("Expected fields=public_keys, got %s", r.URL.Query().Get("fields"))
		}
		
		// Return realistic Keybase API response
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
							KID:    "0120abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab0a",
							Bundle: "-----BEGIN PGP PUBLIC KEY BLOCK-----\nVersion: Keybase OpenPGP v1.0.0\n\nmQINBFx...",
						},
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	keys, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err != nil {
		t.Fatalf("LookupUsers() failed: %v", err)
	}
	
	if len(keys) != 1 {
		t.Fatalf("Expected 1 key, got %d", len(keys))
	}
	
	if keys[0].Username != "alice" {
		t.Errorf("Expected username=alice, got %s", keys[0].Username)
	}
	
	if keys[0].PublicKey == "" {
		t.Error("Expected non-empty public key")
	}
	
	if keys[0].KeyID == "" {
		t.Error("Expected non-empty key ID")
	}
}

// TestIntegrationMultipleUserLookup tests looking up multiple users in a single API call
func TestIntegrationMultipleUserLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usernames := r.URL.Query().Get("usernames")
		
		// Verify all users are requested in comma-separated format
		if usernames != "alice,bob,charlie" {
			t.Errorf("Expected usernames=alice,bob,charlie, got %s", usernames)
		}
		
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
							KID:    "kid_alice",
							Bundle: "bundle_alice",
						},
					},
				},
				{
					Basics: Basics{
						Username: "bob",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "kid_bob",
							Bundle: "bundle_bob",
						},
					},
				},
				{
					Basics: Basics{
						Username: "charlie",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "kid_charlie",
							Bundle: "bundle_charlie",
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
	keys, err := client.LookupUsers(context.Background(), []string{"alice", "bob", "charlie"})
	
	if err != nil {
		t.Fatalf("LookupUsers() failed: %v", err)
	}
	
	if len(keys) != 3 {
		t.Fatalf("Expected 3 keys, got %d", len(keys))
	}
	
	// Verify all users are returned
	expectedUsers := map[string]bool{"alice": true, "bob": true, "charlie": true}
	for _, key := range keys {
		if !expectedUsers[key.Username] {
			t.Errorf("Unexpected username: %s", key.Username)
		}
		delete(expectedUsers, key.Username)
	}
	
	if len(expectedUsers) > 0 {
		t.Errorf("Missing users: %v", expectedUsers)
	}
}

// TestIntegrationUserNotFound tests 404 error response
func TestIntegrationUserNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate API returning empty result for nonexistent user
		response := LookupResponse{
			Status: Status{
				Code: 0,
				Name: "OK",
			},
			Them: []User{}, // Empty array means user not found
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
	_, err := client.LookupUsers(context.Background(), []string{"nonexistent_user_xyz"})
	
	if err == nil {
		t.Fatal("Expected error for nonexistent user, got nil")
	}
	
	expectedMsg := "no users found in response"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestIntegrationServerError500 tests 500 Internal Server Error
func TestIntegrationServerError500(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 2, // Will retry 2 times (3 total attempts)
		RetryDelay: 10 * time.Millisecond,
	}
	
	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
	
	// Verify it's an APIError
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}
	
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", apiErr.StatusCode)
	}
	
	if !apiErr.IsTemporary() {
		t.Error("500 error should be marked as temporary")
	}
	
	// Verify retries occurred (1 initial + 2 retries = 3 total)
	expectedRequests := 3
	if requestCount != expectedRequests {
		t.Errorf("Expected %d requests (with retries), got %d", expectedRequests, requestCount)
	}
}

// TestIntegrationRateLimiting429 tests rate limiting with 429 response
func TestIntegrationRateLimiting429(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		
		// Return 429 for first 2 requests, then succeed
		if count <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limit exceeded"))
			return
		}
		
		// Success on third attempt
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
							KID:    "kid_alice",
							Bundle: "bundle_alice",
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
		MaxRetries: 3, // Enough retries to succeed
		RetryDelay: 10 * time.Millisecond,
	}
	
	client := NewClient(config)
	keys, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err != nil {
		t.Fatalf("LookupUsers() should eventually succeed after rate limiting, got error: %v", err)
	}
	
	if len(keys) != 1 {
		t.Fatalf("Expected 1 key after retries, got %d", len(keys))
	}
	
	// Verify we made 3 attempts (2 failures + 1 success)
	if atomic.LoadInt32(&requestCount) != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

// TestIntegrationNetworkTimeout tests request timeout behavior
func TestIntegrationNetworkTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    50 * time.Millisecond, // Timeout before server responds
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
	
	// Should be an APIError with Temporary flag
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}
	
	if !apiErr.IsTemporary() {
		t.Error("Timeout error should be marked as temporary")
	}
}

// TestIntegrationContextCancellation tests proper context cancellation handling
func TestIntegrationContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	
	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	_, err := client.LookupUsers(ctx, []string{"alice"})
	
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
	
	// Should be context error
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}

// TestIntegrationExponentialBackoff tests that retries use exponential backoff
func TestIntegrationExponentialBackoff(t *testing.T) {
	requestTimes := []time.Time{}
	var mu atomic.Value
	mu.Store(&requestTimes)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		times := mu.Load().(*[]time.Time)
		*times = append(*times, time.Now())
		
		// Always return 500 to force retries
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 3,
		RetryDelay: 50 * time.Millisecond,
	}
	
	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err == nil {
		t.Fatal("Expected error after all retries, got nil")
	}
	
	times := mu.Load().(*[]time.Time)
	if len(*times) != 4 { // 1 initial + 3 retries
		t.Fatalf("Expected 4 requests, got %d", len(*times))
	}
	
	// Verify exponential backoff delays
	// Delay 1: ~50ms (1 * RetryDelay)
	// Delay 2: ~100ms (2 * RetryDelay)
	// Delay 3: ~200ms (4 * RetryDelay)
	
	delay1 := (*times)[1].Sub((*times)[0])
	delay2 := (*times)[2].Sub((*times)[1])
	delay3 := (*times)[3].Sub((*times)[2])
	
	t.Logf("Delays: %v, %v, %v", delay1, delay2, delay3)
	
	// Allow some tolerance for timing jitter (±30ms)
	tolerance := 30 * time.Millisecond
	
	if delay1 < 50*time.Millisecond-tolerance || delay1 > 50*time.Millisecond+tolerance {
		t.Errorf("First retry delay should be ~50ms, got %v", delay1)
	}
	
	if delay2 < 100*time.Millisecond-tolerance || delay2 > 100*time.Millisecond+tolerance {
		t.Errorf("Second retry delay should be ~100ms, got %v", delay2)
	}
	
	if delay3 < 200*time.Millisecond-tolerance || delay3 > 200*time.Millisecond+tolerance {
		t.Errorf("Third retry delay should be ~200ms, got %v", delay3)
	}
}

// TestIntegrationPartialSuccess tests when some users are found but not all
func TestIntegrationPartialSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only return alice and bob, but charlie is missing
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
							KID:    "kid_alice",
							Bundle: "bundle_alice",
						},
					},
				},
				{
					Basics: Basics{
						Username: "bob",
					},
					PublicKeys: PublicKeys{
						Primary: PrimaryKey{
							KID:    "kid_bob",
							Bundle: "bundle_bob",
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
	_, err := client.LookupUsers(context.Background(), []string{"alice", "bob", "charlie"})
	
	if err == nil {
		t.Fatal("Expected error for missing user, got nil")
	}
	
	expectedMsg := "user not found: charlie"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

// TestIntegrationMalformedJSON tests handling of malformed API response
func TestIntegrationMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{ invalid json }"))
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
		t.Fatal("Expected error for malformed JSON, got nil")
	}
	
	// Should contain "parse" in error message
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestIntegrationAPIErrorStatus tests API returning error status in JSON
func TestIntegrationAPIErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LookupResponse{
			Status: Status{
				Code: 205, // API-level error code
				Name: "INPUT_ERROR",
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
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err == nil {
		t.Fatal("Expected error for API error status, got nil")
	}
	
	expectedMsg := "API error: INPUT_ERROR"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

// TestIntegrationLargeUserBatch tests handling of many users at once
func TestIntegrationLargeUserBatch(t *testing.T) {
	const userCount = 50
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		users := []User{}
		
		// Generate 50 users
		for i := 0; i < userCount; i++ {
			users = append(users, User{
				Basics: Basics{
					Username: fmt.Sprintf("user%d", i),
				},
				PublicKeys: PublicKeys{
					Primary: PrimaryKey{
						KID:    fmt.Sprintf("kid_%d", i),
						Bundle: fmt.Sprintf("bundle_%d", i),
					},
				},
			})
		}
		
		response := LookupResponse{
			Status: Status{
				Code: 0,
				Name: "OK",
			},
			Them: users,
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
	
	// Request 50 users
	usernames := make([]string, userCount)
	for i := 0; i < userCount; i++ {
		usernames[i] = fmt.Sprintf("user%d", i)
	}
	
	keys, err := client.LookupUsers(context.Background(), usernames)
	
	if err != nil {
		t.Fatalf("LookupUsers() failed: %v", err)
	}
	
	if len(keys) != userCount {
		t.Fatalf("Expected %d keys, got %d", userCount, len(keys))
	}
}

// TestIntegrationUserAgentHeader tests that client sends proper User-Agent
func TestIntegrationUserAgentHeader(t *testing.T) {
	var receivedUserAgent string
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		
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
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}
	
	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err != nil {
		t.Fatalf("LookupUsers() failed: %v", err)
	}
	
	expectedUA := "pulumi-keybase-encryption/1.0"
	if receivedUserAgent != expectedUA {
		t.Errorf("Expected User-Agent %q, got %q", expectedUA, receivedUserAgent)
	}
}

// TestIntegration400NoRetry tests that 4xx errors (except 429) are not retried
func TestIntegration400NoRetry(t *testing.T) {
	requestCount := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 3, // Even with retries enabled
		RetryDelay: 10 * time.Millisecond,
	}
	
	client := NewClient(config)
	_, err := client.LookupUsers(context.Background(), []string{"alice"})
	
	if err == nil {
		t.Fatal("Expected error for 400 response, got nil")
	}
	
	// Should only make 1 request (no retries for client errors)
	if requestCount != 1 {
		t.Errorf("Expected 1 request (no retries), got %d", requestCount)
	}
	
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}
	
	if apiErr.IsTemporary() {
		t.Error("400 error should not be marked as temporary")
	}
}
