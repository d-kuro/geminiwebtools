// Package geminiwebtools provides a Go library for web search and web fetch functionality
// compatible with gemini-cli interfaces using OAuth2 authentication.
//
// This library extracts the WebSearch and WebFetch functionality from the claude-code-mcp server
// into a reusable Go library while maintaining compatibility with gemini-cli interfaces.
//
// Example usage:
//
//	// Create a client with default configuration
//	client, err := geminiwebtools.NewClient(nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Perform a web search
//	result, err := client.Search(ctx, "Go programming language", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result.DisplayText)
//
//	// Fetch web content
//	fetchResult, err := client.Fetch(ctx, "https://golang.org", "Summarize this page", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(fetchResult.DisplayText)
package geminiwebtools

import (
	"context"
	"fmt"

	"github.com/d-kuro/geminiwebtools/pkg/auth"
	"github.com/d-kuro/geminiwebtools/pkg/types"
)

// Client provides a unified interface for web search and web fetch operations.
type Client struct {
	auth     *auth.SharedAuthenticator
	searcher *WebSearcher
	fetcher  *WebFetcher
	config   *Config
}

// NewClient creates a new client with the provided configuration options.
// If no options are provided, default configuration will be used.
func NewClient(opts ...ConfigOption) (*Client, error) {
	config := NewConfig(opts...)

	// Create OAuth2 authenticator and wrap with shared authenticator
	oauth2Auth := auth.NewOAuth2Authenticator(config.OAuth2Config, config.CredentialStore)
	sharedAuth := auth.NewSharedAuthenticator(oauth2Auth)

	// Create web searcher
	searcher, err := NewWebSearcher(config)
	if err != nil {
		return nil, err
	}

	// Create web fetcher
	fetcher, err := NewWebFetcher(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		auth:     sharedAuth,
		searcher: searcher,
		fetcher:  fetcher,
		config:   config,
	}, nil
}

// Search performs a web search using the configured AI model.
// Follows gemini-cli interface: accepts a simple query string.
func (c *Client) Search(ctx context.Context, query string) (*types.WebSearchResult, error) {
	return c.searcher.Search(ctx, query)
}

// Fetch retrieves and processes web content using AI, with fallback to direct HTTP.
// Follows gemini-cli interface: accepts a prompt containing URLs and processing instructions.
func (c *Client) Fetch(ctx context.Context, prompt string) (*types.WebFetchResult, error) {
	return c.fetcher.Fetch(ctx, prompt)
}

// IsAuthenticated checks if the client has valid authentication.
func (c *Client) IsAuthenticated() bool {
	return c.auth.IsAuthenticated()
}

// GetAuthStatus returns the current authentication status.
func (c *Client) GetAuthStatus() (*auth.AuthStatus, error) {
	return c.auth.GetAuthStatus()
}

// AuthenticateWithBrowser performs browser-based OAuth2 authentication.
// This opens a browser window for user authentication and stores the resulting token.
// Compatible with gemini-cli authentication flow.
func (c *Client) AuthenticateWithBrowser(ctx context.Context) error {
	return c.auth.AuthenticateWithBrowser(ctx)
}

// ClearAuthentication removes stored authentication credentials.
func (c *Client) ClearAuthentication() error {
	return c.auth.ClearAuthentication()
}

// GetConfig returns the client configuration.
func (c *Client) GetConfig() *Config {
	return c.config
}

// NewClientWithBrowserAuth creates a new client and performs browser authentication.
// This is a convenience function for CLI-like usage that matches gemini-cli behavior.
func NewClientWithBrowserAuth(ctx context.Context, opts ...ConfigOption) (*Client, error) {
	client, err := NewClient(opts...)
	if err != nil {
		return nil, err
	}

	// Check if already authenticated
	if client.IsAuthenticated() {
		return client, nil
	}

	// Perform browser authentication
	err = client.AuthenticateWithBrowser(ctx)
	if err != nil {
		return nil, fmt.Errorf("browser authentication failed: %w", err)
	}

	return client, nil
}
