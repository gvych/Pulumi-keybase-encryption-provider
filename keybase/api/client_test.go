package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
