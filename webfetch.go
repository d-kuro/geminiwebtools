package geminiwebtools

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/d-kuro/geminiwebtools/pkg/auth"
	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/types"
)

// extractUrls extracts URLs from a string using regex
func extractUrls(text string) []string {
	urlRegex := regexp.MustCompile(constants.URLRegexPattern)
	return urlRegex.FindAllString(text, -1)
}

// validateURL performs comprehensive URL validation
func validateURL(urlStr string) error {
	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http/https allowed)", parsedURL.Scheme)
	}

	// Check host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL missing host")
	}

	// Check for suspicious characters in path
	if strings.ContainsAny(parsedURL.Path, "\x00\r\n") {
		return fmt.Errorf("URL contains invalid characters")
	}

	// Check for localhost/private IPs (basic check)
	host := strings.ToLower(parsedURL.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Check URL length
	if len(urlStr) > constants.MaxURLLength {
		return fmt.Errorf("URL exceeds maximum length of %d characters", constants.MaxURLLength)
	}

	return nil
}

// convertGitHubBlobURL converts GitHub blob URLs to raw URLs for direct access
// This matches the gemini-cli implementation
func convertGitHubBlobURL(url string) string {
	if strings.Contains(url, constants.GitHubDomain) && strings.Contains(url, constants.GitHubBlobPath) {
		// Convert GitHub blob URL to raw URL
		url = strings.Replace(url, constants.GitHubDomain, constants.GitHubRawDomain, 1)
		url = strings.Replace(url, constants.GitHubBlobPath, constants.GitHubRawPath, 1)
	}
	return url
}

// WebFetcher provides web content fetching functionality using Google's AI with OAuth2 authentication.
type WebFetcher struct {
	config     *Config
	auth       *auth.SharedAuthenticator
	codeAssist *auth.CodeAssistClient
	grounding  *GroundingProcessor
	httpClient *HTTPClient
}

// NewWebFetcher creates a new web fetcher with the provided configuration.
func NewWebFetcher(config *Config) (*WebFetcher, error) {
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

	// Create HTTP client for fallback
	httpClient := NewHTTPClient(&HTTPClientConfig{
		Timeout:         constants.DefaultHTTPTimeout,
		FollowRedirects: true,
		AllowPrivateIPs: false,
	})

	return &WebFetcher{
		config:     config,
		auth:       sharedAuth,
		codeAssist: codeAssist,
		grounding:  grounding,
		httpClient: httpClient,
	}, nil
}

// Fetch retrieves and processes web content using AI, with fallback to direct HTTP.
// Follows gemini-cli interface: accepts a prompt containing URLs and processing instructions.
func (wf *WebFetcher) Fetch(ctx context.Context, prompt string) (*types.WebFetchResult, error) {
	startTime := time.Now()

	// Extract URLs from prompt
	urls := extractUrls(prompt)
	if len(urls) == 0 {
		return &types.WebFetchResult{
			Summary:     "No URLs found in prompt",
			Content:     "",
			DisplayText: "Error: No URLs found in the prompt",
			Metadata: types.WebFetchMetadata{
				URL:            "",
				Prompt:         prompt,
				ProcessingTime: time.Since(startTime).String(),
				APIUsed:        "none",
				HasGrounding:   false,
				Error:          "No URLs found in prompt",
			},
		}, fmt.Errorf("no URLs found in prompt")
	}

	// Validate the first URL
	if err := validateURL(urls[0]); err != nil {
		return &types.WebFetchResult{
			Summary:     "Invalid URL",
			Content:     "",
			DisplayText: fmt.Sprintf("Error: %v", err),
			Metadata: types.WebFetchMetadata{
				URL:            urls[0],
				Prompt:         prompt,
				ProcessingTime: time.Since(startTime).String(),
				APIUsed:        "none",
				HasGrounding:   false,
				Error:          err.Error(),
			},
		}, err
	}

	// First try AI-powered fetch using CodeAssist
	result, err := wf.fetchWithAI(ctx, prompt, startTime)
	if err == nil {
		return result, nil
	}

	// If AI fetch fails, try direct HTTP fallback
	// Convert GitHub blob URL for fallback
	fallbackURL := convertGitHubBlobURL(urls[0])

	// Validate fallback URL if it's different
	if fallbackURL != urls[0] {
		if err := validateURL(fallbackURL); err != nil {
			return &types.WebFetchResult{
				Summary:     "Invalid fallback URL",
				Content:     "",
				DisplayText: fmt.Sprintf("Error: %v", err),
				Metadata: types.WebFetchMetadata{
					URL:            fallbackURL,
					Prompt:         prompt,
					ProcessingTime: time.Since(startTime).String(),
					APIUsed:        "none",
					HasGrounding:   false,
					Error:          err.Error(),
				},
			}, err
		}
	}

	return wf.fetchWithHTTP(ctx, fallbackURL, prompt, startTime)
}

