package geminiwebtools

import (
	"testing"

	"golang.org/x/oauth2"

	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// Mock credential store for websearch testing
type mockWebSearchCredentialStore struct {
	hasToken bool
}

func (m *mockWebSearchCredentialStore) LoadToken() (*oauth2.Token, error) {
	if m.hasToken {
		return &oauth2.Token{AccessToken: "test-token"}, nil
	}
	return nil, storage.ErrStorageNotFound
}

func (m *mockWebSearchCredentialStore) StoreToken(token *oauth2.Token) error {
	m.hasToken = true
	return nil
}

func (m *mockWebSearchCredentialStore) ClearToken() error {
	m.hasToken = false
	return nil
}

func (m *mockWebSearchCredentialStore) HasToken() bool {
	return m.hasToken
}

func (m *mockWebSearchCredentialStore) GetStoragePath() string {
	return "/tmp/test-storage"
}

func TestNewWebSearcher(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: false,
		},
		{
			name:        "default config",
			config:      NewConfig(),
			expectError: false,
		},
		{
			name: "custom config",
			config: NewConfig(
				WithCredentialStore(&mockWebSearchCredentialStore{}),
			),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher, err := NewWebSearcher(tt.config)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && searcher == nil {
				t.Error("Expected searcher to be created but got nil")
			}

			if searcher != nil {
				// Test basic searcher structure
				if searcher.config == nil {
					t.Error("Searcher config should not be nil")
				}
				if searcher.auth == nil {
					t.Error("Searcher auth should not be nil")
				}
				if searcher.codeAssist == nil {
					t.Error("Searcher codeAssist should not be nil")
				}
				if searcher.grounding == nil {
					t.Error("Searcher grounding should not be nil")
				}
			}
		})
	}
}

func TestWebSearcherIsAuthenticated(t *testing.T) {
	// Create searcher with mock credential store
	store := &mockWebSearchCredentialStore{hasToken: false}
	searcher, err := NewWebSearcher(NewConfig(WithCredentialStore(store)))
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}

	// Should not be authenticated initially
	if searcher.IsAuthenticated() {
		t.Error("Searcher should not be authenticated initially")
	}

	// Simulate authentication
	store.hasToken = true

	// Note: The actual IsAuthenticated method depends on the OAuth2 implementation
	// and may not immediately reflect the store state without token validation
	// This test mainly verifies the method exists and doesn't panic
}

func TestWebSearcherGetAuthStatus(t *testing.T) {
	searcher, err := NewWebSearcher(nil)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}

	// Test that GetAuthStatus doesn't panic
	status, err := searcher.GetAuthStatus()
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

func TestWebSearcherClearAuthentication(t *testing.T) {
	searcher, err := NewWebSearcher(nil)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}

	// Test that ClearAuthentication doesn't panic
	err = searcher.ClearAuthentication()
	// We don't check the specific result since it depends on the OAuth2 implementation
	// This test mainly verifies the method exists and doesn't panic

	if err != nil {
		t.Logf("ClearAuthentication returned error: %v", err)
	}
}

func TestWebSearcherWithNilConfig(t *testing.T) {
	// Test that NewWebSearcher handles nil config gracefully
	searcher, err := NewWebSearcher(nil)
	if err != nil {
		t.Fatalf("NewWebSearcher with nil config should not fail: %v", err)
	}

	if searcher == nil {
		t.Fatal("NewWebSearcher should return a valid searcher even with nil config")
	}

	// Verify that default config was used
	if searcher.config == nil {
		t.Error("Searcher config should not be nil when created with nil config")
	}

	// Check that the config has expected default values
	if searcher.config.CodeAssistEndpoint == "" {
		t.Error("CodeAssistEndpoint should not be empty in default config")
	}
	if searcher.config.GeminiAPIEndpoint == "" {
		t.Error("GeminiAPIEndpoint should not be empty in default config")
	}
	if searcher.config.DefaultModel == "" {
		t.Error("DefaultModel should not be empty in default config")
	}
}

func TestWebSearcherComponents(t *testing.T) {
	searcher, err := NewWebSearcher(NewConfig())
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}

	// Test that all components are properly initialized
	if searcher.auth == nil {
		t.Error("Auth component should be initialized")
	}

	if searcher.codeAssist == nil {
		t.Error("CodeAssist component should be initialized")
	}

	if searcher.grounding == nil {
		t.Error("Grounding component should be initialized")
	}

	if searcher.config == nil {
		t.Error("Config should be initialized")
	}
}
