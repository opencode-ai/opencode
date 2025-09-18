package provider

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_Streaming(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("basic streaming", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(200),
			WithSystemMessage("You are a helpful assistant. Be concise."),
		)
		require.NoError(t, err)

		// Create a simple message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Count from 1 to 5, one number per line."},
				},
			},
		}

		// Stream response
		ctx := context.Background()
		eventChan := provider.StreamResponse(ctx, messages, nil)

		// Collect events
		var contentChunks []string
		var finalResponse *ProviderResponse
		eventCount := 0
		hasContentDelta := false

		for event := range eventChan {
			eventCount++
			switch event.Type {
			case EventContentDelta:
				hasContentDelta = true
				contentChunks = append(contentChunks, event.Content)
				t.Logf("Content delta: %q", event.Content)

			case EventComplete:
				finalResponse = event.Response
				t.Logf("Stream complete - Total chunks: %d", len(contentChunks))

			case EventError:
				t.Fatalf("Streaming error: %v", event.Error)
			}
		}

		// Verify streaming worked correctly
		require.NotNil(t, finalResponse)
		assert.True(t, hasContentDelta, "Should have received content deltas")
		assert.Greater(t, eventCount, 1, "Should have multiple events")

		// Verify content accumulation
		accumulatedContent := strings.Join(contentChunks, "")
		assert.Equal(t, finalResponse.Content, accumulatedContent, "Accumulated content should match final response")

		// Verify xAI-specific fields
		assert.NotEmpty(t, finalResponse.SystemFingerprint, "Should have system fingerprint")
		assert.Greater(t, finalResponse.Usage.InputTokens, int64(0), "Should have input tokens")
		assert.Greater(t, finalResponse.Usage.OutputTokens, int64(0), "Should have output tokens")

		t.Logf("Final content: %s", finalResponse.Content)
		t.Logf("System fingerprint: %s", finalResponse.SystemFingerprint)
		t.Logf("Usage: %+v", finalResponse.Usage)
	})

	t.Run("streaming with enhanced metrics", func(t *testing.T) {
		// Create xAI client directly to access enhanced features
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     150,
			systemMessage: "You are a helpful assistant.",
		}

		xaiClient := newXAIClient(opts).(*xaiClient)

		// Create message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What is 2+2? Answer in one sentence."},
				},
			},
		}

		// Stream with metrics
		ctx := context.Background()
		startTime := time.Now()
		eventChan := xaiClient.streamWithMetrics(ctx, messages, nil)

		// Track metrics
		var firstTokenTime time.Duration
		chunkCount := 0
		firstChunkReceived := false

		for event := range eventChan {
			switch event.Type {
			case EventContentDelta:
				if !firstChunkReceived {
					firstTokenTime = time.Since(startTime)
					firstChunkReceived = true
					t.Logf("Time to first token: %v", firstTokenTime)
				}
				chunkCount++

			case EventComplete:
				totalTime := time.Since(startTime)
				t.Logf("Total streaming time: %v", totalTime)
				t.Logf("Total chunks: %d", chunkCount)

				// Verify reasonable performance
				assert.Less(t, firstTokenTime, 5*time.Second, "First token should arrive quickly")
				assert.Less(t, totalTime, 10*time.Second, "Total streaming should complete reasonably fast")
			}
		}

		assert.True(t, firstChunkReceived, "Should have received at least one chunk")
		assert.Greater(t, chunkCount, 0, "Should have received chunks")
	})

	t.Run("streaming with reasoning model", func(t *testing.T) {
		// Create provider with grok-4
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant."),
		)
		require.NoError(t, err)

		// Create a reasoning task
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What is 15% of 80? Show your calculation."},
				},
			},
		}

		// Stream with longer timeout for reasoning
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		eventChan := provider.StreamResponse(ctx, messages, nil)

		// Collect response
		var finalResponse *ProviderResponse
		hasContent := false

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

		// Verify response
		require.NotNil(t, finalResponse)
		assert.True(t, hasContent, "Should have received content")
		assert.NotEmpty(t, finalResponse.Content)
		assert.NotEmpty(t, finalResponse.SystemFingerprint)

		t.Logf("Reasoning response: %s", finalResponse.Content)
	})

	t.Run("streaming interruption handling", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant."),
		)
		require.NoError(t, err)

		// Create a longer task
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Count from 1 to 20, one number per line."},
				},
			},
		}

		// Create cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure cancel is always called

		eventChan := provider.StreamResponse(ctx, messages, nil)

		// Cancel after receiving some chunks
		chunkCount := 0
		for event := range eventChan {
			if event.Type == EventContentDelta {
				chunkCount++
				if chunkCount >= 3 {
					cancel() // Interrupt streaming
					break
				}
			}
		}

		// Verify we received some chunks before cancellation
		assert.GreaterOrEqual(t, chunkCount, 3, "Should have received chunks before cancellation")

		// Drain remaining events (should close soon after cancellation)
		timeout := time.After(5 * time.Second)
		for {
			select {
			case _, ok := <-eventChan:
				if !ok {
					return // Channel closed as expected
				}
			case <-timeout:
				t.Fatal("Channel did not close after cancellation")
			}
		}
	})
}
