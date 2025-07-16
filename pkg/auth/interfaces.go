// Package auth provides authentication interfaces and implementations for the geminiwebtools library.
package auth

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/d-kuro/geminiwebtools/pkg/types"
)

// Authenticatable defines the common authentication interface that all components requiring
// authentication must implement. This interface consolidates authentication behavior
// across WebSearcher, WebFetcher, and other components.
type Authenticatable interface {
	// IsAuthenticated checks if the component has valid authentication.
	IsAuthenticated() bool

	// GetAuthStatus returns the current authentication status with detailed information.
	GetAuthStatus() (*AuthStatus, error)

	// AuthenticateWithBrowser performs browser-based OAuth2 authentication.
	// This opens a browser window for user authentication and stores the resulting token.
	AuthenticateWithBrowser(ctx context.Context) error

	// ClearAuthentication removes stored authentication credentials.
	ClearAuthentication() error
}

// TokenProvider defines the interface for components that can provide OAuth2 tokens.
// This is used internally by components that need to make authenticated API calls.
type TokenProvider interface {
	// GetValidToken returns a valid OAuth2 token, refreshing if necessary.
	GetValidToken(ctx context.Context) (*oauth2.Token, error)

	// RefreshToken refreshes an OAuth2 token and stores the new token.
	RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error)

	// GetAuthenticatedClient returns an HTTP client configured with OAuth2 authentication.
	GetAuthenticatedClient(ctx context.Context) (*http.Client, error)
}

// WebSearchProvider defines the interface for components that provide web search functionality.
type WebSearchProvider interface {
	Authenticatable
	// Search performs a web search using the configured AI model.
	Search(ctx context.Context, query string) (*types.WebSearchResult, error)
}

// WebFetchProvider defines the interface for components that provide web fetch functionality.
type WebFetchProvider interface {
	Authenticatable
	// Fetch retrieves and processes web content using AI, with fallback to direct HTTP.
	Fetch(ctx context.Context, prompt string) (*types.WebFetchResult, error)
}

// AuthenticatorConfig defines the configuration interface for authenticators.
type AuthenticatorConfig interface {
	// GetClientID returns the OAuth2 client ID.
	GetClientID() string

	// GetClientSecret returns the OAuth2 client secret.
	GetClientSecret() string

	// GetScopes returns the OAuth2 scopes required.
	GetScopes() []string

	// GetAuthURL returns the OAuth2 authorization URL.
	GetAuthURL() string

	// GetTokenURL returns the OAuth2 token URL.
	GetTokenURL() string
}

// SharedAuthenticator provides a centralized authentication implementation that can be
// embedded in multiple components to share authentication state and behavior.
type SharedAuthenticator struct {
	oauth2Auth *OAuth2Authenticator
}

// NewSharedAuthenticator creates a new shared authenticator with the provided OAuth2 authenticator.
func NewSharedAuthenticator(oauth2Auth *OAuth2Authenticator) *SharedAuthenticator {
	return &SharedAuthenticator{
		oauth2Auth: oauth2Auth,
	}
}

// IsAuthenticated checks if the authenticator has valid authentication.
func (sa *SharedAuthenticator) IsAuthenticated() bool {
	return sa.oauth2Auth.IsAuthenticated()
}

// GetAuthStatus returns the current authentication status.
func (sa *SharedAuthenticator) GetAuthStatus() (*AuthStatus, error) {
	return sa.oauth2Auth.GetAuthStatus()
}

// AuthenticateWithBrowser performs browser-based OAuth2 authentication.
func (sa *SharedAuthenticator) AuthenticateWithBrowser(ctx context.Context) error {
	return sa.oauth2Auth.AuthenticateWithBrowser(ctx)
}

// ClearAuthentication removes stored authentication credentials.
func (sa *SharedAuthenticator) ClearAuthentication() error {
	return sa.oauth2Auth.ClearAuthentication()
}

// GetValidToken returns a valid OAuth2 token, refreshing if necessary.
func (sa *SharedAuthenticator) GetValidToken(ctx context.Context) (*oauth2.Token, error) {
	return sa.oauth2Auth.GetValidToken(ctx)
}

// RefreshToken refreshes an OAuth2 token and stores the new token.
func (sa *SharedAuthenticator) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	return sa.oauth2Auth.RefreshToken(ctx, token)
}

// GetAuthenticatedClient returns an HTTP client configured with OAuth2 authentication.
func (sa *SharedAuthenticator) GetAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	return sa.oauth2Auth.GetAuthenticatedClient(ctx)
}

// GetOAuth2Authenticator returns the underlying OAuth2 authenticator for advanced usage.
func (sa *SharedAuthenticator) GetOAuth2Authenticator() *OAuth2Authenticator {
	return sa.oauth2Auth
}