// IsAuthenticated checks if the fetcher has valid authentication.
func (wf *WebFetcher) IsAuthenticated() bool {
	return wf.auth.IsAuthenticated()
}

// GetAuthStatus returns the current authentication status.
func (wf *WebFetcher) GetAuthStatus() (*auth.AuthStatus, error) {
	return wf.auth.GetAuthStatus()
}

// ClearAuthentication removes stored authentication credentials.
func (wf *WebFetcher) ClearAuthentication() error {
	return wf.auth.ClearAuthentication()
}

// fetchWithAI performs web fetch using the AI model with URLContext tool.
func (wf *WebFetcher) fetchWithAI(ctx context.Context, prompt string, startTime time.Time) (*types.WebFetchResult, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create URL context request
	req := wf.codeAssist.CreateURLContextRequest("", prompt)

	// Create a timeout context that respects the parent context cancellation
	timeoutCtx, cancel := context.WithTimeout(ctx, constants.AIRequestTimeout)
	defer cancel()

	// Use a channel to handle the response and enable proper cancellation
	type result struct {
		resp *types.GenerateContentResponse
		err  error
	}

	resultChan := make(chan result, 1)

	// Run the AI request in a goroutine to allow for cancellation
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- result{nil, fmt.Errorf("panic in AI request: %v", r)}
			}
		}()

		resp, err := wf.codeAssist.GenerateContent(timeoutCtx, req)
		select {
		case resultChan <- result{resp, err}:
		case <-timeoutCtx.Done():
			// Context was cancelled, don't send result
		}
	}()

	// Wait for either result or cancellation
	select {
	case res := <-resultChan:
		if res.err != nil {
			return &types.WebFetchResult{
				Summary:     "Fetch failed",
				Content:     "",
				DisplayText: fmt.Sprintf("Error fetching content: %v", res.err),
				Metadata: types.WebFetchMetadata{
					URL:            "",
					Prompt:         prompt,
					ProcessingTime: time.Since(startTime).String(),
					APIUsed:        "codeassist",
					HasGrounding:   false,
					Error:          res.err.Error(),
				},
			}, fmt.Errorf("web fetch failed: %w", res.err)
		}

		// Process the response
		return wf.processFetchResponse(res.resp, prompt, startTime, false)

	case <-timeoutCtx.Done():
		return &types.WebFetchResult{
			Summary:     "Fetch timeout",
			Content:     "",
			DisplayText: "Error: AI request timed out or was cancelled",
			Metadata: types.WebFetchMetadata{
				URL:            "",
				Prompt:         prompt,
				ProcessingTime: time.Since(startTime).String(),
				APIUsed:        "codeassist",
				HasGrounding:   false,
				Error:          timeoutCtx.Err().Error(),
			},
		}, timeoutCtx.Err()
	}
}

