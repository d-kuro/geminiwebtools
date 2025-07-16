package geminiwebtools

import (
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/d-kuro/geminiwebtools/pkg/auth"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// Mock credential store for testing
type mockCredentialStore struct {
	shouldFailLoad bool
	shouldFailSave bool
	hasToken       bool
}

func (m *mockCredentialStore) LoadToken() (*oauth2.Token, error) {
	if m.shouldFailLoad {
		return nil, storage.ErrStorageNotFound
	}
	if m.hasToken {
		return &oauth2.Token{AccessToken: "test-token"}, nil
	}
	return nil, storage.ErrStorageNotFound
}

func (m *mockCredentialStore) StoreToken(token *oauth2.Token) error {
	if m.shouldFailSave {
		return storage.ErrStoragePermission
	}
	m.hasToken = true
	return nil
}

func (m *mockCredentialStore) ClearToken() error {
	m.hasToken = false
	return nil
}

func (m *mockCredentialStore) HasToken() bool {
	return m.hasToken
}

func (m *mockCredentialStore) GetStoragePath() string {
	return "/tmp/test-storage"
}

func TestNewConfig(t *testing.T) {
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
			name: "with max content size option",
			options: []ConfigOption{
				WithMaxContentSize(1024),
			},
			expectError: false,
		},
		{
			name: "with credential store option",
			options: []ConfigOption{
				WithCredentialStore(&mockCredentialStore{}),
			},
			expectError: false,
		},
		{
			name: "with multiple options",
			options: []ConfigOption{
				WithTimeout(30 * time.Second),
				WithMaxContentSize(1024),
				WithCredentialStore(&mockCredentialStore{}),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig(tt.options...)
			if config == nil {
				t.Fatal("NewConfig returned nil")
			}

			// Check that basic fields are set
			if config.CodeAssistEndpoint == "" {
				t.Error("CodeAssistEndpoint should not be empty")
			}
			if config.GeminiAPIEndpoint == "" {
				t.Error("GeminiAPIEndpoint should not be empty")
			}
			if config.DefaultModel == "" {
				t.Error("DefaultModel should not be empty")
			}
			if config.CredentialStore == nil {
				t.Error("CredentialStore should not be nil")
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 45 * time.Second
	config := NewConfig(WithTimeout(timeout))

	if config.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, config.Timeout)
	}
}

func TestWithMaxContentSize(t *testing.T) {
	maxSize := 2048
	config := NewConfig(WithMaxContentSize(maxSize))

	if config.MaxContentSize != maxSize {
		t.Errorf("Expected max content size %d, got %d", maxSize, config.MaxContentSize)
	}
}

func TestWithCredentialStore(t *testing.T) {
	store := &mockCredentialStore{}
	config := NewConfig(WithCredentialStore(store))

	if config.CredentialStore != store {
		t.Error("Expected credential store to be set correctly")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorField  string
	}{
		{
			name:        "valid config",
			config:      NewConfig(),
			expectError: false,
		},
		{
			name: "missing CodeAssistEndpoint",
			config: &Config{
				CodeAssistEndpoint: "",
				GeminiAPIEndpoint:  "https://api.gemini.com",
				OAuth2Config: auth.OAuth2Config{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
				CredentialStore: &mockCredentialStore{},
			},
			expectError: true,
			errorField:  "CodeAssistEndpoint",
		},
		{
			name: "missing GeminiAPIEndpoint",
			config: &Config{
				CodeAssistEndpoint: "https://codeassist.com",
				GeminiAPIEndpoint:  "",
				OAuth2Config: auth.OAuth2Config{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
				CredentialStore: &mockCredentialStore{},
			},
			expectError: true,
			errorField:  "GeminiAPIEndpoint",
		},
		{
			name: "missing OAuth2 ClientID",
			config: &Config{
				CodeAssistEndpoint: "https://codeassist.com",
				GeminiAPIEndpoint:  "https://api.gemini.com",
				OAuth2Config: auth.OAuth2Config{
					ClientID:     "",
					ClientSecret: "client-secret",
				},
				CredentialStore: &mockCredentialStore{},
			},
			expectError: true,
			errorField:  "OAuth2Config.ClientID",
		},
		{
			name: "missing OAuth2 ClientSecret",
			config: &Config{
				CodeAssistEndpoint: "https://codeassist.com",
				GeminiAPIEndpoint:  "https://api.gemini.com",
				OAuth2Config: auth.OAuth2Config{
					ClientID:     "client-id",
					ClientSecret: "",
				},
				CredentialStore: &mockCredentialStore{},
			},
			expectError: true,
			errorField:  "OAuth2Config.ClientSecret",
		},
		{
			name: "missing CredentialStore",
			config: &Config{
				CodeAssistEndpoint: "https://codeassist.com",
				GeminiAPIEndpoint:  "https://api.gemini.com",
				OAuth2Config: auth.OAuth2Config{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
				},
				CredentialStore: nil,
			},
			expectError: true,
			errorField:  "CredentialStore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}

			if tt.expectError && err != nil {
				configErr, ok := err.(*ConfigError)
				if !ok {
					t.Errorf("Expected ConfigError but got: %T", err)
				} else if configErr.Field != tt.errorField {
					t.Errorf("Expected error field %s but got %s", tt.errorField, configErr.Field)
				}
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "TestField",
		Message: "Test message",
	}

	expected := "config error in TestField: Test message"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Should have the same behavior as NewConfig()
	newConfig := NewConfig()

	if config.CodeAssistEndpoint != newConfig.CodeAssistEndpoint {
		t.Error("DefaultConfig should match NewConfig behavior")
	}
	if config.GeminiAPIEndpoint != newConfig.GeminiAPIEndpoint {
		t.Error("DefaultConfig should match NewConfig behavior")
	}
	if config.DefaultModel != newConfig.DefaultModel {
		t.Error("DefaultConfig should match NewConfig behavior")
	}
}
