package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// WebSearchTool provides web search functionality for AI models that support it.
// This tool is primarily used by providers like xAI Grok models which have built-in
// web search capabilities. The actual search execution is handled by the model provider,
// not by this tool implementation. It supports advanced Live Search parameters for
// enhanced search control and filtering.
type WebSearchTool struct{}

// WebSearchParameters defines the input parameters for web search requests.
// Supports xAI Live Search parameters for enhanced search capabilities.
type WebSearchParameters struct {
	Query            string            `json:"query" jsonschema:"description=The search query to execute,required"`
	Mode             *string           `json:"mode,omitempty" jsonschema:"description=Search mode: auto|on|off (default: auto)"`
	MaxSearchResults *int              `json:"max_search_results,omitempty" jsonschema:"description=Maximum number of search results (1-20, default: 20)"`
	FromDate         *string           `json:"from_date,omitempty" jsonschema:"description=Start date for search results in YYYY-MM-DD format"`
	ToDate           *string           `json:"to_date,omitempty" jsonschema:"description=End date for search results in YYYY-MM-DD format"`
	ReturnCitations  *bool             `json:"return_citations,omitempty" jsonschema:"description=Whether to return citations (default: true)"`
	Sources          []WebSearchSource `json:"sources,omitempty" jsonschema:"description=List of data sources to search"`
}

// WebSearchSource represents a data source for Live Search
type WebSearchSource struct {
	Type              string   `json:"type" jsonschema:"description=Source type: web|x|news|rss,required"`
	Country           *string  `json:"country,omitempty" jsonschema:"description=ISO alpha-2 country code (web, news)"`
	ExcludedWebsites  []string `json:"excluded_websites,omitempty" jsonschema:"description=Websites to exclude (max 5, web/news)"`
	AllowedWebsites   []string `json:"allowed_websites,omitempty" jsonschema:"description=Allowed websites only (max 5, web only)"`
	SafeSearch        *bool    `json:"safe_search,omitempty" jsonschema:"description=Enable safe search (default: true, web/news)"`
	IncludedXHandles  []string `json:"included_x_handles,omitempty" jsonschema:"description=X handles to include (max 10, x only)"`
	ExcludedXHandles  []string `json:"excluded_x_handles,omitempty" jsonschema:"description=X handles to exclude (max 10, x only)"`
	PostFavoriteCount *int     `json:"post_favorite_count,omitempty" jsonschema:"description=Minimum favorite count for X posts"`
	PostViewCount     *int     `json:"post_view_count,omitempty" jsonschema:"description=Minimum view count for X posts"`
	Links             []string `json:"links,omitempty" jsonschema:"description=RSS feed URLs (1 link max, rss only)"`
}

// Info returns metadata about the web search tool including its parameters and description.
func (t *WebSearchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "web_search",
		Description: "Search the web for current information with advanced Live Search capabilities. Supports multiple data sources (web, X, news, RSS), date filtering, and citation tracking.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query to execute",
				},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Search mode: 'auto' (default), 'on', or 'off'",
				"enum":        []string{"auto", "on", "off"},
			},
			"max_search_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of search results (1-20, default: 20)",
				"minimum":     1,
				"maximum":     20,
			},
			"from_date": map[string]interface{}{
				"type":        "string",
				"description": "Start date for search results in YYYY-MM-DD format",
				"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
			},
			"to_date": map[string]interface{}{
				"type":        "string",
				"description": "End date for search results in YYYY-MM-DD format",
				"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
			},
			"return_citations": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to return citations (default: true)",
			},
			"sources": map[string]interface{}{
				"type":        "array",
				"description": "List of data sources to search",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Source type",
							"enum":        []string{"web", "x", "news", "rss"},
						},
						"country": map[string]interface{}{
							"type":        "string",
							"description": "ISO alpha-2 country code (web, news)",
							"pattern":     "^[A-Z]{2}$",
						},
						"excluded_websites": map[string]interface{}{
							"type":        "array",
							"description": "Websites to exclude (max 5, web/news)",
							"items":       map[string]interface{}{"type": "string"},
							"maxItems":    5,
						},
						"allowed_websites": map[string]interface{}{
							"type":        "array",
							"description": "Allowed websites only (max 5, web only)",
							"items":       map[string]interface{}{"type": "string"},
							"maxItems":    5,
						},
						"safe_search": map[string]interface{}{
							"type":        "boolean",
							"description": "Enable safe search (default: true, web/news)",
						},
						"included_x_handles": map[string]interface{}{
							"type":        "array",
							"description": "X handles to include (max 10, x only)",
							"items":       map[string]interface{}{"type": "string"},
							"maxItems":    10,
						},
						"excluded_x_handles": map[string]interface{}{
							"type":        "array",
							"description": "X handles to exclude (max 10, x only)",
							"items":       map[string]interface{}{"type": "string"},
							"maxItems":    10,
						},
						"post_favorite_count": map[string]interface{}{
							"type":        "integer",
							"description": "Minimum favorite count for X posts",
							"minimum":     0,
						},
						"post_view_count": map[string]interface{}{
							"type":        "integer",
							"description": "Minimum view count for X posts",
							"minimum":     0,
						},
						"links": map[string]interface{}{
							"type":        "array",
							"description": "RSS feed URLs (1 link max, rss only)",
							"items":       map[string]interface{}{"type": "string", "format": "uri"},
							"maxItems":    1,
						},
					},
					"required": []string{"type"},
				},
			},
		},
		"required": []string{"query"},
	},
		Required:  []string{"query"},
		Providers: []string{"xai"}, // Web search is currently only supported by xAI models
	}
}

