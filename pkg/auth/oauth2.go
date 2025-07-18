// Package auth provides OAuth2 authentication compatible with gemini-cli.
//
// This package includes enterprise-grade enhanced automatic token refresh functionality
// with the following features:
//
// 1. Concurrent Access Protection:
//   - Thread-safe token refresh operations using mutexes
//   - Prevents race conditions in GetValidToken()
//
// 2. Background Refresh Strategy:
//   - Proactive token refresh when 50% through token lifetime (configurable)
//   - Configurable refresh intervals and thresholds
//   - Reduces likelihood of expired tokens during active usage
//
// 3. Retry Mechanisms:
//   - Exponential backoff for failed refresh attempts (max 3 retries by default)
//   - Intelligent error classification to determine retryable vs non-retryable errors
//   - Jitter to prevent thundering herd problems
//
// 4. Refresh State Tracking:
//   - Prevents duplicate concurrent refresh attempts
//   - Tracks refresh status, timing, and attempt counts
//   - Provides monitoring capabilities through GetRefreshState()
//
// 5. Enhanced Error Handling:
//   - Detailed error classification and recovery strategies
//   - Graceful fallback to existing token if refresh fails during grace period
//   - Comprehensive logging for debugging refresh issues
//
// All existing TokenProvider interface methods are preserved exactly for
// backward compatibility. Enhanced features are available through new methods
// on OAuth2Authenticator and SharedAuthenticator.
//
// Usage Example:
//
//	// Create with default configuration
//	auth := NewOAuth2Authenticator(store)
//
//	// Or create with custom refresh configuration
//	refreshConfig := &RefreshConfig{
//	    BackgroundRefreshThreshold: 0.7, // Refresh at 70% of lifetime
//	    RetryMaxAttempts: 5,
//	    RetryBaseDelay: 2 * time.Second,
//	}
//	auth := NewOAuth2AuthenticatorWithConfig(store, refreshConfig)
//
//	// Use normally - enhanced features work automatically
//	token, err := auth.GetValidToken(ctx)
//
//	// Monitor refresh state
//	state := auth.GetRefreshState()
//	fmt.Printf("Last refresh: %v, Attempts: %d", state.LastRefreshSuccess, state.RefreshAttempts)
//
//	// Clean shutdown (important for background goroutines)
//	defer auth.Shutdown()
package auth

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/d-kuro/geminiwebtools/pkg/browser"
	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// RefreshConfig holds configuration for the enhanced token refresh functionality.
type RefreshConfig struct {
	// BackgroundRefreshThreshold determines when to start background refresh (0.0-1.0)
	// 0.5 means refresh when token is 50% through its lifetime
	BackgroundRefreshThreshold float64

	// RetryMaxAttempts is the maximum number of retry attempts for failed refreshes
	RetryMaxAttempts int

	// RetryBaseDelay is the base delay for exponential backoff
	RetryBaseDelay time.Duration

	// RetryMaxDelay is the maximum delay between retry attempts
	RetryMaxDelay time.Duration

	// RetryMultiplier is the multiplier for exponential backoff
	RetryMultiplier float64

	// JitterPercent is the jitter percentage to avoid thundering herd (0.0-1.0)
	JitterPercent float64

	// GracePeriod is how long to keep using old token if refresh fails
	GracePeriod time.Duration

	// BackgroundRefreshInterval is the interval for checking background refresh needs
	BackgroundRefreshInterval time.Duration

	// RefreshLockTimeout is the timeout for acquiring refresh lock
	RefreshLockTimeout time.Duration
}

// DefaultRefreshConfig returns the default refresh configuration.
func DefaultRefreshConfig() *RefreshConfig {
	return &RefreshConfig{
		BackgroundRefreshThreshold: constants.BackgroundRefreshThreshold,
		RetryMaxAttempts:           constants.RefreshRetryMaxAttempts,
		RetryBaseDelay:             constants.RefreshRetryBaseDelay,
		RetryMaxDelay:              constants.RefreshRetryMaxDelay,
		RetryMultiplier:            constants.RefreshRetryMultiplier,
		JitterPercent:              constants.RefreshJitterPercent,
		GracePeriod:                constants.RefreshGracePeriod,
		BackgroundRefreshInterval:  constants.BackgroundRefreshInterval,
		RefreshLockTimeout:         constants.RefreshLockTimeout,
	}
}

// RefreshState tracks the current state of token refresh operations.
type RefreshState struct {
	// IsRefreshing indicates if a refresh is currently in progress
	IsRefreshing bool

	// LastRefreshAttempt is the timestamp of the last refresh attempt
	LastRefreshAttempt time.Time

	// LastRefreshSuccess is the timestamp of the last successful refresh
	LastRefreshSuccess time.Time

	// RefreshAttempts is the number of consecutive failed refresh attempts
	RefreshAttempts int

	// LastError is the last error encountered during refresh
	LastError error
}

