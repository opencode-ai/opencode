package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_DeferredCompletions(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("basic deferred completion", func(t *testing.T) {
		// Create xAI client with deferred completion enabled
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     100,
			systemMessage: "You are a helpful assistant.",
			xaiOptions: []XAIOption{
				WithDeferredCompletion(),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)
		require.True(t, xaiClient.deferredEnabled)

		// Create a simple message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What is 2+2? Answer in one word."},
				},
			},
		}

		// Send deferred request
		resp, err := xaiClient.send(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify response
		assert.NotEmpty(t, resp.Content)
		assert.NotEmpty(t, resp.SystemFingerprint)
		assert.Equal(t, message.FinishReasonEndTurn, resp.FinishReason)

		t.Logf("Deferred response: %s", resp.Content)
		t.Logf("System fingerprint: %s", resp.SystemFingerprint)
	})

	t.Run("deferred completion with custom options", func(t *testing.T) {
		// Create xAI client with custom deferred options
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     150,
			systemMessage: "You are a helpful assistant.",
			xaiOptions: []XAIOption{
				WithDeferredCompletion(),
				WithDeferredOptions(5*time.Minute, 5*time.Second),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)
		assert.Equal(t, 5*time.Minute, xaiClient.deferredOptions.Timeout)
		assert.Equal(t, 5*time.Second, xaiClient.deferredOptions.Interval)

		// Create message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Explain quantum computing in one sentence."},
				},
			},
		}

		// Send request
		resp, err := xaiClient.send(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.NotEmpty(t, resp.Content)
		assert.NotEmpty(t, resp.SystemFingerprint)
	})

	t.Run("deferred completion with tool use", func(t *testing.T) {
		// Create xAI client with deferred completion
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     200,
			systemMessage: "You are a helpful assistant. Use tools when appropriate.",
			xaiOptions: []XAIOption{
				WithDeferredCompletion(),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)

		// Create a mock tool
		mockTool := &MockTool{
			name:        "calculate",
			description: "Perform calculations",
			parameters: map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "The mathematical expression",
				},
			},
			required: []string{"expression"},
			response: `{"result": 9}`,
		}

		// Create message that should trigger tool use
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What is 3 times 3?"},
				},
			},
		}

		// Send request
		resp, err := xaiClient.send(context.Background(), messages, []tools.BaseTool{mockTool})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify response (may or may not use tool depending on model)
		assert.NotEmpty(t, resp.SystemFingerprint)
		if resp.FinishReason == message.FinishReasonToolUse {
			assert.NotEmpty(t, resp.ToolCalls)
			t.Logf("Tool was called: %+v", resp.ToolCalls)
		} else {
			assert.NotEmpty(t, resp.Content)
			t.Logf("Direct response: %s", resp.Content)
		}
	})
}

func TestXAIProvider_DeferredCompletionsMock(t *testing.T) {
	// Mock server tests for deferred completions
	t.Run("mock deferred completion flow", func(t *testing.T) {
		requestCount := int32(0)
		requestID := "test-request-123"

		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)

			switch r.URL.Path {
			case "/chat/completions":
				// Initial deferred request
				assert.Equal(t, "POST", r.Method)

				// Verify deferred flag
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, true, reqBody["deferred"])

				// Return request ID
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(DeferredCompletionResponse{
					RequestID: requestID,
				})

			case "/chat/deferred-completion/" + requestID:
				// Polling request
				assert.Equal(t, "GET", r.Method)

				if count < 3 {
					// First two polls return 202 (still processing)
					w.WriteHeader(http.StatusAccepted)
				} else {
					// Third poll returns the result
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(DeferredResult{
						ID:      "completion-123",
						Object:  "chat.completion",
						Created: time.Now().Unix(),
						Model:   "grok-3-fast",
						Choices: []DeferredChoice{
							{
								Index: 0,
								Message: DeferredMessage{
									Role:    "assistant",
									Content: "42",
								},
								FinishReason: "stop",
							},
						},
						Usage: DeferredUsage{
							PromptTokens:     10,
							CompletionTokens: 5,
							TotalTokens:      15,
						},
						SystemFingerprint: "fp_test123",
					})
				}

			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Create xAI client pointing to mock server
		opts := providerClientOptions{
			apiKey:        "test-key",
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     100,
			systemMessage: "Test system message",
			openaiOptions: []OpenAIOption{
				WithOpenAIBaseURL(server.URL),
			},
			xaiOptions: []XAIOption{
				WithDeferredCompletion(),
				WithDeferredOptions(30*time.Second, 100*time.Millisecond),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)

		// Override base URL for deferred requests
		xaiClient.openaiClient.options.baseURL = server.URL

		// Create test messages
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What is the answer?"},
				},
			},
		}

		// Send request
		start := time.Now()
		resp, err := xaiClient.SendDeferred(context.Background(), messages, nil, xaiClient.deferredOptions)
		duration := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify response
		assert.Equal(t, "42", resp.Content)
		assert.Equal(t, message.FinishReasonEndTurn, resp.FinishReason)
		assert.Equal(t, int64(10), resp.Usage.InputTokens)
		assert.Equal(t, int64(5), resp.Usage.OutputTokens)
		assert.Equal(t, "fp_test123", resp.SystemFingerprint)

		// Verify polling happened (should have made at least 3 requests)
		assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(3))

		// Verify timing (should have taken at least 200ms due to 2 polls at 100ms interval)
		assert.GreaterOrEqual(t, duration, 200*time.Millisecond)
	})

	t.Run("mock deferred completion timeout", func(t *testing.T) {
		// Create mock server that always returns 202
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/chat/completions":
				// Return request ID
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(DeferredCompletionResponse{
					RequestID: "timeout-test",
				})
			default:
				// Always return 202 (processing)
				w.WriteHeader(http.StatusAccepted)
			}
		}))
		defer server.Close()

		// Create client with short timeout
		opts := providerClientOptions{
			apiKey:        "test-key",
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     100,
			systemMessage: "Test",
			openaiOptions: []OpenAIOption{
				WithOpenAIBaseURL(server.URL),
			},
			xaiOptions: []XAIOption{
				WithDeferredCompletion(),
				WithDeferredOptions(500*time.Millisecond, 100*time.Millisecond),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)
		xaiClient.openaiClient.options.baseURL = server.URL

		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Test"},
				},
			},
		}

		// Should timeout
		_, err := xaiClient.SendDeferred(context.Background(), messages, nil, xaiClient.deferredOptions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}
