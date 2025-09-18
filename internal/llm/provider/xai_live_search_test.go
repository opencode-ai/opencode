package provider

import (
	"context"
	"os"
	"testing"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_LiveSearchOptions(t *testing.T) {
	t.Run("default live search options", func(t *testing.T) {
		opts := DefaultLiveSearchOptions()

		assert.Equal(t, "auto", opts.Mode)
		assert.NotNil(t, opts.ReturnCitations)
		assert.True(t, *opts.ReturnCitations)
		assert.Len(t, opts.Sources, 2)
		assert.Equal(t, "web", opts.Sources[0].Type)
		assert.Equal(t, "x", opts.Sources[1].Type)
	})

	t.Run("custom live search options", func(t *testing.T) {
		maxResults := 10
		fromDate := "2025-01-01"
		toDate := "2025-12-31"
		returnCitations := false

		opts := LiveSearchOptions{
			Mode:             "on",
			MaxSearchResults: &maxResults,
			FromDate:         &fromDate,
			ToDate:           &toDate,
			ReturnCitations:  &returnCitations,
			Sources: []LiveSearchSource{
				{
					Type:             "web",
					Country:          stringPtr("US"),
					ExcludedWebsites: []string{"example.com"},
				},
				{
					Type:    "news",
					Country: stringPtr("UK"),
				},
				{
					Type:              "x",
					IncludedXHandles:  []string{"xai", "openai"},
					PostFavoriteCount: intPtr(100),
				},
				{
					Type:  "rss",
					Links: []string{"https://example.com/feed.xml"},
				},
			},
		}

		assert.Equal(t, "on", opts.Mode)
		assert.Equal(t, 10, *opts.MaxSearchResults)
		assert.Equal(t, "2025-01-01", *opts.FromDate)
		assert.Equal(t, "2025-12-31", *opts.ToDate)
		assert.False(t, *opts.ReturnCitations)
		assert.Len(t, opts.Sources, 4)

		// Check web source
		webSource := opts.Sources[0]
		assert.Equal(t, "web", webSource.Type)
		assert.Equal(t, "US", *webSource.Country)
		assert.Equal(t, []string{"example.com"}, webSource.ExcludedWebsites)

		// Check X source
		xSource := opts.Sources[2]
		assert.Equal(t, "x", xSource.Type)
		assert.Equal(t, []string{"xai", "openai"}, xSource.IncludedXHandles)
		assert.Equal(t, 100, *xSource.PostFavoriteCount)

		// Check RSS source
		rssSource := opts.Sources[3]
		assert.Equal(t, "rss", rssSource.Type)
		assert.Equal(t, []string{"https://example.com/feed.xml"}, rssSource.Links)
	})

	t.Run("xai client with live search options", func(t *testing.T) {
		opts := providerClientOptions{
			apiKey:        "test-key",
			model:         models.SupportedModels[models.XAIGrok4],
			maxTokens:     1000,
			systemMessage: "Test system message",
			xaiOptions: []XAIOption{
				WithLiveSearch(),
			},
		}

		client := newXAIClient(opts).(*xaiClient)

		assert.True(t, client.liveSearchEnabled)
		assert.Equal(t, "auto", client.liveSearchOptions.Mode)
		assert.NotNil(t, client.liveSearchOptions.ReturnCitations)
		assert.True(t, *client.liveSearchOptions.ReturnCitations)
	})

	t.Run("xai client with custom live search options", func(t *testing.T) {
		customOpts := LiveSearchOptions{
			Mode:             "on",
			MaxSearchResults: intPtr(5),
			Sources: []LiveSearchSource{
				{Type: "web"},
			},
		}

		opts := providerClientOptions{
			apiKey:        "test-key",
			model:         models.SupportedModels[models.XAIGrok4],
			maxTokens:     1000,
			systemMessage: "Test system message",
			xaiOptions: []XAIOption{
				WithLiveSearchOptions(customOpts),
			},
		}

		client := newXAIClient(opts).(*xaiClient)

		assert.True(t, client.liveSearchEnabled)
		assert.Equal(t, "on", client.liveSearchOptions.Mode)
		assert.Equal(t, 5, *client.liveSearchOptions.MaxSearchResults)
		assert.Len(t, client.liveSearchOptions.Sources, 1)
		assert.Equal(t, "web", client.liveSearchOptions.Sources[0].Type)
	})

	t.Run("combined xai options", func(t *testing.T) {
		opts := providerClientOptions{
			apiKey:        "test-key",
			model:         models.SupportedModels[models.XAIGrok4],
			maxTokens:     1000,
			systemMessage: "Test system message",
			xaiOptions: []XAIOption{
				WithMaxConcurrentRequests(3),
				WithDeferredCompletion(),
				WithLiveSearch(),
			},
		}

		client := newXAIClient(opts).(*xaiClient)

		// Verify all options are applied
		assert.NotNil(t, client.concurrent)
		assert.True(t, client.deferredEnabled)
		assert.True(t, client.liveSearchEnabled)

		assert.Equal(t, int64(3), client.concurrent.GetMaxConcurrent())
		assert.Equal(t, "auto", client.liveSearchOptions.Mode)
	})
}

func TestXAIProvider_LiveSearchIntegration(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("basic live search request", func(t *testing.T) {
		// Create provider with Live Search enabled
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant that can search the web for current information."),
			WithXAIOptions(
				WithLiveSearch(),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		// Create a message that should trigger live search
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What are the latest news about artificial intelligence in 2025?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{webSearchTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Live Search responses should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Response should have content (either from tools or direct search)
		assert.True(t, response.Content != "" || len(response.ToolCalls) > 0)

		// Log response details for manual verification
		t.Logf("Response content length: %d", len(response.Content))
		t.Logf("Number of tool calls: %d", len(response.ToolCalls))
		t.Logf("Citations: %v", response.Citations)
		t.Logf("System fingerprint: %s", response.SystemFingerprint)

		// If citations are present, they should be valid URLs
		for i, citation := range response.Citations {
			assert.NotEmpty(t, citation, "Citation %d should not be empty", i)
			t.Logf("Citation %d: %s", i+1, citation)
		}
	})

	t.Run("live search with date filtering", func(t *testing.T) {
		fromDate := "2025-01-01"
		toDate := "2025-01-31"

		customOpts := LiveSearchOptions{
			Mode:             "on",
			MaxSearchResults: intPtr(5),
			FromDate:         &fromDate,
			ToDate:           &toDate,
			ReturnCitations:  boolPtr(true),
			Sources: []LiveSearchSource{
				{Type: "web"},
				{Type: "news"},
			},
		}

		// Create provider with custom Live Search options
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearchOptions(customOpts),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		// Create a message that should use the date-filtered search
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What major technology announcements happened in January 2025?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{webSearchTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Log response for verification
		t.Logf("Date-filtered search response: %s", response.Content)
		t.Logf("Citations: %v", response.Citations)
	})

	t.Run("live search with x source filtering", func(t *testing.T) {
		customOpts := LiveSearchOptions{
			Mode:             "on",
			MaxSearchResults: intPtr(3),
			ReturnCitations:  boolPtr(true),
			Sources: []LiveSearchSource{
				{
					Type:              "x",
					IncludedXHandles:  []string{"xai", "elonmusk"},
					PostFavoriteCount: intPtr(10),
				},
			},
		}

		// Create provider with X-specific search
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearchOptions(customOpts),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		// Create a message about xAI updates
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What are the latest updates from xAI?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{webSearchTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Log response for verification
		t.Logf("X-filtered search response: %s", response.Content)
		t.Logf("Citations: %v", response.Citations)
	})

	t.Run("live search combined with deferred completion", func(t *testing.T) {
		// Create provider with both Live Search and deferred completion
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearch(),
				WithDeferredCompletion(),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		// Create a complex message that might require deferred processing
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Please provide a comprehensive analysis of recent AI developments, including the latest research papers, company announcements, and market trends."},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{webSearchTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Should have substantial content for comprehensive analysis
		assert.NotEmpty(t, response.Content)

		// Log response for verification
		t.Logf("Deferred + Live Search response length: %d characters", len(response.Content))
		t.Logf("Citations: %d", len(response.Citations))
		t.Logf("System fingerprint: %s", response.SystemFingerprint)
	})

	t.Run("streaming with live search", func(t *testing.T) {
		// Create provider with Live Search
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(300),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearch(),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		// Create a message for streaming search
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What's happening in the tech world today?"},
				},
			},
		}

		// Stream response
		eventChan := provider.StreamResponse(context.Background(), messages, []tools.BaseTool{webSearchTool})

		var finalResponse *ProviderResponse
		var hasContent bool

		for event := range eventChan {
			switch event.Type {
			case EventContentDelta:
				hasContent = true
			case EventComplete:
				finalResponse = event.Response
			case EventError:
				t.Fatalf("Streaming error: %v", event.Error)
			}
		}

		require.NotNil(t, finalResponse)

		// Should have content or tool calls
		assert.True(t, hasContent || finalResponse.Content != "" || len(finalResponse.ToolCalls) > 0)

		// Should have system fingerprint
		assert.NotEmpty(t, finalResponse.SystemFingerprint)

		// Log final response
		t.Logf("Streaming final response: %s", finalResponse.Content)
		t.Logf("Streaming citations: %v", finalResponse.Citations)
	})
}

// Helper functions are defined in xai_test.go to avoid duplication