// OAuth2Authenticator provides OAuth2 authentication compatible with gemini-cli.
// Enhanced with enterprise-grade reliability features including concurrent access protection,
// background refresh, retry mechanisms, and comprehensive error handling.
type OAuth2Authenticator struct {
	config        *oauth2.Config
	store         storage.CredentialStore
	refreshConfig *RefreshConfig

	// Concurrent access protection
	mu sync.RWMutex

	// Token refresh state tracking
	refreshState *RefreshState
	refreshMu    sync.Mutex

	// Background refresh management
	backgroundCtx    context.Context
	backgroundCancel context.CancelFunc
	backgroundWg     sync.WaitGroup

	// Cached token with its retrieval time
	cachedToken     *oauth2.Token
	cachedTokenTime time.Time
	cacheValidFor   time.Duration
}

// OAuth2Config holds OAuth2 authentication configuration.
type OAuth2Config struct {
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	AuthURL      string   `json:"authUrl,omitempty"`
	TokenURL     string   `json:"tokenUrl,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// NewOAuth2Authenticator creates a new OAuth2 authenticator with default refresh configuration.
func NewOAuth2Authenticator(oauth2Config OAuth2Config, store storage.CredentialStore) *OAuth2Authenticator {
	return NewOAuth2AuthenticatorWithConfig(oauth2Config, store, DefaultRefreshConfig())
}

// NewOAuth2AuthenticatorWithConfig creates a new OAuth2 authenticator with custom refresh configuration.
func NewOAuth2AuthenticatorWithConfig(oauth2Config OAuth2Config, store storage.CredentialStore, refreshConfig *RefreshConfig) *OAuth2Authenticator {
	config := &oauth2.Config{
		ClientID:     oauth2Config.ClientID,
		ClientSecret: oauth2Config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oauth2Config.AuthURL,
			TokenURL: oauth2Config.TokenURL,
		},
		Scopes: oauth2Config.Scopes,
	}

	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())

	auth := &OAuth2Authenticator{
		config:           config,
		store:            store,
		refreshConfig:    refreshConfig,
		refreshState:     &RefreshState{},
		backgroundCtx:    backgroundCtx,
		backgroundCancel: backgroundCancel,
		cacheValidFor:    1 * time.Minute, // Cache tokens for 1 minute to reduce storage I/O
	}

	// Start background refresh goroutine
	auth.startBackgroundRefresh()

	return auth
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
// Enhanced with concurrent access protection, caching, and comprehensive error handling.
func (auth *OAuth2Authenticator) GetValidToken(ctx context.Context) (*oauth2.Token, error) {
	// First check cache with read lock
	auth.mu.RLock()
	if auth.cachedToken != nil && time.Since(auth.cachedTokenTime) < auth.cacheValidFor {
		if !IsTokenExpired(auth.cachedToken) {
			token := auth.cachedToken
			auth.mu.RUnlock()
			return token, nil
		}
	}
	auth.mu.RUnlock()

	// Acquire write lock for token operations
	auth.mu.Lock()
	defer auth.mu.Unlock()

	// Double-check cache after acquiring write lock
	if auth.cachedToken != nil && time.Since(auth.cachedTokenTime) < auth.cacheValidFor {
		if !IsTokenExpired(auth.cachedToken) {
			return auth.cachedToken, nil
		}
	}

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

		refreshedToken, err := auth.refreshTokenWithRetry(ctx, token)
		if err != nil {
			// Check if we can use the old token during grace period
			if auth.canUseTokenDuringGracePeriod(token) {
				log.Printf("Warning: Using expired token during grace period due to refresh failure: %v", err)
				auth.updateCache(token)
				return token, nil
			}
			return nil, &AuthError{
				Op:      "refresh_token",
				Message: "failed to refresh expired token and grace period exceeded",
				Err:     err,
			}
		}
		token = refreshedToken
	}

	// Update cache
	auth.updateCache(token)

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
	// Clear the stored token
	if err := auth.store.ClearToken(); err != nil {
		return &AuthError{
			Op:      "clear_token",
			Message: "failed to clear stored token",
			Err:     err,
		}
	}

	// Clear the cache
	auth.mu.Lock()
	auth.cachedToken = nil
	auth.cachedTokenTime = time.Time{}
	auth.mu.Unlock()

	// Reset refresh state
	auth.refreshMu.Lock()
	auth.refreshState = &RefreshState{}
	auth.refreshMu.Unlock()

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

// refreshTokenWithRetry performs token refresh with exponential backoff retry logic.
func (auth *OAuth2Authenticator) refreshTokenWithRetry(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	// Check if already refreshing
	auth.refreshMu.Lock()
	if auth.refreshState.IsRefreshing {
		auth.refreshMu.Unlock()
		// Wait for ongoing refresh to complete with timeout
		return auth.waitForRefresh(ctx)
	}

	// Mark as refreshing
	auth.refreshState.IsRefreshing = true
	auth.refreshState.LastRefreshAttempt = time.Now()
	auth.refreshMu.Unlock()

	defer func() {
		auth.refreshMu.Lock()
		auth.refreshState.IsRefreshing = false
		auth.refreshMu.Unlock()
	}()

	var lastErr error
	for attempt := 0; attempt < auth.refreshConfig.RetryMaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff and jitter
			delay := auth.calculateBackoffDelay(attempt)
			log.Printf("Token refresh attempt %d failed, retrying in %v: %v", attempt, delay, lastErr)

			select {
			case <-time.After(delay):
				// Continue to retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		refreshedToken, err := auth.RefreshToken(ctx, token)
		if err == nil {
			auth.refreshMu.Lock()
			auth.refreshState.LastRefreshSuccess = time.Now()
			auth.refreshState.RefreshAttempts = 0
			auth.refreshState.LastError = nil
			auth.refreshMu.Unlock()
			return refreshedToken, nil
		}

		lastErr = err
		auth.refreshMu.Lock()
		auth.refreshState.RefreshAttempts++
		auth.refreshState.LastError = err
		auth.refreshMu.Unlock()

		// Check if this is a non-retryable error
		if !auth.isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("token refresh failed after %d attempts: %w", auth.refreshConfig.RetryMaxAttempts, lastErr)
}

// waitForRefresh waits for an ongoing refresh operation to complete.
func (auth *OAuth2Authenticator) waitForRefresh(ctx context.Context) (*oauth2.Token, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(auth.refreshConfig.RefreshLockTimeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for refresh to complete")
		case <-ticker.C:
			auth.refreshMu.Lock()
			if !auth.refreshState.IsRefreshing {
				// Refresh completed, try to get the token
				auth.refreshMu.Unlock()
				return auth.store.LoadToken()
			}
			auth.refreshMu.Unlock()
		}
	}
}

// calculateBackoffDelay calculates the delay for exponential backoff with jitter.
func (auth *OAuth2Authenticator) calculateBackoffDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * multiplier^attempt
	delay := float64(auth.refreshConfig.RetryBaseDelay) * math.Pow(auth.refreshConfig.RetryMultiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(auth.refreshConfig.RetryMaxDelay) {
		delay = float64(auth.refreshConfig.RetryMaxDelay)
	}

	// Add jitter to avoid thundering herd
	jitter := delay * auth.refreshConfig.JitterPercent * (rand.Float64()*2 - 1) // -jitter to +jitter
	finalDelay := time.Duration(delay + jitter)

	// Ensure minimum delay
	if finalDelay < auth.refreshConfig.RetryBaseDelay {
		finalDelay = auth.refreshConfig.RetryBaseDelay
	}

	return finalDelay
}

// isRetryableError determines if an error is retryable.
func (auth *OAuth2Authenticator) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())

	// Retry on network errors, timeouts, and temporary failures
	retryableErrors := []string{
		"timeout",
		"connection",
		"network",
		"temporary",
		"unavailable",
		"rate limit",
		"429", // Too Many Requests
		"500", // Internal Server Error
		"502", // Bad Gateway
		"503", // Service Unavailable
		"504", // Gateway Timeout
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errorStr, retryable) {
			return true
		}
	}

	return false
}

// canUseTokenDuringGracePeriod checks if we can use an expired token during grace period.
func (auth *OAuth2Authenticator) canUseTokenDuringGracePeriod(token *oauth2.Token) bool {
	if token == nil || token.Expiry.IsZero() {
		return false
	}

	timeSinceExpiry := time.Since(token.Expiry)
	return timeSinceExpiry <= auth.refreshConfig.GracePeriod
}

// updateCache updates the cached token and timestamp.
func (auth *OAuth2Authenticator) updateCache(token *oauth2.Token) {
	auth.cachedToken = token
	auth.cachedTokenTime = time.Now()
}

// startBackgroundRefresh starts the background token refresh goroutine.
func (auth *OAuth2Authenticator) startBackgroundRefresh() {
	auth.backgroundWg.Add(1)
	go func() {
		defer auth.backgroundWg.Done()
		auth.backgroundRefreshLoop()
	}()
}

// backgroundRefreshLoop runs the background refresh check loop.
func (auth *OAuth2Authenticator) backgroundRefreshLoop() {
	ticker := time.NewTicker(auth.refreshConfig.BackgroundRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-auth.backgroundCtx.Done():
			return
		case <-ticker.C:
			auth.checkAndRefreshToken()
		}
	}
}

// checkAndRefreshToken checks if a token needs background refresh and performs it.
func (auth *OAuth2Authenticator) checkAndRefreshToken() {
	// Use a short timeout for background operations
	ctx, cancel := context.WithTimeout(auth.backgroundCtx, 30*time.Second)
	defer cancel()

	token, err := auth.store.LoadToken()
	if err != nil || token == nil {
		return // No token to refresh
	}

	if auth.shouldBackgroundRefresh(token) {
		log.Printf("Starting background token refresh")
		_, err := auth.refreshTokenWithRetry(ctx, token)
		if err != nil {
			log.Printf("Background token refresh failed: %v", err)
		} else {
			log.Printf("Background token refresh completed successfully")
		}
	}
}

// shouldBackgroundRefresh determines if a token should be refreshed in the background.
func (auth *OAuth2Authenticator) shouldBackgroundRefresh(token *oauth2.Token) bool {
	if token == nil || token.Expiry.IsZero() || token.RefreshToken == "" {
		return false
	}

	// Check if already refreshing
	auth.refreshMu.Lock()
	isRefreshing := auth.refreshState.IsRefreshing
	auth.refreshMu.Unlock()

	if isRefreshing {
		return false
	}

	// Calculate token lifetime progress
	now := time.Now()

	// For OAuth2 tokens, we need to estimate the issue time
	// A typical token lifetime is 1 hour, so we estimate issue time
	estimatedLifetime := 1 * time.Hour
	estimatedIssueTime := token.Expiry.Add(-estimatedLifetime)

	// If the estimated issue time is in the future, use a different approach
	if estimatedIssueTime.After(now) {
		// Token was likely issued recently, use time until expiry
		timeUntilExpiry := token.Expiry.Sub(now)
		// Refresh when less than half the estimated lifetime remains
		return timeUntilExpiry <= estimatedLifetime/2
	}

	// Calculate how much of the token lifetime has been used
	timeUsed := now.Sub(estimatedIssueTime)
	lifetimeUsed := float64(timeUsed) / float64(estimatedLifetime)

	return lifetimeUsed >= auth.refreshConfig.BackgroundRefreshThreshold
}

// Shutdown gracefully shuts down the background refresh process.
func (auth *OAuth2Authenticator) Shutdown() {
	if auth.backgroundCancel != nil {
		auth.backgroundCancel()
	}
	auth.backgroundWg.Wait()
}

// GetRefreshState returns the current refresh state for monitoring.
func (auth *OAuth2Authenticator) GetRefreshState() *RefreshState {
	auth.refreshMu.Lock()
	defer auth.refreshMu.Unlock()

	// Return a copy to avoid race conditions
	return &RefreshState{
		IsRefreshing:       auth.refreshState.IsRefreshing,
		LastRefreshAttempt: auth.refreshState.LastRefreshAttempt,
		LastRefreshSuccess: auth.refreshState.LastRefreshSuccess,
		RefreshAttempts:    auth.refreshState.RefreshAttempts,
		LastError:          auth.refreshState.LastError,
	}
}

// SetRefreshConfig updates the refresh configuration.
func (auth *OAuth2Authenticator) SetRefreshConfig(config *RefreshConfig) {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	auth.refreshConfig = config
}

// GetRefreshConfig returns a copy of the current refresh configuration.
func (auth *OAuth2Authenticator) GetRefreshConfig() *RefreshConfig {
	auth.mu.RLock()
	defer auth.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &RefreshConfig{
		BackgroundRefreshThreshold: auth.refreshConfig.BackgroundRefreshThreshold,
		RetryMaxAttempts:           auth.refreshConfig.RetryMaxAttempts,
		RetryBaseDelay:             auth.refreshConfig.RetryBaseDelay,
		RetryMaxDelay:              auth.refreshConfig.RetryMaxDelay,
		RetryMultiplier:            auth.refreshConfig.RetryMultiplier,
		JitterPercent:              auth.refreshConfig.JitterPercent,
		GracePeriod:                auth.refreshConfig.GracePeriod,
		BackgroundRefreshInterval:  auth.refreshConfig.BackgroundRefreshInterval,
		RefreshLockTimeout:         auth.refreshConfig.RefreshLockTimeout,
	}
}
