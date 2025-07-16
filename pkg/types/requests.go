// Package types provides common data structures for web tools,
// compatible with both Gemini API and CodeAssist Server formats.
package types

// GenerateContentRequest represents a request to generate content using AI.
// This structure is compatible with both direct Gemini API and CodeAssist Server calls.
type GenerateContentRequest struct {
	Contents []Content `json:"contents"`
	Tools    []Tool    `json:"tools,omitempty"`
}

// Content represents a piece of content in a conversation.
type Content struct {
	Role  string `json:"role"`  // "user" or "model"
	Parts []Part `json:"parts"` // Array of content parts
}

// Part represents a single part of content (text, image, etc.).
type Part struct {
	Text string `json:"text"` // Text content
}

// Tool represents a tool configuration for AI requests.
type Tool struct {
	GoogleSearch *GoogleSearchTool `json:"googleSearch,omitempty"` // For web search
	URLContext   *URLContextTool   `json:"urlContext,omitempty"`   // For web fetch
}

// GoogleSearchTool represents the Google Search tool configuration.
// This tool enables web search functionality through the AI API.
type GoogleSearchTool struct {
	// Empty struct as the Google Search tool requires no configuration
}

// URLContextTool represents the URL Context tool configuration.
// This tool enables fetching and processing web content through the AI API.
type URLContextTool struct {
	// Empty struct as the URL Context tool requires no configuration
}

// CodeAssist-specific request structures (used internally)

// CodeAssistGenerateContentRequest represents a request to the CodeAssist Server.
type CodeAssistGenerateContentRequest struct {
	Model   string                         `json:"model"`
	Project string                         `json:"project,omitempty"`
	Request CodeAssistVertexContentRequest `json:"request"`
}

// CodeAssistVertexContentRequest represents the inner request for CodeAssist.
type CodeAssistVertexContentRequest struct {
	Contents  []CodeAssistContent `json:"contents"`
	Tools     []CodeAssistTool    `json:"tools,omitempty"`
	SessionID string              `json:"session_id,omitempty"`
}

// CodeAssistContent represents content in CodeAssist format.
type CodeAssistContent struct {
	Role  string           `json:"role"`
	Parts []CodeAssistPart `json:"parts"`
}

// CodeAssistPart represents a part in CodeAssist format.
type CodeAssistPart struct {
	Text string `json:"text"`
}

// CodeAssistTool represents a tool in CodeAssist format.
type CodeAssistTool struct {
	GoogleSearch *CodeAssistGoogleSearchTool `json:"googleSearch,omitempty"`
	URLContext   *CodeAssistURLContextTool   `json:"urlContext,omitempty"`
}

// CodeAssistGoogleSearchTool represents the Google Search tool in CodeAssist format.
type CodeAssistGoogleSearchTool struct {
	// Empty struct as the Google Search tool requires no configuration
}

// CodeAssistURLContextTool represents the URL Context tool in CodeAssist format.
type CodeAssistURLContextTool struct {
	// Empty struct as the URL Context tool requires no configuration
}