// fetchWithHTTP performs fallback web fetch using direct HTTP.
func (wf *WebFetcher) fetchWithHTTP(ctx context.Context, url, prompt string, startTime time.Time) (*types.WebFetchResult, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create a timeout context that respects the parent context cancellation
	timeoutCtx, cancel := context.WithTimeout(ctx, constants.HTTPFetchTimeout)
	defer cancel()

	// Use a channel to handle the response and enable proper cancellation
	type httpResult struct {
		content     string
		contentType string
		contentSize int
		err         error
	}

	resultChan := make(chan httpResult, 1)

	// Run the HTTP request in a goroutine to allow for cancellation
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- httpResult{"", "", 0, fmt.Errorf("panic in HTTP request: %v", r)}
			}
		}()

		content, contentType, contentSize, err := wf.httpClient.FetchContent(timeoutCtx, url)
		select {
		case resultChan <- httpResult{content, contentType, contentSize, err}:
		case <-timeoutCtx.Done():
			// Context was cancelled, don't send result
		}
	}()

	// Wait for either result or cancellation
	select {
	case res := <-resultChan:
		if res.err != nil {
			return &types.WebFetchResult{
				Summary:     fmt.Sprintf("HTTP fetch failed: %s", url),
				Content:     "",
				DisplayText: fmt.Sprintf("Error fetching content via HTTP: %v", res.err),
				Metadata: types.WebFetchMetadata{
					URL:            url,
					Prompt:         prompt,
					ProcessingTime: time.Since(startTime).String(),
					APIUsed:        "fallback",
					HasGrounding:   false,
					UsedFallback:   true,
					Error:          res.err.Error(),
				},
			}, fmt.Errorf("HTTP fetch failed: %w", res.err)
		}

		// Continue with successful response processing...
		return wf.processHTTPResponse(res.content, res.contentType, res.contentSize, url, prompt, startTime)

	case <-timeoutCtx.Done():
		return &types.WebFetchResult{
			Summary:     fmt.Sprintf("HTTP fetch timeout: %s", url),
			Content:     "",
			DisplayText: "Error: HTTP request timed out or was cancelled",
			Metadata: types.WebFetchMetadata{
				URL:            url,
				Prompt:         prompt,
				ProcessingTime: time.Since(startTime).String(),
				APIUsed:        "fallback",
				HasGrounding:   false,
				UsedFallback:   true,
				Error:          timeoutCtx.Err().Error(),
			},
		}, timeoutCtx.Err()
	}
}

// processHTTPResponse processes the successful HTTP response.
func (wf *WebFetcher) processHTTPResponse(content, contentType string, contentSize int, url, prompt string, startTime time.Time) (*types.WebFetchResult, error) {
	// Apply default content processing (use config defaults)
	processedContent := content
	if isHTMLContent(contentType) {
		processedContent = convertHTMLToMarkdown(content)
	}
	// Apply default truncation from config
	maxLength := constants.DefaultTruncateLength // Default from gemini-cli
	if len(processedContent) > maxLength {
		processedContent = processedContent[:maxLength] + "..."
	}

	// Create result with processed content
	displayText := processedContent
	if prompt != "" {
		displayText = fmt.Sprintf("Content from %s:\n\n%s\n\nUser request: %s", url, processedContent, prompt)
	}

	return &types.WebFetchResult{
		Summary:     fmt.Sprintf("Fetched content from: %s", url),
		Content:     processedContent,
		DisplayText: displayText,
		Metadata: types.WebFetchMetadata{
			URL:            url,
			Prompt:         prompt,
			ContentType:    contentType,
			ContentSize:    contentSize,
			ProcessingTime: time.Since(startTime).String(),
			APIUsed:        "fallback",
			HasGrounding:   false,
			UsedFallback:   true,
		},
	}, nil
}

// processFetchResponse processes the AI response into a structured fetch result.
func (wf *WebFetcher) processFetchResponse(resp *types.GenerateContentResponse, prompt string, startTime time.Time, usedFallback bool) (*types.WebFetchResult, error) {
	// Extract URLs from prompt for metadata
	urls := extractUrls(prompt)
	firstUrl := ""
	if len(urls) > 0 {
		firstUrl = urls[0]
	}

	result := &types.WebFetchResult{
		Summary: "Processed web content from prompt",
		Metadata: types.WebFetchMetadata{
			URL:            firstUrl,
			Prompt:         prompt,
			ProcessingTime: time.Since(startTime).String(),
			APIUsed:        "codeassist",
			UsedFallback:   usedFallback,
		},
	}

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
			if wf.grounding != nil {
				processed := wf.grounding.ProcessGrounding(result.DisplayText, candidate.GroundingMetadata)
				result.DisplayText = processed
			}
		}
	}

	return result, nil
}

// Helper functions

// isHTMLContent checks if the content type indicates HTML content.
func isHTMLContent(contentType string) bool {
	return contentType == constants.ContentTypeHTML || contentType == constants.ContentTypeXHTML
}

// convertHTMLToMarkdown converts HTML content to markdown format.
// This is a simplified implementation - in practice, you might want to use a proper HTML to Markdown converter.
func convertHTMLToMarkdown(htmlContent string) string {
	// For now, return as-is. In a real implementation, you would use a library like goquery or html2text
	// to properly convert HTML to markdown format.
	return htmlContent
}
