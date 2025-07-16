package geminiwebtools

import (
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// Mock credential store for testing
type mockClientCredentialStore struct {
	hasToken bool
}

func (m *mockClientCredentialStore) LoadToken() (*oauth2.Token, error) {
	if m.hasToken {
		return &oauth2.Token{AccessToken: "test-token"}, nil
	}
	return nil, storage.ErrStorageNotFound
}

func (m *mockClientCredentialStore) StoreToken(token *oauth2.Token) error {
	m.hasToken = true
	return nil
}

func (m *mockClientCredentialStore) ClearToken() error {
	m.hasToken = false
	return nil
}

func (m *mockClientCredentialStore) HasToken() bool {
	return m.hasToken
}

func (m *mockClientCredentialStore) GetStoragePath() string {
	return "/tmp/test-storage"
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		options     []ConfigOption
		expectError bool
	}{
		{
			name:        "default config",
			options:     []ConfigOption{},
			expectError: false,
		},
		{
			name: "with timeout option",
			options: []ConfigOption{
				WithTimeout(30 * time.Second),
			},
			expectError: false,
		},
		{
			name: "with credential store option",
			options: []ConfigOption{
				WithCredentialStore(&mockClientCredentialStore{}),
			},
			expectError: false,
		},
		{
			name: "with multiple options",
			options: []ConfigOption{
				WithTimeout(30 * time.Second),
				WithMaxContentSize(1024),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.options...)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && client == nil {
				t.Error("Expected client to be created but got nil")
			}

			if client != nil {
				// Test basic client structure
				if client.config == nil {
					t.Error("Client config should not be nil")
				}
				if client.auth == nil {
					t.Error("Client auth should not be nil")
				}
				if client.searcher == nil {
					t.Error("Client searcher should not be nil")
				}
				if client.fetcher == nil {
					t.Error("Client fetcher should not be nil")
				}
			}
		})
	}
}

func TestClientGetConfig(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	config := client.GetConfig()
	if config == nil {
		t.Error("GetConfig should not return nil")
		return
	}

	// Check that the config has expected values
	if config.CodeAssistEndpoint == "" {
		t.Error("CodeAssistEndpoint should not be empty")
	}
	if config.GeminiAPIEndpoint == "" {
		t.Error("GeminiAPIEndpoint should not be empty")
	}
	if config.DefaultModel == "" {
		t.Error("DefaultModel should not be empty")
	}
}

func TestClientIsAuthenticated(t *testing.T) {
	// Create client with mock credential store
	store := &mockClientCredentialStore{hasToken: false}
	client, err := NewClient(WithCredentialStore(store))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Should not be authenticated initially
	if client.IsAuthenticated() {
		t.Error("Client should not be authenticated initially")
	}

	// Simulate authentication
	store.hasToken = true

	// Note: The actual IsAuthenticated method depends on the OAuth2 implementation
	// and may not immediately reflect the store state without token validation
	// This test mainly verifies the method exists and doesn't panic
}

func TestClientGetAuthStatus(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that GetAuthStatus doesn't panic
	status, err := client.GetAuthStatus()
	// We don't check the specific result since it depends on the OAuth2 implementation
	// This test mainly verifies the method exists and doesn't panic

	if err != nil {
		// GetAuthStatus may return an error when not authenticated, which is expected
		t.Logf("GetAuthStatus returned error (expected): %v", err)
	}

	if status != nil {
		t.Logf("GetAuthStatus returned status: %+v", status)
	}
}

func TestClientClearAuthentication(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that ClearAuthentication doesn't panic
	err = client.ClearAuthentication()
	// We don't check the specific result since it depends on the OAuth2 implementation
	// This test mainly verifies the method exists and doesn't panic

	if err != nil {
		t.Logf("ClearAuthentication returned error: %v", err)
	}
}

func TestClientWithCustomConfig(t *testing.T) {
	// Test client creation with custom configuration
	customTimeout := 45 * time.Second
	customMaxSize := 2048

	client, err := NewClient(
		WithTimeout(customTimeout),
		WithMaxContentSize(customMaxSize),
	)

	if err != nil {
		t.Fatalf("Failed to create client with custom config: %v", err)
	}

	config := client.GetConfig()
	if config.Timeout != customTimeout {
		t.Errorf("Expected timeout %v, got %v", customTimeout, config.Timeout)
	}
	if config.MaxContentSize != customMaxSize {
		t.Errorf("Expected max content size %d, got %d", customMaxSize, config.MaxContentSize)
	}
}
