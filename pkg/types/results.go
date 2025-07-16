package types

// WebFetchResult represents the result of a web fetch operation.
// This structure is compatible with the gemini-cli ToolResult interface.
type WebFetchResult struct {
	// Summary provides a one-line summary of the action performed
	Summary string `json:"summary,omitempty"`

	// Content contains the processed content for LLM context
	Content string `json:"content"`

	// DisplayText contains formatted content for user display
	DisplayText string `json:"displayText"`

	// Sources contains source information if grounding is available
	Sources []GroundingChunk `json:"sources,omitempty"`

	// Metadata contains additional information about the fetch operation
	Metadata WebFetchMetadata `json:"metadata"`
}

// WebSearchResult represents the result of a web search operation.
// This structure extends the base result with search-specific information.
type WebSearchResult struct {
	// Summary provides a one-line summary of the search performed
	Summary string `json:"summary,omitempty"`

	// Content contains the processed search results for LLM context
	Content string `json:"content"`

	// DisplayText contains formatted search results for user display
	DisplayText string `json:"displayText"`

	// Sources contains the search result sources with citations
	Sources []GroundingChunk `json:"sources,omitempty"`

	// Metadata contains additional information about the search operation
	Metadata WebSearchMetadata `json:"metadata"`
}

// WebFetchMetadata contains metadata about a web fetch operation.
type WebFetchMetadata struct {
	// URL is the original URL that was fetched
	URL string `json:"url"`

	// Prompt is the processing prompt that was applied
	Prompt string `json:"prompt"`

	// ContentType is the MIME type of the fetched content
	ContentType string `json:"contentType,omitempty"`

	// ContentSize is the size of the original content in bytes
	ContentSize int `json:"contentSize,omitempty"`

	// ProcessingTime is the time taken to process the request
	ProcessingTime string `json:"processingTime,omitempty"`

	// APIUsed indicates which API was used (codeassist, gemini, fallback)
	APIUsed string `json:"apiUsed"`

	// HasGrounding indicates if grounding metadata was available
	HasGrounding bool `json:"hasGrounding"`

	// SourceCount is the number of sources found
	SourceCount int `json:"sourceCount,omitempty"`

	// SupportCount is the number of grounding supports found
	SupportCount int `json:"supportCount,omitempty"`

	// UsedFallback indicates if fallback processing was used
	UsedFallback bool `json:"usedFallback,omitempty"`

	// Error contains error information if the operation failed
	Error string `json:"error,omitempty"`
}

// WebSearchMetadata contains metadata about a web search operation.
type WebSearchMetadata struct {
	// Query is the original search query
	Query string `json:"query"`

	// SearchRegion is the region where the search was performed
	SearchRegion string `json:"searchRegion,omitempty"`

	// AllowedDomains are the domains that were allowed in results
	AllowedDomains []string `json:"allowedDomains,omitempty"`

	// BlockedDomains are the domains that were blocked from results
	BlockedDomains []string `json:"blockedDomains,omitempty"`

	// ProcessingTime is the time taken to process the search
	ProcessingTime string `json:"processingTime,omitempty"`

	// APIUsed indicates which API was used (codeassist, gemini, fallback)
	APIUsed string `json:"apiUsed"`

	// HasGrounding indicates if grounding metadata was available
	HasGrounding bool `json:"hasGrounding"`

	// SourceCount is the number of sources found
	SourceCount int `json:"sourceCount,omitempty"`

	// SupportCount is the number of grounding supports found
	SupportCount int `json:"supportCount,omitempty"`

	// WebSearchQueries are the actual search queries used by the AI
	WebSearchQueries []string `json:"webSearchQueries,omitempty"`

	// Error contains error information if the search failed
	Error string `json:"error,omitempty"`
}

// SearchOptions contains options for web search operations.
type SearchOptions struct {
	// AllowedDomains restricts search results to these domains
	AllowedDomains []string `json:"allowedDomains,omitempty"`

	// BlockedDomains excludes these domains from search results
	BlockedDomains []string `json:"blockedDomains,omitempty"`

	// MaxResults limits the number of search results (if supported)
	MaxResults int `json:"maxResults,omitempty"`

	// SearchRegion specifies the region for search (if supported)
	SearchRegion string `json:"searchRegion,omitempty"`
}

// FetchOptions contains options for web fetch operations.
type FetchOptions struct {
	// ConvertHTML specifies whether to convert HTML to markdown
	ConvertHTML bool `json:"convertHtml,omitempty"`

	// TruncateContent specifies whether to truncate long content
	TruncateContent bool `json:"truncateContent,omitempty"`

	// TruncateLength specifies the maximum content length
	TruncateLength int `json:"truncateLength,omitempty"`

	// AllowPrivateIPs specifies whether to allow private IP addresses
	AllowPrivateIPs bool `json:"allowPrivateIps,omitempty"`

	// FollowRedirects specifies whether to follow HTTP redirects
	FollowRedirects bool `json:"followRedirects,omitempty"`

	// EnableFallback specifies whether to use fallback processing
	EnableFallback bool `json:"enableFallback,omitempty"`
}

// ErrorResult represents an error result from web operations.
type ErrorResult struct {
	// Operation is the operation that failed (fetch, search)
	Operation string `json:"operation"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// Code is a machine-readable error code
	Code string `json:"code,omitempty"`

	// Details contains additional error details
	Details map[string]any `json:"details,omitempty"`
}