// Run processes the web search request. Since the actual web search is performed
// by the AI model provider (e.g., xAI), this method validates the input parameters
// and returns a response indicating that the search will be handled by the provider.
func (t *WebSearchTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var searchParams WebSearchParameters

	if err := json.Unmarshal([]byte(params.Input), &searchParams); err != nil {
		return ToolResponse{
			Type:    ToolResponseTypeText,
			Content: fmt.Sprintf("Failed to parse web search parameters: %v", err),
			IsError: true,
		}, nil
	}

	// Validate query is not empty
	if searchParams.Query == "" {
		return ToolResponse{
			Type:    ToolResponseTypeText,
			Content: "Search query cannot be empty",
			IsError: true,
		}, nil
	}

	// Validate Live Search parameters
	if err := t.validateLiveSearchParams(&searchParams); err != nil {
		return ToolResponse{
			Type:    ToolResponseTypeText,
			Content: fmt.Sprintf("Invalid Live Search parameters: %v", err),
			IsError: true,
		}, nil
	}

	// Build description of search configuration
	description := fmt.Sprintf("Searching the web for: %s", searchParams.Query)

	if searchParams.Mode != nil && *searchParams.Mode != "auto" {
		description += fmt.Sprintf(" (mode: %s)", *searchParams.Mode)
	}

	if searchParams.MaxSearchResults != nil {
		description += fmt.Sprintf(" (max results: %d)", *searchParams.MaxSearchResults)
	}

	if searchParams.FromDate != nil || searchParams.ToDate != nil {
		if searchParams.FromDate != nil && searchParams.ToDate != nil {
			description += fmt.Sprintf(" (date range: %s to %s)", *searchParams.FromDate, *searchParams.ToDate)
		} else if searchParams.FromDate != nil {
			description += fmt.Sprintf(" (from: %s)", *searchParams.FromDate)
		} else {
			description += fmt.Sprintf(" (until: %s)", *searchParams.ToDate)
		}
	}

	if len(searchParams.Sources) > 0 {
		sourceTypes := make([]string, len(searchParams.Sources))
		for i, source := range searchParams.Sources {
			sourceTypes[i] = source.Type
		}
		description += fmt.Sprintf(" (sources: %v)", sourceTypes)
	}

	// Return success response indicating the provider will handle the search
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: description,
		IsError: false,
	}, nil
}

