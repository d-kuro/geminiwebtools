package auth

import (
	"context"
	"testing"

	"golang.org/x/oauth2"

	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// Mock credential store for testing
type mockCredStore struct {
	hasToken bool
}

func (m *mockCredStore) LoadToken() (*oauth2.Token, error) {
	if m.hasToken {
		return &oauth2.Token{AccessToken: "test-token"}, nil
	}
	return nil, storage.ErrStorageNotFound
}

func (m *mockCredStore) StoreToken(token *oauth2.Token) error {
	m.hasToken = true
	return nil
}

func (m *mockCredStore) ClearToken() error {
	m.hasToken = false
	return nil
}

func (m *mockCredStore) HasToken() bool {
	return m.hasToken
}

func (m *mockCredStore) GetStoragePath() string {
	return "/tmp/test-storage"
}

func TestNewSharedAuthenticator(t *testing.T) {
	store := &mockCredStore{}
	config := OAuth2Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com",
		TokenURL:     "https://token.example.com",
		Scopes:       []string{"scope1", "scope2"},
	}

	oauth2Auth := NewOAuth2Authenticator(config, store)
	sharedAuth := NewSharedAuthenticator(oauth2Auth)

	if sharedAuth == nil {
		t.Fatal("NewSharedAuthenticator should not return nil")
	}

	if sharedAuth.oauth2Auth != oauth2Auth {
		t.Error("SharedAuthenticator should store the provided OAuth2 authenticator")
	}
}

func TestSharedAuthenticatorIsAuthenticated(t *testing.T) {
	tests := []struct {
		name           string
		hasToken       bool
		expectedResult bool
	}{
		{
			name:           "authenticated",
			hasToken:       true,
			expectedResult: true, // OAuth2 implementation validates token
		},
		{
			name:           "not authenticated",
			hasToken:       false,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCredStore{hasToken: tt.hasToken}
			config := OAuth2Config{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				AuthURL:      "https://auth.example.com",
				TokenURL:     "https://token.example.com",
				Scopes:       []string{"scope1", "scope2"},
			}

			oauth2Auth := NewOAuth2Authenticator(config, store)
			sharedAuth := NewSharedAuthenticator(oauth2Auth)

			result := sharedAuth.IsAuthenticated()
			if result != tt.expectedResult {
				t.Errorf("Expected IsAuthenticated to return %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestSharedAuthenticatorGetAuthStatus(t *testing.T) {
	store := &mockCredStore{}
	config := OAuth2Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com",
		TokenURL:     "https://token.example.com",
		Scopes:       []string{"scope1", "scope2"},
	}

	oauth2Auth := NewOAuth2Authenticator(config, store)
	sharedAuth := NewSharedAuthenticator(oauth2Auth)

	// Test that GetAuthStatus doesn't panic
	status, err := sharedAuth.GetAuthStatus()
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

func TestSharedAuthenticatorClearAuthentication(t *testing.T) {
	store := &mockCredStore{hasToken: true}
	config := OAuth2Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com",
		TokenURL:     "https://token.example.com",
		Scopes:       []string{"scope1", "scope2"},
	}

	oauth2Auth := NewOAuth2Authenticator(config, store)
	sharedAuth := NewSharedAuthenticator(oauth2Auth)

	// Test that ClearAuthentication doesn't panic
	err := sharedAuth.ClearAuthentication()
	// We don't check the specific result since it depends on the OAuth2 implementation
	// This test mainly verifies the method exists and doesn't panic

	if err != nil {
		t.Logf("ClearAuthentication returned error: %v", err)
	}

	// Verify that the store was cleared
	if store.hasToken {
		t.Error("Expected token to be cleared from store")
	}
}

func TestSharedAuthenticatorGetValidToken(t *testing.T) {
	ctx := context.Background()
	store := &mockCredStore{hasToken: true}
	config := OAuth2Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com",
		TokenURL:     "https://token.example.com",
		Scopes:       []string{"scope1", "scope2"},
	}

	oauth2Auth := NewOAuth2Authenticator(config, store)
	sharedAuth := NewSharedAuthenticator(oauth2Auth)

	// Test that GetValidToken doesn't panic
	token, err := sharedAuth.GetValidToken(ctx)
	// We don't check the specific result since it depends on the OAuth2 implementation
	// This test mainly verifies the method exists and doesn't panic

	if err != nil {
		t.Logf("GetValidToken returned error: %v", err)
	}

	if token != nil {
		t.Logf("GetValidToken returned token: %+v", token)
	}
}

func TestSharedAuthenticatorGetOAuth2Authenticator(t *testing.T) {
	store := &mockCredStore{}
	config := OAuth2Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://auth.example.com",
		TokenURL:     "https://token.example.com",
		Scopes:       []string{"scope1", "scope2"},
	}

	oauth2Auth := NewOAuth2Authenticator(config, store)
	sharedAuth := NewSharedAuthenticator(oauth2Auth)

	result := sharedAuth.GetOAuth2Authenticator()
	if result != oauth2Auth {
		t.Error("GetOAuth2Authenticator should return the original OAuth2 authenticator")
	}
}
