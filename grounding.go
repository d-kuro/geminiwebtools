package geminiwebtools

import (
	"fmt"
	"strings"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/types"
)

// GroundingProcessor handles processing of grounding metadata to enhance search results with citations.
type GroundingProcessor struct {
	// Configuration for grounding processing
	includeCitations bool
	maxCitations     int
}

// NewGroundingProcessor creates a new grounding processor with default settings.
func NewGroundingProcessor() *GroundingProcessor {
	return &GroundingProcessor{
		includeCitations: true,
		maxCitations:     constants.DefaultMaxCitations,
	}
}

// ProcessGrounding processes grounding metadata and enhances the content with citations.
func (gp *GroundingProcessor) ProcessGrounding(content string, metadata *types.GroundingMetadata) string {
	if metadata == nil || !gp.includeCitations {
		return content
	}

	// Start with the original content
	enhancedContent := content

	// Add citations section if grounding chunks are available
	if len(metadata.GroundingChunks) > 0 {
		enhancedContent += gp.formatCitations(metadata.GroundingChunks)
	}

	// Add search queries information if available
	if len(metadata.WebSearchQueries) > 0 {
		enhancedContent += gp.formatSearchQueries(metadata.WebSearchQueries)
	}

	return enhancedContent
}

// formatCitations formats grounding chunks as a citations section.
func (gp *GroundingProcessor) formatCitations(chunks []types.GroundingChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	var citations strings.Builder
	citations.WriteString(constants.SourcesHeader)

	maxCitations := len(chunks)
	if gp.maxCitations > 0 && maxCitations > gp.maxCitations {
		maxCitations = gp.maxCitations
	}

	for i := 0; i < maxCitations; i++ {
		chunk := chunks[i]
		citations.WriteString(fmt.Sprintf("- [%s](%s)", chunk.Web.Title, chunk.Web.URI))
		if chunk.Web.Domain != "" {
			citations.WriteString(fmt.Sprintf(" (%s)", chunk.Web.Domain))
		}
		citations.WriteString("\n")
	}

	if len(chunks) > maxCitations {
		citations.WriteString(fmt.Sprintf(constants.MoreSourcesFormat, len(chunks)-maxCitations))
	}

	return citations.String()
}

// formatSearchQueries formats web search queries information.
func (gp *GroundingProcessor) formatSearchQueries(queries []string) string {
	if len(queries) == 0 {
		return ""
	}

	var queryInfo strings.Builder
	queryInfo.WriteString(constants.SearchQueriesHeader)

	for i, query := range queries {
		if i < constants.DefaultMaxQueryDisplay { // Limit to first 3 queries to avoid clutter
			queryInfo.WriteString(fmt.Sprintf("- %s\n", query))
		}
	}

	if len(queries) > constants.DefaultMaxQueryDisplay {
		queryInfo.WriteString(fmt.Sprintf(constants.MoreQueriesFormat, len(queries)-constants.DefaultMaxQueryDisplay))
	}

	return queryInfo.String()
}
