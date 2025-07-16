// Package auth provides OAuth2 authentication compatible with gemini-cli.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/d-kuro/geminiwebtools/pkg/browser"
	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// OAuth2Authenticator provides OAuth2 authentication compatible with gemini-cli.
type OAuth2Authenticator struct {
	config *oauth2.Config
	store  storage.CredentialStore
}

// OAuth2Config holds OAuth2 authentication configuration.
type OAuth2Config struct {
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	AuthURL      string   `json:"authUrl,omitempty"`
	TokenURL     string   `json:"tokenUrl,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// NewOAuth2Authenticator creates a new OAuth2 authenticator.
func NewOAuth2Authenticator(oauth2Config OAuth2Config, store storage.CredentialStore) *OAuth2Authenticator {
	config := &oauth2.Config{
		ClientID:     oauth2Config.ClientID,
		ClientSecret: oauth2Config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oauth2Config.AuthURL,
			TokenURL: oauth2Config.TokenURL,
		},
		Scopes: oauth2Config.Scopes,
	}

	return &OAuth2Authenticator{
		config: config,
		store:  store,
	}
}

// NewGeminiCodeAssistAuthenticator creates an OAuth2 authenticator configured for Gemini Code Assist.
// This uses the default OAuth2 credentials and scopes compatible with gemini-cli.
func NewGeminiCodeAssistAuthenticator(store storage.CredentialStore) *OAuth2Authenticator {
	config := &oauth2.Config{
		ClientID:     constants.DefaultOAuthClientID,
		ClientSecret: constants.DefaultOAuthClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       constants.DefaultOAuthScopes,
	}

	return &OAuth2Authenticator{
		config: config,
		store:  store,
	}
}

// GetAuthStatus checks the current authentication status.
func (auth *OAuth2Authenticator) GetAuthStatus() (*AuthStatus, error) {
	token, err := auth.store.LoadToken()
	if err != nil {
		return &AuthStatus{
			Authenticated: false,
			Error:         err.Error(),
		}, nil
	}

	if token == nil {
		return &AuthStatus{
			Authenticated: false,
			Error:         "no token stored",
		}, nil
	}

	status := &AuthStatus{
		Authenticated:   true,
		TokenType:       token.TokenType,
		HasRefreshToken: token.RefreshToken != "",
		StoragePath:     auth.store.GetStoragePath(),
	}

	if !token.Expiry.IsZero() {
		status.ExpiresAt = token.Expiry
		status.ExpiresIn = time.Until(token.Expiry)
		status.IsExpired = token.Expiry.Before(time.Now())
	}

	return status, nil
}

// IsAuthenticated checks if a valid token is available.
func (auth *OAuth2Authenticator) IsAuthenticated() bool {
	status, err := auth.GetAuthStatus()
	return err == nil && status.Authenticated && !status.IsExpired
}

// GetValidToken returns a valid OAuth2 token, refreshing if necessary.
func (auth *OAuth2Authenticator) GetValidToken(ctx context.Context) (*oauth2.Token, error) {
	token, err := auth.store.LoadToken()
	if err != nil {
		return nil, &AuthError{
			Op:      "load_token",
			Message: "failed to load stored token",
			Err:     err,
		}
	}

	if token == nil {
		return nil, &AuthError{
			Op:      "load_token",
			Message: "no token stored - authentication required",
		}
	}

	// Validate token structure
	if err := validateTokenStructure(token); err != nil {
		return nil, &AuthError{
			Op:      "validate_token",
			Message: "token validation failed",
			Err:     err,
		}
	}

	// Check if token is expired or needs refresh
	if IsTokenExpired(token) {
		if token.RefreshToken == "" {
			return nil, &AuthError{
				Op:      "refresh_token",
				Message: "token expired and no refresh token available - re-authentication required",
			}
		}

		refreshedToken, err := auth.RefreshToken(ctx, token)
		if err != nil {
			return nil, &AuthError{
				Op:      "refresh_token",
				Message: "failed to refresh expired token",
				Err:     err,
			}
		}
		token = refreshedToken
	}

	return token, nil
}

// RefreshToken refreshes an OAuth2 token and stores the new token.
func (auth *OAuth2Authenticator) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken == "" {
		return nil, &AuthError{
			Op:      "refresh_token",
			Message: "no refresh token available",
		}
	}

	// Add timeout to refresh operation
	ctx, cancel := context.WithTimeout(ctx, constants.TokenRefreshTimeout)
	defer cancel()

	tokenSource := auth.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &AuthError{
				Op:      "refresh_token",
				Message: "token refresh timeout",
				Err:     err,
			}
		}
		return nil, &AuthError{
			Op:      "refresh_token",
			Message: "failed to refresh token",
			Err:     err,
		}
	}

	// Validate the new token
	if err := validateTokenStructure(newToken); err != nil {
		return nil, &AuthError{
			Op:      "validate_refreshed_token",
			Message: "refreshed token validation failed",
			Err:     err,
		}
	}

	// Store the refreshed token
	if err := auth.store.StoreToken(newToken); err != nil {
		return nil, &AuthError{
			Op:      "store_token",
			Message: "failed to store refreshed token",
			Err:     err,
		}
	}

	return newToken, nil
}

// AuthenticateWithBrowser performs browser-based OAuth2 authentication flow.
// This opens a browser window for user authentication and stores the resulting token.
func (auth *OAuth2Authenticator) AuthenticateWithBrowser(ctx context.Context) error {
	browserAuth := browser.NewBrowserAuth(auth.config)

	token, err := browserAuth.Authenticate(ctx)
	if err != nil {
		return &AuthError{
			Op:      "browser_auth",
			Message: "browser authentication failed",
			Err:     err,
		}
	}

	// Store the token
	if err := auth.store.StoreToken(token); err != nil {
		return &AuthError{
			Op:      "store_token",
			Message: "failed to store authentication token",
			Err:     err,
		}
	}

	return nil
}

// GetAuthenticatedClient returns an HTTP client configured with OAuth2 authentication.
func (auth *OAuth2Authenticator) GetAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	token, err := auth.GetValidToken(ctx)
	if err != nil {
		return nil, err
	}

	client := auth.config.Client(ctx, token)
	return client, nil
}

// ClearAuthentication removes stored authentication credentials.
func (auth *OAuth2Authenticator) ClearAuthentication() error {
	if err := auth.store.ClearToken(); err != nil {
		return &AuthError{
			Op:      "clear_token",
			Message: "failed to clear stored token",
			Err:     err,
		}
	}
	return nil
}

// AuthStatus represents the current authentication status.
type AuthStatus struct {
	Authenticated   bool          `json:"authenticated"`
	TokenType       string        `json:"tokenType,omitempty"`
	ExpiresAt       time.Time     `json:"expiresAt,omitempty"`
	ExpiresIn       time.Duration `json:"expiresIn,omitempty"`
	IsExpired       bool          `json:"isExpired,omitempty"`
	HasRefreshToken bool          `json:"hasRefreshToken,omitempty"`
	StoragePath     string        `json:"storagePath,omitempty"`
	Error           string        `json:"error,omitempty"`
}

// AuthError represents an authentication error.
type AuthError struct {
	Op      string // The operation that failed
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("auth %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("auth %s: %s", e.Op, e.Message)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// IsTokenExpired checks if a token is expired or will expire soon.
func IsTokenExpired(token *oauth2.Token) bool {
	if token == nil || token.Expiry.IsZero() {
		return false
	}
	// Consider token expired if it expires within the threshold
	return token.Expiry.Before(time.Now().Add(constants.TokenRefreshThreshold))
}

// validateTokenStructure validates the structure and content of a token
func validateTokenStructure(token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if token.AccessToken == "" {
		return fmt.Errorf("access token is empty")
	}

	// Check token format (basic validation)
	if len(token.AccessToken) < constants.MinTokenLength {
		return fmt.Errorf("access token too short")
	}

	if len(token.AccessToken) > constants.MaxTokenLength {
		return fmt.Errorf("access token too long")
	}

	// Check if token contains suspicious characters
	if strings.ContainsAny(token.AccessToken, "\x00\r\n") {
		return fmt.Errorf("access token contains invalid characters")
	}

	// If refresh token exists, validate it too
	if token.RefreshToken != "" {
		if len(token.RefreshToken) < constants.MinTokenLength {
			return fmt.Errorf("refresh token too short")
		}
		if len(token.RefreshToken) > constants.MaxTokenLength {
			return fmt.Errorf("refresh token too long")
		}
		if strings.ContainsAny(token.RefreshToken, "\x00\r\n") {
			return fmt.Errorf("refresh token contains invalid characters")
		}
	}

	return nil
}