// validateLiveSearchParams validates Live Search parameters according to xAI specifications
func (t *WebSearchTool) validateLiveSearchParams(params *WebSearchParameters) error {
	// Validate mode
	if params.Mode != nil {
		mode := *params.Mode
		if mode != "auto" && mode != "on" && mode != "off" {
			return fmt.Errorf("mode must be 'auto', 'on', or 'off', got: %s", mode)
		}
	}

	// Validate max_search_results range
	if params.MaxSearchResults != nil {
		if *params.MaxSearchResults < 1 || *params.MaxSearchResults > 20 {
			return fmt.Errorf("max_search_results must be between 1 and 20, got: %d", *params.MaxSearchResults)
		}
	}

	// Validate date formats (basic YYYY-MM-DD validation)
	if params.FromDate != nil {
		date := *params.FromDate
		if len(date) != 10 || date[4] != '-' || date[7] != '-' {
			return fmt.Errorf("from_date must be in YYYY-MM-DD format, got: %s", date)
		}
	}
	if params.ToDate != nil {
		date := *params.ToDate
		if len(date) != 10 || date[4] != '-' || date[7] != '-' {
			return fmt.Errorf("to_date must be in YYYY-MM-DD format, got: %s", date)
		}
	}

	// Validate sources
	for i, source := range params.Sources {
		if err := t.validateSource(&source, i); err != nil {
			return fmt.Errorf("source %d: %w", i, err)
		}
	}

	return nil
}

// validateSource validates individual source parameters
func (t *WebSearchTool) validateSource(source *WebSearchSource, index int) error {
	// Validate source type
	validTypes := map[string]bool{"web": true, "x": true, "news": true, "rss": true}
	if !validTypes[source.Type] {
		return fmt.Errorf("invalid source type: %s (must be web, x, news, or rss)", source.Type)
	}

	// Validate website exclusions/inclusions
	if len(source.ExcludedWebsites) > 5 {
		return fmt.Errorf("excluded_websites cannot exceed 5 entries, got: %d", len(source.ExcludedWebsites))
	}
	if len(source.AllowedWebsites) > 5 {
		return fmt.Errorf("allowed_websites cannot exceed 5 entries, got: %d", len(source.AllowedWebsites))
	}
	if len(source.ExcludedWebsites) > 0 && len(source.AllowedWebsites) > 0 {
		return fmt.Errorf("cannot use both excluded_websites and allowed_websites in the same source")
	}

	// Validate X handles
	if len(source.IncludedXHandles) > 10 {
		return fmt.Errorf("included_x_handles cannot exceed 10 entries, got: %d", len(source.IncludedXHandles))
	}
	if len(source.ExcludedXHandles) > 10 {
		return fmt.Errorf("excluded_x_handles cannot exceed 10 entries, got: %d", len(source.ExcludedXHandles))
	}
	if len(source.IncludedXHandles) > 0 && len(source.ExcludedXHandles) > 0 {
		return fmt.Errorf("cannot use both included_x_handles and excluded_x_handles in the same source")
	}

	// Validate RSS links
	if len(source.Links) > 1 {
		return fmt.Errorf("RSS source can only have 1 link, got: %d", len(source.Links))
	}

	// Validate source-specific parameters
	switch source.Type {
	case "web":
		if len(source.IncludedXHandles) > 0 || len(source.ExcludedXHandles) > 0 ||
			source.PostFavoriteCount != nil || source.PostViewCount != nil {
			return fmt.Errorf("X-specific parameters not allowed for web source")
		}
		if len(source.Links) > 0 {
			return fmt.Errorf("RSS links not allowed for web source")
		}
	case "x":
		if source.Country != nil || len(source.ExcludedWebsites) > 0 ||
			len(source.AllowedWebsites) > 0 || source.SafeSearch != nil {
			return fmt.Errorf("web/news-specific parameters not allowed for X source")
		}
		if len(source.Links) > 0 {
			return fmt.Errorf("RSS links not allowed for X source")
		}
	case "news":
		if len(source.IncludedXHandles) > 0 || len(source.ExcludedXHandles) > 0 ||
			source.PostFavoriteCount != nil || source.PostViewCount != nil {
			return fmt.Errorf("X-specific parameters not allowed for news source")
		}
		if len(source.AllowedWebsites) > 0 {
			return fmt.Errorf("allowed_websites not supported for news source")
		}
		if len(source.Links) > 0 {
			return fmt.Errorf("RSS links not allowed for news source")
		}
	case "rss":
		if source.Country != nil || len(source.ExcludedWebsites) > 0 ||
			len(source.AllowedWebsites) > 0 || source.SafeSearch != nil ||
			len(source.IncludedXHandles) > 0 || len(source.ExcludedXHandles) > 0 ||
			source.PostFavoriteCount != nil || source.PostViewCount != nil {
			return fmt.Errorf("only links parameter allowed for RSS source")
		}
		if len(source.Links) == 0 {
			return fmt.Errorf("RSS source requires at least one link")
		}
	}

	return nil
}
