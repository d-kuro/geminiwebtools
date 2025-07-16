// Package browser provides browser-based OAuth2 authentication functionality
// compatible with gemini-cli implementation.
package browser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"golang.org/x/oauth2"
)

// AuthResult represents the result of browser authentication.
type AuthResult struct {
	Token *oauth2.Token
	Error error
}

// BrowserAuth handles OAuth2 browser authentication flow.
type BrowserAuth struct {
	config *oauth2.Config
	state  string
	server *http.Server
}

// NewBrowserAuth creates a new browser authentication handler.
func NewBrowserAuth(config *oauth2.Config) *BrowserAuth {
	state := generateState()
	return &BrowserAuth{
		config: config,
		state:  state,
	}
}

// Authenticate performs browser-based OAuth2 authentication.
// This matches the gemini-cli implementation.
func (ba *BrowserAuth) Authenticate(ctx context.Context) (*oauth2.Token, error) {
	// Find available port
	port, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Update redirect URI
	redirectURI := fmt.Sprintf("http://localhost:%d/oauth2callback", port)
	ba.config.RedirectURL = redirectURI

	// Generate auth URL
	authURL := ba.config.AuthCodeURL(ba.state, oauth2.AccessTypeOffline)

	// Create result channel
	resultChan := make(chan AuthResult, 1)

	// Start local HTTP server
	ba.startServer(port, resultChan)

	// Open browser
	fmt.Printf("\nGemini Web Tools authentication required.\n")
	fmt.Printf("Opening authentication page in your browser...\n")
	fmt.Printf("If the browser doesn't open automatically, visit:\n\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
		fmt.Printf("Please manually open the URL above.\n")
	}

	fmt.Println("Waiting for authentication...")

	// Wait for result
	select {
	case result := <-resultChan:
		ba.shutdown()
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Token, nil
	case <-ctx.Done():
		ba.shutdown()
		return nil, ctx.Err()
	case <-time.After(constants.AuthTimeout):
		ba.shutdown()
		return nil, fmt.Errorf("authentication timeout")
	}
}

// startServer starts the local HTTP server for OAuth callback.
func (ba *BrowserAuth) startServer(port int, resultChan chan<- AuthResult) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", ba.handleCallback(resultChan))

	ba.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := ba.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			resultChan <- AuthResult{Error: fmt.Errorf("server error: %w", err)}
		}
	}()
}

// handleCallback handles the OAuth2 callback.
func (ba *BrowserAuth) handleCallback(resultChan chan<- AuthResult) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		query := r.URL.Query()

		// Check for error
		if errMsg := query.Get("error"); errMsg != "" {
			resultChan <- AuthResult{Error: fmt.Errorf("authentication error: %s", errMsg)}
			http.Redirect(w, r, getFailureURL(), http.StatusFound)
			return
		}

		// Verify state parameter (CSRF protection)
		if state := query.Get("state"); state != ba.state {
			resultChan <- AuthResult{Error: fmt.Errorf("state mismatch, possible CSRF attack")}
			http.Error(w, "State mismatch. Possible CSRF attack", http.StatusBadRequest)
			return
		}

		// Get authorization code
		code := query.Get("code")
		if code == "" {
			resultChan <- AuthResult{Error: fmt.Errorf("no authorization code received")}
			http.Error(w, "No authorization code found", http.StatusBadRequest)
			return
		}

		// Exchange code for token
		token, err := ba.config.Exchange(context.Background(), code)
		if err != nil {
			resultChan <- AuthResult{Error: fmt.Errorf("failed to exchange token: %w", err)}
			http.Redirect(w, r, getFailureURL(), http.StatusFound)
			return
		}

		// Send success response
		resultChan <- AuthResult{Token: token}
		http.Redirect(w, r, getSuccessURL(), http.StatusFound)
	}
}

// shutdown gracefully shuts down the server.
func (ba *BrowserAuth) shutdown() {
	if ba.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), constants.ServerShutdownTimeout)
		defer cancel()
		_ = ba.server.Shutdown(ctx) // Ignore error during shutdown
	}
}

// getAvailablePort finds an available port for the local server.
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = listener.Close() // Ignore error during close
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// generateState generates a random state parameter for CSRF protection.
func generateState() string {
	bytes := make([]byte, constants.StateRandomBytes)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based state if crypto/rand fails
		return fmt.Sprintf("state_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string

	if commands, exists := constants.BrowserCommands[runtime.GOOS]; exists {
		cmd = commands[0]
		if len(commands) > 1 {
			args = commands[1:]
		}
	} else {
		// Fallback for unsupported OS
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// getSuccessURL returns the success URL to redirect to after authentication.
func getSuccessURL() string {
	return constants.AuthSuccessURL
}

// getFailureURL returns the failure URL to redirect to after authentication failure.
func getFailureURL() string {
	return constants.AuthFailureURL
}
