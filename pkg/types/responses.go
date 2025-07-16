package types

// GenerateContentResponse represents a response from content generation.
type GenerateContentResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate represents a candidate response from the AI.
type Candidate struct {
	Content           CandidateContent   `json:"content"`
	FinishReason      string             `json:"finishReason"`
	Index             int                `json:"index"`
	SafetyRatings     []SafetyRating     `json:"safetyRatings,omitempty"`
	GroundingMetadata *GroundingMetadata `json:"groundingMetadata,omitempty"`
}

// CandidateContent represents the content of a candidate response.
type CandidateContent struct {
	Role  string          `json:"role"`
	Parts []CandidatePart `json:"parts"`
}

// CandidatePart represents a part of candidate content.
type CandidatePart struct {
	Text string `json:"text"`
}

// SafetyRating represents a safety rating for the response.
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GroundingMetadata represents grounding information for search results.
type GroundingMetadata struct {
	WebSearchQueries  []string           `json:"webSearchQueries,omitempty"`
	SearchEntryPoint  *SearchEntryPoint  `json:"searchEntryPoint,omitempty"`
	GroundingChunks   []GroundingChunk   `json:"groundingChunks,omitempty"`
	GroundingSupports []GroundingSupport `json:"groundingSupports,omitempty"`
	RetrievalMetadata map[string]any     `json:"retrievalMetadata,omitempty"`
}

// SearchEntryPoint represents the search entry point with visual elements.
type SearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
}

// GroundingChunk represents a source of grounding information.
type GroundingChunk struct {
	Web struct {
		URI    string `json:"uri"`
		Title  string `json:"title"`
		Domain string `json:"domain,omitempty"`
	} `json:"web"`
}

// GroundingSupport represents grounding support with segment information.
type GroundingSupport struct {
	Segment struct {
		StartIndex int    `json:"startIndex"`
		EndIndex   int    `json:"endIndex"`
		Text       string `json:"text,omitempty"`
	} `json:"segment"`
	GroundingChunkIndices []int `json:"groundingChunkIndices"`
}

// CodeAssist-specific response structures

// CodeAssistGenerateContentResponse represents a response from the CodeAssist Server.
type CodeAssistGenerateContentResponse struct {
	Response CodeAssistVertexContentResponse `json:"response"`
}

// CodeAssistVertexContentResponse represents the inner response from CodeAssist.
type CodeAssistVertexContentResponse struct {
	Candidates []CodeAssistCandidate `json:"candidates"`
}

// CodeAssistCandidate represents a candidate in CodeAssist format.
type CodeAssistCandidate struct {
	Content           CodeAssistCandidateContent   `json:"content"`
	FinishReason      string                       `json:"finishReason"`
	Index             int                          `json:"index"`
	SafetyRatings     []CodeAssistSafetyRating     `json:"safetyRatings,omitempty"`
	GroundingMetadata *CodeAssistGroundingMetadata `json:"groundingMetadata,omitempty"`
}

// CodeAssistCandidateContent represents content in CodeAssist format.
type CodeAssistCandidateContent struct {
	Role  string                    `json:"role"`
	Parts []CodeAssistCandidatePart `json:"parts"`
}

// CodeAssistCandidatePart represents a part in CodeAssist format.
type CodeAssistCandidatePart struct {
	Text string `json:"text"`
}

// CodeAssistSafetyRating represents safety rating in CodeAssist format.
type CodeAssistSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// CodeAssistGroundingMetadata represents grounding metadata in CodeAssist format.
type CodeAssistGroundingMetadata struct {
	WebSearchQueries  []string                     `json:"webSearchQueries,omitempty"`
	SearchEntryPoint  *CodeAssistSearchEntryPoint  `json:"searchEntryPoint,omitempty"`
	GroundingChunks   []CodeAssistGroundingChunk   `json:"groundingChunks,omitempty"`
	GroundingSupports []CodeAssistGroundingSupport `json:"groundingSupports,omitempty"`
	RetrievalMetadata map[string]any               `json:"retrievalMetadata,omitempty"`
}

// CodeAssistSearchEntryPoint represents search entry point in CodeAssist format.
type CodeAssistSearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
}

// CodeAssistGroundingChunk represents grounding chunk in CodeAssist format.
type CodeAssistGroundingChunk struct {
	Web struct {
		URI   string `json:"uri"`
		Title string `json:"title"`
	} `json:"web"`
}

// CodeAssistGroundingSupport represents grounding support in CodeAssist format.
type CodeAssistGroundingSupport struct {
	Segment struct {
		StartIndex int    `json:"startIndex"`
		EndIndex   int    `json:"endIndex"`
		Text       string `json:"text,omitempty"`
	} `json:"segment"`
	GroundingChunkIndices []int `json:"groundingChunkIndices"`
}
