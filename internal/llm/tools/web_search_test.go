package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWebSearchTool(t *testing.T) {
	tool := &WebSearchTool{}
	ctx := context.Background()

	t.Run("Info returns correct metadata", func(t *testing.T) {
		info := tool.Info()

		if info.Name != "web_search" {
			t.Errorf("Expected tool name 'web_search', got '%s'", info.Name)
		}

		if info.Description == "" {
			t.Error("Tool description should not be empty")
		}

		if !strings.Contains(info.Description, "Live Search") {
			t.Error("Description should mention Live Search functionality")
		}

		// Check parameters structure
		if paramType, ok := info.Parameters["type"]; !ok || paramType != "object" {
			t.Error("Parameters should have type 'object'")
		}

		properties, ok := info.Parameters["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("Parameters should have 'properties' field")
		}

		// Check all Live Search parameters are present
		expectedParams := []string{
			"query", "mode", "max_search_results", "from_date", "to_date",
			"return_citations", "sources",
		}

		for _, param := range expectedParams {
			if _, hasParam := properties[param]; !hasParam {
				t.Errorf("Parameters should have '%s' property", param)
			}
		}

		// Check mode enum values
		if modeParam, ok := properties["mode"].(map[string]interface{}); ok {
			if enumValues, ok := modeParam["enum"].([]string); ok {
				expectedModes := []string{"auto", "on", "off"}
				for _, mode := range expectedModes {
					found := false
					for _, enumVal := range enumValues {
						if enumVal == mode {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Mode enum should include '%s'", mode)
					}
				}
			}
		}

		if len(info.Required) != 1 || info.Required[0] != "query" {
			t.Errorf("Expected required fields to be ['query'], got %v", info.Required)
		}

		// Check provider restriction
		if len(info.Providers) != 1 || info.Providers[0] != "xai" {
			t.Errorf("Expected providers to be ['xai'], got %v", info.Providers)
		}
	})

	t.Run("Run with valid query", func(t *testing.T) {
		params := WebSearchParameters{
			Query: "test search query",
		}

		inputJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal parameters: %v", err)
		}

		toolCall := ToolCall{
			ID:    "test-id",
			Name:  "web_search",
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, toolCall)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if response.IsError {
			t.Errorf("Expected success response, got error: %s", response.Content)
		}

		if response.Content == "" {
			t.Error("Response content should not be empty")
		}

		if !strings.Contains(response.Content, "test search query") {
			t.Error("Response should mention the search query")
		}

		if response.Type != ToolResponseTypeText {
			t.Errorf("Expected response type %s, got %s", ToolResponseTypeText, response.Type)
		}
	})

	t.Run("Run with invalid JSON", func(t *testing.T) {
		toolCall := ToolCall{
			ID:    "test-id",
			Name:  "web_search",
			Input: "invalid json{",
		}

		response, err := tool.Run(ctx, toolCall)
		if err != nil {
			t.Errorf("Expected no error from Run method, got: %v", err)
		}

		if !response.IsError {
			t.Error("Expected error response for invalid JSON")
		}

		if !strings.Contains(response.Content, "parse") {
			t.Error("Error message should mention parsing failure")
		}
	})

	t.Run("Run with empty query", func(t *testing.T) {
		params := WebSearchParameters{
			Query: "",
		}

		inputJSON, _ := json.Marshal(params)
		toolCall := ToolCall{
			ID:    "test-id",
			Name:  "web_search",
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, toolCall)
		if err != nil {
			t.Errorf("Expected no error from Run method, got: %v", err)
		}

		if !response.IsError {
			t.Error("Expected error response for empty query")
		}

		if !strings.Contains(response.Content, "empty") {
			t.Error("Error message should mention empty query")
		}
	})

	// Live Search parameter tests
	t.Run("Run with Live Search parameters", func(t *testing.T) {
		mode := "auto"
		maxResults := 10
		fromDate := "2025-01-01"
		toDate := "2025-12-31"
		returnCitations := true

		params := WebSearchParameters{
			Query:            "AI developments 2025",
			Mode:             &mode,
			MaxSearchResults: &maxResults,
			FromDate:         &fromDate,
			ToDate:           &toDate,
			ReturnCitations:  &returnCitations,
			Sources: []WebSearchSource{
				{
					Type:    "web",
					Country: stringPtr("US"),
				},
				{
					Type:             "news",
					ExcludedWebsites: []string{"example.com"},
				},
			},
		}

		inputJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal parameters: %v", err)
		}

		toolCall := ToolCall{
			ID:    "test-id",
			Name:  "web_search",
			Input: string(inputJSON),
		}

		response, err := tool.Run(ctx, toolCall)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if response.IsError {
			t.Errorf("Expected success response, got error: %s", response.Content)
		}

		content := response.Content
		if !strings.Contains(content, "AI developments 2025") {
			t.Error("Response should mention the search query")
		}

		// Should include parameter details in response (mode auto is default, not shown)
		if !strings.Contains(content, "max results: 10") {
			t.Error("Response should mention max results")
		}

		if !strings.Contains(content, "2025-01-01 to 2025-12-31") {
			t.Error("Response should mention date range")
		}

		if !strings.Contains(content, "[web news]") {
			t.Error("Response should mention source types")
		}
	})

	t.Run("Parameter validation tests", func(t *testing.T) {
		testCases := []struct {
			name        string
			params      WebSearchParameters
			expectError bool
			errorMsg    string
		}{
			{
				name: "invalid mode",
				params: WebSearchParameters{
					Query: "test",
					Mode:  stringPtr("invalid"),
				},
				expectError: true,
				errorMsg:    "mode must be",
			},
			{
				name: "max results too high",
				params: WebSearchParameters{
					Query:            "test",
					MaxSearchResults: intPtr(25),
				},
				expectError: true,
				errorMsg:    "between 1 and 20",
			},
			{
				name: "max results too low",
				params: WebSearchParameters{
					Query:            "test",
					MaxSearchResults: intPtr(0),
				},
				expectError: true,
				errorMsg:    "between 1 and 20",
			},
			{
				name: "invalid date format",
				params: WebSearchParameters{
					Query:    "test",
					FromDate: stringPtr("2025/01/01"),
				},
				expectError: true,
				errorMsg:    "YYYY-MM-DD format",
			},
			{
				name: "invalid source type",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{Type: "invalid"},
					},
				},
				expectError: true,
				errorMsg:    "invalid source type",
			},
			{
				name: "too many excluded websites",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:             "web",
							ExcludedWebsites: []string{"1.com", "2.com", "3.com", "4.com", "5.com", "6.com"},
						},
					},
				},
				expectError: true,
				errorMsg:    "cannot exceed 5",
			},
			{
				name: "conflicting website filters",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:             "web",
							ExcludedWebsites: []string{"example.com"},
							AllowedWebsites:  []string{"test.com"},
						},
					},
				},
				expectError: true,
				errorMsg:    "cannot use both",
			},
			{
				name: "too many X handles",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:             "x",
							IncludedXHandles: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"},
						},
					},
				},
				expectError: true,
				errorMsg:    "cannot exceed 10",
			},
			{
				name: "conflicting X handle filters",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:             "x",
							IncludedXHandles: []string{"xai"},
							ExcludedXHandles: []string{"openai"},
						},
					},
				},
				expectError: true,
				errorMsg:    "cannot use both",
			},
			{
				name: "RSS without links",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{Type: "rss"},
					},
				},
				expectError: true,
				errorMsg:    "requires at least one link",
			},
			{
				name: "RSS with too many links",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:  "rss",
							Links: []string{"feed1.xml", "feed2.xml"},
						},
					},
				},
				expectError: true,
				errorMsg:    "can only have 1 link",
			},
			{
				name: "X parameters on web source",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:             "web",
							IncludedXHandles: []string{"xai"},
						},
					},
				},
				expectError: true,
				errorMsg:    "X-specific parameters not allowed",
			},
			{
				name: "web parameters on X source",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:    "x",
							Country: stringPtr("US"),
						},
					},
				},
				expectError: true,
				errorMsg:    "web/news-specific parameters not allowed",
			},
			{
				name: "allowed websites on news source",
				params: WebSearchParameters{
					Query: "test",
					Sources: []WebSearchSource{
						{
							Type:            "news",
							AllowedWebsites: []string{"news.com"},
						},
					},
				},
				expectError: true,
				errorMsg:    "allowed_websites not supported for news",
			},
			{
				name: "valid parameters",
				params: WebSearchParameters{
					Query:            "test query",
					Mode:             stringPtr("auto"),
					MaxSearchResults: intPtr(10),
					FromDate:         stringPtr("2025-01-01"),
					ToDate:           stringPtr("2025-12-31"),
					ReturnCitations:  boolPtr(true),
					Sources: []WebSearchSource{
						{
							Type:             "web",
							Country:          stringPtr("US"),
							ExcludedWebsites: []string{"spam.com"},
						},
						{
							Type:              "x",
							IncludedXHandles:  []string{"xai"},
							PostFavoriteCount: intPtr(100),
						},
						{
							Type:    "news",
							Country: stringPtr("UK"),
						},
						{
							Type:  "rss",
							Links: []string{"https://feeds.example.com/rss.xml"},
						},
					},
				},
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				inputJSON, err := json.Marshal(tc.params)
				if err != nil {
					t.Fatalf("Failed to marshal parameters: %v", err)
				}

				toolCall := ToolCall{
					ID:    "test-id",
					Name:  "web_search",
					Input: string(inputJSON),
				}

				response, err := tool.Run(ctx, toolCall)
				if err != nil {
					t.Errorf("Expected no error from Run method, got: %v", err)
				}

				if tc.expectError {
					if !response.IsError {
						t.Errorf("Expected error response for %s", tc.name)
					}
					if !strings.Contains(response.Content, tc.errorMsg) {
						t.Errorf("Expected error message to contain '%s', got: %s", tc.errorMsg, response.Content)
					}
				} else {
					if response.IsError {
						t.Errorf("Expected success response for %s, got error: %s", tc.name, response.Content)
					}
				}
			})
		}
	})
}

// Helper functions for pointer creation
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
