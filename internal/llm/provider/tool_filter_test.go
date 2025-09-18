package provider

import (
	"context"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/tools"
)

// mockTool is a test implementation of tools.BaseTool
type mockTool struct {
	name      string
	providers []string
}

func (m *mockTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        m.name,
		Description: "Mock tool for testing",
		Parameters:  map[string]any{"test": "param"},
		Required:    []string{"test"},
		Providers:   m.providers,
	}
}

func (m *mockTool) Run(ctx context.Context, params tools.ToolCall) (tools.ToolResponse, error) {
	return tools.ToolResponse{
		Type:    tools.ToolResponseTypeText,
		Content: "mock response",
	}, nil
}

func TestFilterToolsByProvider(t *testing.T) {
	tests := []struct {
		name         string
		tools        []tools.BaseTool
		providerName string
		wantCount    int
		wantTools    []string
	}{
		{
			name: "no provider restrictions - all tools available",
			tools: []tools.BaseTool{
				&mockTool{name: "tool1", providers: []string{}},
				&mockTool{name: "tool2", providers: nil},
				&mockTool{name: "tool3", providers: []string{}},
			},
			providerName: "openai",
			wantCount:    3,
			wantTools:    []string{"tool1", "tool2", "tool3"},
		},
		{
			name: "provider specific tools - only matching tools returned",
			tools: []tools.BaseTool{
				&mockTool{name: "universal", providers: []string{}},
				&mockTool{name: "xai_only", providers: []string{"xai"}},
				&mockTool{name: "openai_only", providers: []string{"openai"}},
				&mockTool{name: "multi_provider", providers: []string{"openai", "anthropic"}},
			},
			providerName: "openai",
			wantCount:    3,
			wantTools:    []string{"universal", "openai_only", "multi_provider"},
		},
		{
			name: "case insensitive provider matching",
			tools: []tools.BaseTool{
				&mockTool{name: "tool1", providers: []string{"OpenAI"}},
				&mockTool{name: "tool2", providers: []string{"OPENAI"}},
				&mockTool{name: "tool3", providers: []string{"openai"}},
			},
			providerName: "openai",
			wantCount:    3,
			wantTools:    []string{"tool1", "tool2", "tool3"},
		},
		{
			name: "xai provider with web search tool",
			tools: []tools.BaseTool{
				&mockTool{name: "general_tool", providers: []string{}},
				&mockTool{name: "web_search", providers: []string{"xai"}},
				&mockTool{name: "other_xai_tool", providers: []string{"xai"}},
			},
			providerName: "xai",
			wantCount:    3,
			wantTools:    []string{"general_tool", "web_search", "other_xai_tool"},
		},
		{
			name: "no matching tools for provider",
			tools: []tools.BaseTool{
				&mockTool{name: "xai_only", providers: []string{"xai"}},
				&mockTool{name: "openai_only", providers: []string{"openai"}},
			},
			providerName: "anthropic",
			wantCount:    0,
			wantTools:    []string{},
		},
		{
			name:         "empty tool list",
			tools:        []tools.BaseTool{},
			providerName: "openai",
			wantCount:    0,
			wantTools:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterToolsByProvider(tt.tools, tt.providerName)

			if len(filtered) != tt.wantCount {
				t.Errorf("FilterToolsByProvider() returned %d tools, want %d", len(filtered), tt.wantCount)
			}

			// Verify the correct tools were returned
			gotTools := make(map[string]bool)
			for _, tool := range filtered {
				gotTools[tool.Info().Name] = true
			}

			for _, wantTool := range tt.wantTools {
				if !gotTools[wantTool] {
					t.Errorf("Expected tool %s not found in filtered results", wantTool)
				}
			}

			// Ensure no extra tools were included
			for toolName := range gotTools {
				found := false
				for _, wantTool := range tt.wantTools {
					if toolName == wantTool {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected tool %s found in filtered results", toolName)
				}
			}
		})
	}
}

func BenchmarkFilterToolsByProvider(b *testing.B) {
	// Create a mix of tools with different provider restrictions
	tools := []tools.BaseTool{
		&mockTool{name: "universal1", providers: []string{}},
		&mockTool{name: "universal2", providers: nil},
		&mockTool{name: "xai_only", providers: []string{"xai"}},
		&mockTool{name: "openai_only", providers: []string{"openai"}},
		&mockTool{name: "multi_provider1", providers: []string{"openai", "anthropic"}},
		&mockTool{name: "multi_provider2", providers: []string{"xai", "gemini", "openai"}},
		&mockTool{name: "anthropic_only", providers: []string{"anthropic"}},
		&mockTool{name: "gemini_only", providers: []string{"gemini"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterToolsByProvider(tools, "openai")
	}
}

func TestFilterToolsByProvider_EdgeCases(t *testing.T) {
	t.Run("nil tools slice", func(t *testing.T) {
		result := FilterToolsByProvider(nil, "openai")
		if result != nil {
			t.Errorf("Expected nil result for nil input, got %v", result)
		}
	})

	t.Run("empty provider name", func(t *testing.T) {
		tools := []tools.BaseTool{
			&mockTool{name: "tool1", providers: []string{""}},
			&mockTool{name: "tool2", providers: []string{"openai", ""}},
		}
		result := FilterToolsByProvider(tools, "")
		if len(result) != 2 {
			t.Errorf("Expected 2 tools for empty provider match, got %d", len(result))
		}
	})

	t.Run("whitespace in provider names", func(t *testing.T) {
		tools := []tools.BaseTool{
			&mockTool{name: "tool1", providers: []string{" openai "}},
			&mockTool{name: "tool2", providers: []string{"openai"}},
		}
		result := FilterToolsByProvider(tools, "openai")
		// Should only match tool2 since we don't trim whitespace
		if len(result) != 1 || result[0].Info().Name != "tool2" {
			t.Errorf("Expected only tool2 to match, got %d tools", len(result))
		}
	})
}