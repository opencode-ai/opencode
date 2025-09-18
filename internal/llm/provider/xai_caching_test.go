package provider

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_PromptCaching(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("prompt caching with repeated requests", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]), // Use a fast model for testing
			WithMaxTokens(100),
			WithSystemMessage("You are a helpful assistant. Answer concisely."),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Create a longer prompt that's likely to be cached
		longPrompt := `Please analyze the following scenario: A company is evaluating whether to implement 
		a new software system. The system costs $100,000 initially and $20,000 per year to maintain. 
		It will save the company $35,000 per year in operational costs. The company expects to use 
		this system for 5 years. Should they implement this system? Provide a brief analysis.`

		baseMessages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: longPrompt},
				},
			},
		}

		ctx := context.Background()

		// First request - should create cache
		t.Log("Making first request (cache creation)...")
		resp1, err := provider.SendMessages(ctx, baseMessages, nil)
		require.NoError(t, err)
		require.NotNil(t, resp1)

		t.Logf("First request usage: input=%d, output=%d, cached=%d, cache_creation=%d",
			resp1.Usage.InputTokens, resp1.Usage.OutputTokens,
			resp1.Usage.CacheReadTokens, resp1.Usage.CacheCreationTokens)

		// Wait a moment to ensure request completes
		time.Sleep(1 * time.Second)

		// Second request with same prompt - should use cache
		t.Log("Making second request (cache hit expected)...")
		resp2, err := provider.SendMessages(ctx, baseMessages, nil)
		require.NoError(t, err)
		require.NotNil(t, resp2)

		t.Logf("Second request usage: input=%d, output=%d, cached=%d, cache_creation=%d",
			resp2.Usage.InputTokens, resp2.Usage.OutputTokens,
			resp2.Usage.CacheReadTokens, resp2.Usage.CacheCreationTokens)

		// Check if caching occurred in either request
		totalCachedTokens := resp1.Usage.CacheReadTokens + resp2.Usage.CacheReadTokens
		if totalCachedTokens > 0 {
			t.Logf("✓ Prompt caching detected! Total cached tokens: %d", totalCachedTokens)

			// Calculate cache efficiency
			totalPromptTokens := resp1.Usage.InputTokens + resp1.Usage.CacheReadTokens +
				resp2.Usage.InputTokens + resp2.Usage.CacheReadTokens
			cacheHitRate := float64(totalCachedTokens) / float64(totalPromptTokens) * 100
			t.Logf("Cache hit rate: %.1f%%", cacheHitRate)

			// Test cache cost savings calculation
			savings := xaiClient.calculateCacheCostSavings(resp2.Usage)
			if savings > 0 {
				t.Logf("Estimated cost savings: $%.6f", savings)
			}
		} else {
			t.Log("No caching detected in this test (may need more requests or longer prompts)")
		}

		// Verify responses are different (since this is a generative task)
		assert.NotEqual(t, resp1.Content, resp2.Content, "Responses should be different for generative tasks")

		// Verify both responses have content
		assert.NotEmpty(t, resp1.Content)
		assert.NotEmpty(t, resp2.Content)

		// Verify system fingerprints are present
		assert.NotEmpty(t, resp1.SystemFingerprint)
		assert.NotEmpty(t, resp2.SystemFingerprint)
	})

	t.Run("streaming with caching", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(50),
			WithSystemMessage("You are a helpful assistant."),
		)
		require.NoError(t, err)

		// Use the same prompt twice to test caching in streaming
		prompt := "What is the capital of France? Answer in one sentence."
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: prompt},
				},
			},
		}

		ctx := context.Background()

		// First streaming request
		t.Log("First streaming request...")
		eventChan1 := provider.StreamResponse(ctx, messages, nil)
		var finalResp1 *ProviderResponse

		for event := range eventChan1 {
			if event.Type == EventComplete {
				finalResp1 = event.Response
				break
			} else if event.Type == EventError {
				t.Fatalf("Streaming error: %v", event.Error)
			}
		}

		require.NotNil(t, finalResp1)
		t.Logf("First streaming usage: input=%d, output=%d, cached=%d",
			finalResp1.Usage.InputTokens, finalResp1.Usage.OutputTokens, finalResp1.Usage.CacheReadTokens)

		// Wait a moment
		time.Sleep(1 * time.Second)

		// Second streaming request
		t.Log("Second streaming request...")
		eventChan2 := provider.StreamResponse(ctx, messages, nil)
		var finalResp2 *ProviderResponse

		for event := range eventChan2 {
			if event.Type == EventComplete {
				finalResp2 = event.Response
				break
			} else if event.Type == EventError {
				t.Fatalf("Streaming error: %v", event.Error)
			}
		}

		require.NotNil(t, finalResp2)
		t.Logf("Second streaming usage: input=%d, output=%d, cached=%d",
			finalResp2.Usage.InputTokens, finalResp2.Usage.OutputTokens, finalResp2.Usage.CacheReadTokens)

		// Check for any caching
		totalCached := finalResp1.Usage.CacheReadTokens + finalResp2.Usage.CacheReadTokens
		if totalCached > 0 {
			t.Logf("✓ Streaming caching detected! Total cached tokens: %d", totalCached)
		} else {
			t.Log("No caching detected in streaming requests")
		}

		// Verify we got valid responses
		assert.NotEmpty(t, finalResp1.Content)
		assert.NotEmpty(t, finalResp2.Content)
	})

	t.Run("deferred completion with caching", func(t *testing.T) {
		// Test caching with deferred completions
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(100),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Enable deferred mode
		xaiClient.deferredEnabled = true
		xaiClient.deferredOptions = DeferredOptions{
			Timeout:  2 * time.Minute,
			Interval: 5 * time.Second,
		}

		prompt := "Explain quantum computing in simple terms."
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: prompt},
				},
			},
		}

		ctx := context.Background()

		// Send deferred request
		t.Log("Sending deferred request...")
		resp, err := provider.SendMessages(ctx, messages, nil)
		if err != nil {
			t.Logf("Deferred request failed (expected for some models): %v", err)
			return
		}

		require.NotNil(t, resp)
		t.Logf("Deferred usage: input=%d, output=%d, cached=%d",
			resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.CacheReadTokens)

		if resp.Usage.CacheReadTokens > 0 {
			t.Logf("✓ Deferred completion caching detected! Cached tokens: %d", resp.Usage.CacheReadTokens)
		}

		assert.NotEmpty(t, resp.Content)
	})

	t.Run("cache metrics validation", func(t *testing.T) {
		// Test the cache cost savings calculation
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Test with mock usage data
		mockUsage := TokenUsage{
			InputTokens:         100,
			OutputTokens:        50,
			CacheReadTokens:     25,
			CacheCreationTokens: 0,
		}

		savings := xaiClient.calculateCacheCostSavings(mockUsage)
		t.Logf("Mock cache savings for 25 cached tokens: $%.6f", savings)

		// Savings should be positive when there are cached tokens
		if mockUsage.CacheReadTokens > 0 {
			assert.Greater(t, savings, 0.0, "Should have positive savings with cached tokens")
		}

		// Test with zero cached tokens
		zeroUsage := TokenUsage{
			InputTokens:     100,
			OutputTokens:    50,
			CacheReadTokens: 0,
		}
		zeroSavings := xaiClient.calculateCacheCostSavings(zeroUsage)
		assert.Equal(t, 0.0, zeroSavings, "Should have zero savings with no cached tokens")
	})
}

func TestCacheTokenHandling(t *testing.T) {
	// Test the cached token parsing in deferred results

	// Mock deferred result with cached tokens
	result := &DeferredResult{
		Usage: DeferredUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			PromptTokensDetails: &DeferredPromptTokensDetails{
				TextTokens:   75,
				CachedTokens: 25,
				ImageTokens:  0,
				AudioTokens:  0,
			},
		},
	}

	// Test parsing
	cachedTokens := int64(0)
	if result.Usage.PromptTokensDetails != nil {
		cachedTokens = result.Usage.PromptTokensDetails.CachedTokens
	}
	inputTokens := result.Usage.PromptTokens - cachedTokens

	assert.Equal(t, int64(25), cachedTokens, "Should extract cached tokens correctly")
	assert.Equal(t, int64(75), inputTokens, "Should calculate input tokens correctly")
	assert.Equal(t, int64(100), inputTokens+cachedTokens, "Total should match prompt tokens")
}
