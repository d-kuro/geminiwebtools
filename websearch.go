// Package geminiwebtools provides web search and web fetch functionality
// compatible with gemini-cli interfaces using OAuth2 authentication.
package geminiwebtools

import (
	"context"
	"fmt"
	"time"

	"github.com/d-kuro/geminiwebtools/pkg/auth"
	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/types"
)

// WebSearcher provides web search functionality using Google's AI with OAuth2 authentication.
type WebSearcher struct {
	config     *Config
	auth       *auth.SharedAuthenticator
	codeAssist *auth.CodeAssistClient
	grounding  *GroundingProcessor
}

// NewWebSearcher creates a new web searcher with the provided configuration.
func NewWebSearcher(config *Config) (*WebSearcher, error) {
	if config == nil {
		config = NewConfig()
	}

	// Create OAuth2 authenticator and wrap with shared authenticator
	oauth2Auth := auth.NewOAuth2Authenticator(config.OAuth2Config, config.CredentialStore)
	sharedAuth := auth.NewSharedAuthenticator(oauth2Auth)

	// Create CodeAssist client
	codeAssist := auth.NewCodeAssistClient(
		oauth2Auth,
		config.CodeAssistEndpoint,
		config.DefaultModel,
	)

	// Create grounding processor
	grounding := NewGroundingProcessor()

	return &WebSearcher{
		config:     config,
		auth:       sharedAuth,
		codeAssist: codeAssist,
		grounding:  grounding,
	}, nil
}

// Search performs a web search using the configured AI model and returns processed results.
// Follows gemini-cli interface: accepts a simple query string.
func (ws *WebSearcher) Search(ctx context.Context, query string) (*types.WebSearchResult, error) {
	startTime := time.Now()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create search request
	req := ws.codeAssist.CreateSearchRequest(query)

	// Create a timeout context for the search request
	searchCtx, cancel := context.WithTimeout(ctx, constants.AIRequestTimeout)
	defer cancel()

	// Use a channel to handle the response and enable proper cancellation
	type searchResult struct {
		resp *types.GenerateContentResponse
		err  error
	}

	resultChan := make(chan searchResult, 1)

	// Run the search request in a goroutine to allow for cancellation
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- searchResult{nil, fmt.Errorf("panic in search request: %v", r)}
			}
		}()

		resp, err := ws.codeAssist.GenerateContent(searchCtx, req)
		select {
		case resultChan <- searchResult{resp, err}:
		case <-searchCtx.Done():
			// Context was cancelled, don't send result
		}
	}()

	// Wait for either result or cancellation
	select {
	case res := <-resultChan:
		if res.err != nil {
			return &types.WebSearchResult{
				Summary:     fmt.Sprintf("Search failed: %s", query),
				Content:     "",
				DisplayText: fmt.Sprintf("Error performing search: %v", res.err),
				Metadata: types.WebSearchMetadata{
					Query:          query,
					ProcessingTime: time.Since(startTime).String(),
					APIUsed:        "codeassist",
					HasGrounding:   false,
					Error:          res.err.Error(),
				},
			}, fmt.Errorf("web search failed: %w", res.err)
		}

		// Process the response
		return ws.processSearchResponse(res.resp, query, startTime)

	case <-searchCtx.Done():
		return &types.WebSearchResult{
			Summary:     fmt.Sprintf("Search timeout: %s", query),
			Content:     "",
			DisplayText: "Error: Search request timed out or was cancelled",
			Metadata: types.WebSearchMetadata{
				Query:          query,
				ProcessingTime: time.Since(startTime).String(),
				APIUsed:        "codeassist",
				HasGrounding:   false,
				Error:          searchCtx.Err().Error(),
			},
		}, searchCtx.Err()
	}
}

// IsAuthenticated checks if the searcher has valid authentication.
func (ws *WebSearcher) IsAuthenticated() bool {
	return ws.auth.IsAuthenticated()
}

// GetAuthStatus returns the current authentication status.
func (ws *WebSearcher) GetAuthStatus() (*auth.AuthStatus, error) {
	return ws.auth.GetAuthStatus()
}

// AuthenticateWithBrowser performs browser-based OAuth2 authentication.
// This opens a browser window for user authentication and stores the resulting token.
// Compatible with gemini-cli authentication flow.
func (ws *WebSearcher) AuthenticateWithBrowser(ctx context.Context) error {
	return ws.auth.AuthenticateWithBrowser(ctx)
}

// ClearAuthentication removes stored authentication credentials.
func (ws *WebSearcher) ClearAuthentication() error {
	return ws.auth.ClearAuthentication()
}

// processSearchResponse processes the AI response into a structured search result.
func (ws *WebSearcher) processSearchResponse(resp *types.GenerateContentResponse, query string, startTime time.Time) (*types.WebSearchResult, error) {
	result := &types.WebSearchResult{
		Summary: fmt.Sprintf("Web search for: %s", query),
		Metadata: types.WebSearchMetadata{
			Query:          query,
			ProcessingTime: time.Since(startTime).String(),
			APIUsed:        "codeassist",
		},
	}

	// No options processing needed for simplified interface

	// Extract content from the first candidate
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]

		// Build content from parts
		var contentBuilder, displayBuilder string
		for _, part := range candidate.Content.Parts {
			contentBuilder += part.Text
			displayBuilder += part.Text
		}

		result.Content = contentBuilder
		result.DisplayText = displayBuilder

		// Process grounding metadata if available
		if candidate.GroundingMetadata != nil {
			result.Metadata.HasGrounding = true
			result.Metadata.WebSearchQueries = candidate.GroundingMetadata.WebSearchQueries

			// Process grounding chunks as sources
			if len(candidate.GroundingMetadata.GroundingChunks) > 0 {
				result.Sources = candidate.GroundingMetadata.GroundingChunks
				result.Metadata.SourceCount = len(candidate.GroundingMetadata.GroundingChunks)
			}

			// Count grounding supports
			if len(candidate.GroundingMetadata.GroundingSupports) > 0 {
				result.Metadata.SupportCount = len(candidate.GroundingMetadata.GroundingSupports)
			}

			// Apply grounding processing for better formatting
			if ws.grounding != nil {
				processed := ws.grounding.ProcessGrounding(result.DisplayText, candidate.GroundingMetadata)
				result.DisplayText = processed
			}
		}
	}

	return result, nil
}
