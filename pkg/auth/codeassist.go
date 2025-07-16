package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/types"
)

// CodeAssistClient provides access to Google's Code Assist Server with OAuth2 authentication.
type CodeAssistClient struct {
	auth       *OAuth2Authenticator
	baseURL    string
	apiVersion string
	model      string
	projectID  string
	httpClient *http.Client
}

// NewCodeAssistClient creates a new CodeAssist client with optimized HTTP settings.
func NewCodeAssistClient(auth *OAuth2Authenticator, baseURL, model string) *CodeAssistClient {
	// Create optimized HTTP client with connection pooling for API calls
	transport := &http.Transport{
		// API-specific connection pooling settings
		MaxIdleConns:        constants.APIMaxIdleConns,
		MaxIdleConnsPerHost: constants.APIMaxIdleConnsPerHost,
		MaxConnsPerHost:     constants.APIMaxConnsPerHost,
		IdleConnTimeout:     constants.APIIdleConnTimeout,

		// Timeouts optimized for API calls
		DialContext: (&net.Dialer{
			Timeout:   constants.DefaultDialerTimeout,
			KeepAlive: constants.KeepAliveTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   constants.TLSHandshakeTimeout,
		ResponseHeaderTimeout: constants.ResponseHeaderTimeout,
		ExpectContinueTimeout: constants.ExpectContinueTimeout,

		// Enable HTTP/2 for better API performance
		ForceAttemptHTTP2: true,

		// Additional optimizations for API calls
		DisableKeepAlives:  false,     // Enable keep-alives for API connection reuse
		DisableCompression: false,     // Keep compression for smaller payloads
		WriteBufferSize:    16 * 1024, // 16KB write buffer (smaller for API)
		ReadBufferSize:     16 * 1024, // 16KB read buffer (smaller for API)
	}

	client := &http.Client{
		Timeout:   constants.DefaultHTTPTimeout,
		Transport: transport,
	}

	return &CodeAssistClient{
		auth:       auth,
		baseURL:    baseURL,
		apiVersion: constants.DefaultAPIVersion,
		model:      model,
		httpClient: client,
	}
}

// InitializeProject initializes the CodeAssist project if needed.
func (c *CodeAssistClient) InitializeProject(ctx context.Context) error {
	if c.projectID != "" {
		return nil // Already initialized
	}

	// Get authenticated HTTP client
	httpClient, err := c.auth.GetAuthenticatedClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// Load CodeAssist to get project ID
	loadReq := map[string]interface{}{
		"cloudaicompanionProject": nil,
		"metadata": map[string]string{
			"ideType":     "IDE_UNSPECIFIED",
			"platform":    "PLATFORM_UNSPECIFIED",
			"pluginType":  "GEMINI",
			"duetProject": "",
		},
	}

	loadResp, err := c.callAPI(ctx, httpClient, "loadCodeAssist", loadReq)
	if err != nil {
		return fmt.Errorf("failed to load code assist: %w", err)
	}

	// Extract project ID
	if projectID, ok := loadResp["cloudaicompanionProject"].(string); ok && projectID != "" {
		c.projectID = projectID
	} else {
		return fmt.Errorf("failed to get project ID from loadCodeAssist response")
	}

	// Onboard user
	onboardReq := map[string]interface{}{
		"tierId":                  constants.TierIDFree,
		"cloudaicompanionProject": c.projectID,
		"metadata": map[string]string{
			"ideType":     "IDE_UNSPECIFIED",
			"platform":    "PLATFORM_UNSPECIFIED",
			"pluginType":  "GEMINI",
			"duetProject": c.projectID,
		},
	}

	_, err = c.callAPI(ctx, httpClient, "onboardUser", onboardReq)
	if err != nil {
		return fmt.Errorf("failed to onboard user: %w", err)
	}

	return nil
}

// GenerateContent sends a content generation request to the CodeAssist Server.
func (c *CodeAssistClient) GenerateContent(ctx context.Context, req *types.GenerateContentRequest) (*types.GenerateContentResponse, error) {
	// Ensure project is initialized
	if err := c.InitializeProject(ctx); err != nil {
		return nil, err
	}

	// Get authenticated HTTP client
	httpClient, err := c.auth.GetAuthenticatedClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// Convert to CodeAssist format
	caReq := c.convertToCodeAssistRequest(req)

	// Make API call
	respData, err := c.callAPI(ctx, httpClient, "generateContent", caReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call generateContent: %w", err)
	}

	// Parse response
	var caResp types.CodeAssistGenerateContentResponse
	respBytes, err := json.Marshal(respData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(respBytes, &caResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert back to standard format
	return c.convertFromCodeAssistResponse(&caResp), nil
}

// CreateSearchRequest creates a request for web search.
func (c *CodeAssistClient) CreateSearchRequest(query string) *types.GenerateContentRequest {
	return &types.GenerateContentRequest{
		Contents: []types.Content{
			{
				Role: "user",
				Parts: []types.Part{
					{Text: query},
				},
			},
		},
		Tools: []types.Tool{
			{GoogleSearch: &types.GoogleSearchTool{}},
		},
	}
}

// CreateURLContextRequest creates a request for web fetch with URL context.
func (c *CodeAssistClient) CreateURLContextRequest(url, prompt string) *types.GenerateContentRequest {
	combinedPrompt := fmt.Sprintf("Please analyze the content from this URL: %s\n\nUser request: %s", url, prompt)

	return &types.GenerateContentRequest{
		Contents: []types.Content{
			{
				Role: "user",
				Parts: []types.Part{
					{Text: combinedPrompt},
				},
			},
		},
		Tools: []types.Tool{
			{URLContext: &types.URLContextTool{}},
		},
	}
}

// callAPI makes a generic API call to the CodeAssist Server.
func (c *CodeAssistClient) callAPI(ctx context.Context, httpClient *http.Client, method string, reqData interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s:%s", c.baseURL, c.apiVersion, method)

	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Check payload size limit
	if len(reqBytes) > constants.MaxAPIRequestSize {
		return nil, fmt.Errorf("request payload too large: %d bytes (max: %d)", len(reqBytes), constants.MaxAPIRequestSize)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", constants.ContentTypeJSON)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(reqBytes)))

	// Apply timeout to the request
	ctx, cancel := context.WithTimeout(ctx, constants.APIRequestTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("request timeout after %v", constants.APIRequestTimeout)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d %s", resp.StatusCode, resp.Status)
	}

	// Limit response body size
	limitedReader := io.LimitReader(resp.Body, constants.MaxAPIResponseSize)

	var result map[string]interface{}
	if err := json.NewDecoder(limitedReader).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// convertToCodeAssistRequest converts a standard request to CodeAssist format.
func (c *CodeAssistClient) convertToCodeAssistRequest(req *types.GenerateContentRequest) *types.CodeAssistGenerateContentRequest {
	// Pre-allocate slices with exact capacity to avoid reallocations
	caContents := make([]types.CodeAssistContent, 0, len(req.Contents))
	for _, content := range req.Contents {
		caParts := make([]types.CodeAssistPart, 0, len(content.Parts))
		for _, part := range content.Parts {
			caParts = append(caParts, types.CodeAssistPart(part))
		}
		caContents = append(caContents, types.CodeAssistContent{
			Role:  content.Role,
			Parts: caParts,
		})
	}

	caTools := make([]types.CodeAssistTool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		caTool := types.CodeAssistTool{}
		if tool.GoogleSearch != nil {
			caTool.GoogleSearch = &types.CodeAssistGoogleSearchTool{}
		}
		if tool.URLContext != nil {
			caTool.URLContext = &types.CodeAssistURLContextTool{}
		}
		caTools = append(caTools, caTool)
	}

	return &types.CodeAssistGenerateContentRequest{
		Model:   c.model,
		Project: c.projectID,
		Request: types.CodeAssistVertexContentRequest{
			Contents: caContents,
			Tools:    caTools,
		},
	}
}

// convertFromCodeAssistResponse converts a CodeAssist response to standard format.
func (c *CodeAssistClient) convertFromCodeAssistResponse(caResp *types.CodeAssistGenerateContentResponse) *types.GenerateContentResponse {
	// Pre-allocate candidates slice with exact capacity
	candidates := make([]types.Candidate, 0, len(caResp.Response.Candidates))
	for _, caCandidate := range caResp.Response.Candidates {
		// Pre-allocate parts slice with exact capacity
		parts := make([]types.CandidatePart, 0, len(caCandidate.Content.Parts))
		for _, caPart := range caCandidate.Content.Parts {
			parts = append(parts, types.CandidatePart(caPart))
		}

		candidate := types.Candidate{
			Content: types.CandidateContent{
				Role:  caCandidate.Content.Role,
				Parts: parts,
			},
			FinishReason: caCandidate.FinishReason,
			Index:        caCandidate.Index,
		}

		// Convert grounding metadata if present
		if caCandidate.GroundingMetadata != nil {
			candidate.GroundingMetadata = c.convertGroundingMetadata(caCandidate.GroundingMetadata)
		}

		candidates = append(candidates, candidate)
	}

	return &types.GenerateContentResponse{
		Candidates: candidates,
	}
}

// convertGroundingMetadata converts CodeAssist grounding metadata to standard format.
func (c *CodeAssistClient) convertGroundingMetadata(caMeta *types.CodeAssistGroundingMetadata) *types.GroundingMetadata {
	meta := &types.GroundingMetadata{
		WebSearchQueries:  caMeta.WebSearchQueries,
		RetrievalMetadata: caMeta.RetrievalMetadata,
	}

	if caMeta.SearchEntryPoint != nil {
		meta.SearchEntryPoint = &types.SearchEntryPoint{
			RenderedContent: caMeta.SearchEntryPoint.RenderedContent,
		}
	}

	// Convert grounding chunks with pre-allocated capacity
	if len(caMeta.GroundingChunks) > 0 {
		meta.GroundingChunks = make([]types.GroundingChunk, 0, len(caMeta.GroundingChunks))
		for _, caChunk := range caMeta.GroundingChunks {
			meta.GroundingChunks = append(meta.GroundingChunks, types.GroundingChunk{
				Web: struct {
					URI    string `json:"uri"`
					Title  string `json:"title"`
					Domain string `json:"domain,omitempty"`
				}{
					URI:   caChunk.Web.URI,
					Title: caChunk.Web.Title,
				},
			})
		}
	}

	// Convert grounding supports with pre-allocated capacity
	if len(caMeta.GroundingSupports) > 0 {
		meta.GroundingSupports = make([]types.GroundingSupport, 0, len(caMeta.GroundingSupports))
		for _, caSupport := range caMeta.GroundingSupports {
			meta.GroundingSupports = append(meta.GroundingSupports, types.GroundingSupport{
				Segment: struct {
					StartIndex int    `json:"startIndex"`
					EndIndex   int    `json:"endIndex"`
					Text       string `json:"text,omitempty"`
				}{
					StartIndex: caSupport.Segment.StartIndex,
					EndIndex:   caSupport.Segment.EndIndex,
					Text:       caSupport.Segment.Text,
				},
				GroundingChunkIndices: caSupport.GroundingChunkIndices,
			})
		}
	}

	return meta
}
