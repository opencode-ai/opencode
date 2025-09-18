package provider

import (
	"context"
	"encoding/json"
	"os"
	"sync"
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

// MockTool implements a simple tool for testing
type MockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	required    []string
	response    string
	callCount   int
}

func (t *MockTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
		Required:    t.required,
	}
}

func (t *MockTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	t.callCount++
	return tools.NewTextResponse(t.response), nil
}

func TestXAIProvider_FunctionCalling(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("basic function calling", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(1000),
			WithSystemMessage("You are a helpful assistant. Use the provided tools when appropriate."),
		)
		require.NoError(t, err)

		// Create a mock tool
		mockTool := &MockTool{
			name:        "get_weather",
			description: "Get the current weather in a given location",
			parameters: map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The city and state, e.g. San Francisco, CA",
				},
			},
			required: []string{"location"},
			response: `{"temperature": 72, "condition": "sunny"}`,
		}

		// Create a message that should trigger tool use
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What's the weather like in San Francisco?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{mockTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify tool was called
		assert.Equal(t, message.FinishReasonToolUse, response.FinishReason)
		assert.NotEmpty(t, response.ToolCalls)
		assert.NotEmpty(t, response.SystemFingerprint)

		// Verify tool call details
		if len(response.ToolCalls) > 0 {
			assert.Equal(t, "get_weather", response.ToolCalls[0].Name)
			assert.NotEmpty(t, response.ToolCalls[0].ID)
			assert.NotEmpty(t, response.ToolCalls[0].Input)
			assert.True(t, response.ToolCalls[0].Finished)

			// Verify the input contains location
			var input map[string]interface{}
			err := json.Unmarshal([]byte(response.ToolCalls[0].Input), &input)
			assert.NoError(t, err)
			assert.Contains(t, input, "location")
		}
	})

	t.Run("tool choice modes", func(t *testing.T) {
		testCases := []struct {
			name       string
			toolChoice string
			message    string
			expectTool bool
		}{
			{
				name:       "auto mode with tool-triggering prompt",
				toolChoice: "auto",
				message:    "What's the weather in New York?",
				expectTool: true,
			},
			{
				name:       "none mode should not call tools",
				toolChoice: "none",
				message:    "What's the weather in New York?",
				expectTool: false,
			},
			{
				name:       "required mode forces tool call",
				toolChoice: "required",
				message:    "Hello there!",
				expectTool: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create provider with specific tool choice
				provider, err := NewProvider(
					models.ProviderXAI,
					WithAPIKey(apiKey),
					WithModel(models.SupportedModels[models.XAIGrok3Fast]),
					WithMaxTokens(500),
					WithSystemMessage("You are a helpful assistant."),
					WithOpenAIOptions(WithOpenAIToolChoice(tc.toolChoice)),
				)
				require.NoError(t, err)

				// Create a mock tool
				mockTool := &MockTool{
					name:        "get_info",
					description: "Get information about a topic",
					parameters: map[string]interface{}{
						"topic": map[string]interface{}{
							"type":        "string",
							"description": "The topic to get information about",
						},
					},
					required: []string{"topic"},
					response: `{"info": "some information"}`,
				}

				// Create message
				messages := []message.Message{
					{
						Role: message.User,
						Parts: []message.ContentPart{
							message.TextContent{Text: tc.message},
						},
					},
				}

				// Send request
				response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{mockTool})
				require.NoError(t, err)
				require.NotNil(t, response)

				// Verify expectations
				if tc.expectTool {
					assert.Equal(t, message.FinishReasonToolUse, response.FinishReason)
					assert.NotEmpty(t, response.ToolCalls)
				} else {
					assert.NotEqual(t, message.FinishReasonToolUse, response.FinishReason)
					assert.Empty(t, response.ToolCalls)
					assert.NotEmpty(t, response.Content) // Should have text response instead
				}
			})
		}
	})

	t.Run("parallel function calling", func(t *testing.T) {
		// Create provider with parallel tool calls enabled
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(1000),
			WithSystemMessage("You are a helpful assistant. When asked to do multiple things, use multiple tools in parallel if appropriate."),
			WithOpenAIOptions(WithOpenAIParallelToolCalls(true)),
		)
		require.NoError(t, err)

		// Create multiple mock tools
		weatherTool := &MockTool{
			name:        "get_weather",
			description: "Get the weather in a location",
			parameters: map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The location",
				},
			},
			required: []string{"location"},
			response: `{"temperature": 72}`,
		}

		timeTool := &MockTool{
			name:        "get_time",
			description: "Get the current time in a location",
			parameters: map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The location",
				},
			},
			required: []string{"location"},
			response: `{"time": "2:30 PM"}`,
		}

		// Create a message that could trigger multiple tools
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What's the weather and current time in both Paris and London?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{weatherTool, timeTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify function calling occurred
		assert.Equal(t, message.FinishReasonToolUse, response.FinishReason)
		assert.NotEmpty(t, response.ToolCalls)

		// Log the number of tool calls for observation
		t.Logf("Number of tool calls: %d", len(response.ToolCalls))
		for i, call := range response.ToolCalls {
			t.Logf("Tool call %d: %s", i+1, call.Name)
		}
	})

	t.Run("streaming with function calls", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant."),
		)
		require.NoError(t, err)

		// Create a mock tool
		mockTool := &MockTool{
			name:        "calculate",
			description: "Perform a calculation",
			parameters: map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "The mathematical expression",
				},
			},
			required: []string{"expression"},
			response: `{"result": 42}`,
		}

		// Create message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What is 6 times 7?"},
				},
			},
		}

		// Stream response
		eventChan := provider.StreamResponse(context.Background(), messages, []tools.BaseTool{mockTool})

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

		// According to xAI docs, function calls come in whole chunks in streaming
		if finalResponse.FinishReason == message.FinishReasonToolUse {
			assert.NotEmpty(t, finalResponse.ToolCalls)
			assert.NotEmpty(t, finalResponse.SystemFingerprint)
		} else {
			// If no tool was called, we should have content
			assert.True(t, hasContent || finalResponse.Content != "")
		}
	})

	t.Run("system fingerprint tracking", func(t *testing.T) {
		// Create xAI provider (which includes fingerprint tracking)
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(100),
			WithSystemMessage("You are a helpful assistant."),
		)
		require.NoError(t, err)

		// Send multiple requests to check fingerprint
		fingerprints := make([]string, 0)

		for i := 0; i < 3; i++ {
			messages := []message.Message{
				{
					Role: message.User,
					Parts: []message.ContentPart{
						message.TextContent{Text: "Say hello"},
					},
				},
			}

			response, err := provider.SendMessages(context.Background(), messages, nil)
			require.NoError(t, err)
			require.NotNil(t, response)

			// xAI should always return a system fingerprint
			assert.NotEmpty(t, response.SystemFingerprint)
			fingerprints = append(fingerprints, response.SystemFingerprint)

			t.Logf("Request %d - System fingerprint: %s", i+1, response.SystemFingerprint)
		}

		// Fingerprints might be the same or different depending on backend changes
		// We just verify they are populated
		for _, fp := range fingerprints {
			assert.NotEmpty(t, fp)
		}
	})
}

func TestXAIProvider_StructuredOutput(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("JSON mode", func(t *testing.T) {
		// Create provider with JSON mode
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant that responds in JSON format."),
			WithOpenAIOptions(WithOpenAIJSONMode()),
		)
		require.NoError(t, err)

		// Create message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Create a JSON object with a name and age for a fictional person."},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify response is valid JSON
		var result map[string]interface{}
		err = json.Unmarshal([]byte(response.Content), &result)
		assert.NoError(t, err, "Response should be valid JSON")
		assert.NotEmpty(t, result)
	})

	t.Run("JSON schema mode", func(t *testing.T) {
		// Define a schema
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Person's name",
				},
				"age": map[string]interface{}{
					"type":        "integer",
					"description": "Person's age",
					"minimum":     0,
					"maximum":     120,
				},
				"email": map[string]interface{}{
					"type":        "string",
					"description": "Email address",
					"format":      "email",
				},
			},
			"required": []string{"name", "age"},
		}

		// Create provider with JSON schema
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant that provides structured data."),
			WithOpenAIOptions(WithOpenAIJSONSchema("person_info", schema)),
		)
		require.NoError(t, err)

		// Create message
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Create information for a person named Alice who is 25 years old."},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify response matches schema
		var result map[string]interface{}
		err = json.Unmarshal([]byte(response.Content), &result)
		require.NoError(t, err, "Response should be valid JSON")

		// Check required fields
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "age")

		// Verify types
		_, nameIsString := result["name"].(string)
		assert.True(t, nameIsString, "name should be a string")

		// JSON numbers are parsed as float64
		age, ageIsNumber := result["age"].(float64)
		assert.True(t, ageIsNumber, "age should be a number")
		if ageIsNumber {
			assert.GreaterOrEqual(t, age, float64(0))
			assert.LessOrEqual(t, age, float64(120))
		}
	})
}

func TestXAIProvider_ConcurrentRequests(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("concurrent requests with rate limiting", func(t *testing.T) {
		// Create xAI client with max 2 concurrent requests
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     100,
			systemMessage: "You are a helpful assistant. Respond concisely.",
			xaiOptions: []XAIOption{
				WithMaxConcurrentRequests(2),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)
		require.NotNil(t, xaiClient.concurrent)

		// Track request timings
		var requestTimes sync.Map
		var requestCount int32

		// Create 5 concurrent requests
		numRequests := 5
		var wg sync.WaitGroup
		responses := make([]*ProviderResponse, numRequests)
		errors := make([]error, numRequests)

		startTime := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				reqStart := time.Now()
				atomic.AddInt32(&requestCount, 1)

				messages := []message.Message{
					{
						Role: message.User,
						Parts: []message.ContentPart{
							message.TextContent{Text: "Say hello"},
						},
					},
				}

				resp, err := xaiClient.send(context.Background(), messages, nil)

				reqDuration := time.Since(reqStart)
				requestTimes.Store(idx, reqDuration)

				responses[idx] = resp
				errors[idx] = err
			}(i)
		}

		wg.Wait()
		totalDuration := time.Since(startTime)

		// Verify all requests completed
		for i, err := range errors {
			require.NoError(t, err, "Request %d should not error", i)
			require.NotNil(t, responses[i], "Request %d should have response", i)
			assert.NotEmpty(t, responses[i].Content)
			assert.NotEmpty(t, responses[i].SystemFingerprint)
		}

		// Verify fingerprint tracking
		history := xaiClient.GetFingerprintHistory()
		assert.Len(t, history, numRequests)

		// Log timing information
		t.Logf("Total duration for %d requests: %v", numRequests, totalDuration)
		t.Logf("Max concurrent requests: %d", xaiClient.concurrent.GetMaxConcurrent())

		// Since we have max 2 concurrent requests, at least 3 batches should be needed
		// This is a rough check, actual timing depends on API response times
		t.Logf("Average time per request: %v", totalDuration/time.Duration(numRequests))
	})

	t.Run("batch requests", func(t *testing.T) {
		// Create xAI client with concurrent support
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     150,
			systemMessage: "You are a helpful assistant.",
			xaiOptions: []XAIOption{
				WithMaxConcurrentRequests(3),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)

		// Create batch requests
		requests := []BatchRequest{
			{
				Messages: []message.Message{{
					Role: message.User,
					Parts: []message.ContentPart{
						message.TextContent{Text: "What is 2+2?"},
					},
				}},
			},
			{
				Messages: []message.Message{{
					Role: message.User,
					Parts: []message.ContentPart{
						message.TextContent{Text: "What is the capital of France?"},
					},
				}},
			},
			{
				Messages: []message.Message{{
					Role: message.User,
					Parts: []message.ContentPart{
						message.TextContent{Text: "What color is the sky?"},
					},
				}},
			},
		}

		// Send batch
		responses := xaiClient.SendBatch(context.Background(), requests)

		// Verify all responses
		assert.Len(t, responses, 3)
		for i, resp := range responses {
			assert.NoError(t, resp.Error)
			assert.NotNil(t, resp.Response)
			assert.NotEmpty(t, resp.Response.Content)
			assert.Equal(t, i, resp.Index)
			t.Logf("Response %d: %s", i, resp.Response.Content)
		}
	})

	t.Run("streaming batch requests", func(t *testing.T) {
		// Create xAI client with concurrent support
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     100,
			systemMessage: "You are a helpful assistant. Respond concisely.",
			xaiOptions: []XAIOption{
				WithMaxConcurrentRequests(2),
			},
		}

		xaiClient := newXAIClient(opts).(*xaiClient)

		// Create batch streaming requests
		requests := []BatchRequest{
			{
				Messages: []message.Message{{
					Role: message.User,
					Parts: []message.ContentPart{
						message.TextContent{Text: "Count to 3"},
					},
				}},
			},
			{
				Messages: []message.Message{{
					Role: message.User,
					Parts: []message.ContentPart{
						message.TextContent{Text: "Say ABC"},
					},
				}},
			},
		}

		// Stream batch
		channels := xaiClient.StreamBatch(context.Background(), requests)
		assert.Len(t, channels, 2)

		// Collect responses from all streams
		var wg sync.WaitGroup
		responses := make([]*ProviderResponse, len(channels))

		for i, ch := range channels {
			wg.Add(1)
			go func(idx int, eventChan <-chan ProviderEvent) {
				defer wg.Done()

				for event := range eventChan {
					if event.Type == EventComplete {
						responses[idx] = event.Response
					} else if event.Type == EventError {
						t.Errorf("Stream %d error: %v", idx, event.Error)
					}
				}
			}(i, ch)
		}

		wg.Wait()

		// Verify all streams completed
		for i, resp := range responses {
			assert.NotNil(t, resp, "Stream %d should have response", i)
			assert.NotEmpty(t, resp.Content)
			assert.NotEmpty(t, resp.SystemFingerprint)
		}
	})

	t.Run("runtime max concurrent update", func(t *testing.T) {
		// Create xAI client without initial concurrent support
		opts := providerClientOptions{
			apiKey:        apiKey,
			model:         models.SupportedModels[models.XAIGrok3Fast],
			maxTokens:     50,
			systemMessage: "You are a helpful assistant.",
		}

		xaiClient := newXAIClient(opts).(*xaiClient)

		// Initially no concurrent client
		assert.Nil(t, xaiClient.concurrent)

		// Set max concurrent requests at runtime
		xaiClient.SetMaxConcurrentRequests(3)
		assert.NotNil(t, xaiClient.concurrent)
		assert.Equal(t, int64(3), xaiClient.concurrent.GetMaxConcurrent())

		// Update max concurrent requests
		xaiClient.SetMaxConcurrentRequests(5)
		assert.Equal(t, int64(5), xaiClient.concurrent.GetMaxConcurrent())

		// Test that it works
		messages := []message.Message{{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: "Hi"},
			},
		}}

		resp, err := xaiClient.send(context.Background(), messages, nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Content)
	})
}

func TestXAIProvider_LiveSearchFunctionality(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("live search with web search tool", func(t *testing.T) {
		// Create provider with Live Search enabled
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant. Use web search when you need current information."),
			WithXAIOptions(
				WithLiveSearch(),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		// Create a message that should trigger web search
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What are the latest developments in AI in 2025?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{webSearchTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Should have either tool calls or direct content with Live Search
		assert.True(t, len(response.ToolCalls) > 0 || response.Content != "")

		// Log response details
		t.Logf("Response content length: %d", len(response.Content))
		t.Logf("Tool calls: %d", len(response.ToolCalls))
		t.Logf("Citations: %d", len(response.Citations))

		if len(response.ToolCalls) > 0 {
			for i, call := range response.ToolCalls {
				t.Logf("Tool call %d: %s", i, call.Name)
				assert.Equal(t, "web_search", call.Name)
				assert.NotEmpty(t, call.Input)
			}
		}

		// Check for citations if present
		for i, citation := range response.Citations {
			t.Logf("Citation %d: %s", i+1, citation)
			assert.NotEmpty(t, citation)
		}
	})

	t.Run("live search without tools (direct integration)", func(t *testing.T) {
		// Create provider with Live Search enabled
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(300),
			WithSystemMessage("You are a helpful assistant with access to current information."),
			WithXAIOptions(
				WithLiveSearchOptions(LiveSearchOptions{
					Mode:             "on",
					MaxSearchResults: intPtr(5),
					ReturnCitations:  boolPtr(true),
					Sources: []LiveSearchSource{
						{Type: "web"},
						{Type: "news"},
					},
				}),
			),
		)
		require.NoError(t, err)

		// Create a message asking for current information (no tools provided)
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What's happening in the tech industry today?"},
				},
			},
		}

		// Send request without providing tools
		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Should have content since Live Search is enabled
		assert.NotEmpty(t, response.Content)

		// Log response details
		t.Logf("Direct Live Search response length: %d", len(response.Content))
		t.Logf("Citations: %d", len(response.Citations))

		// Should have citations since we enabled them
		if len(response.Citations) > 0 {
			t.Logf("Citations received: %v", response.Citations)
			for _, citation := range response.Citations {
				assert.NotEmpty(t, citation)
			}
		}
	})

	t.Run("live search combined with deferred completion", func(t *testing.T) {
		// Create provider with both Live Search and deferred completion
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a research assistant."),
			WithXAIOptions(
				WithLiveSearch(),
				WithDeferredCompletion(),
			),
		)
		require.NoError(t, err)

		// Create a complex research query
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Please provide a comprehensive analysis of recent AI safety research, including key papers, industry initiatives, and regulatory developments in 2025."},
				},
			},
		}

		// Send request (should use deferred completion with Live Search)
		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Should have substantial content
		assert.NotEmpty(t, response.Content)

		// Log response details
		t.Logf("Deferred + Live Search response length: %d", len(response.Content))
		t.Logf("Citations: %d", len(response.Citations))
		t.Logf("System fingerprint: %s", response.SystemFingerprint)
	})

	t.Run("live search with specific source filtering", func(t *testing.T) {
		// Create provider with specific source configuration
		customOpts := LiveSearchOptions{
			Mode:             "on",
			MaxSearchResults: intPtr(3),
			ReturnCitations:  boolPtr(true),
			Sources: []LiveSearchSource{
				{
					Type:             "x",
					IncludedXHandles: []string{"xai"},
				},
			},
		}

		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(300),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearchOptions(customOpts),
			),
		)
		require.NoError(t, err)

		// Ask about xAI specifically
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What are the latest announcements from xAI?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Should have content
		assert.NotEmpty(t, response.Content)

		// Log response details
		t.Logf("X-filtered Live Search response: %s", response.Content)
		t.Logf("Citations: %v", response.Citations)
	})

	t.Run("live search mode off", func(t *testing.T) {
		// Create provider with Live Search explicitly turned off
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(200),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearchOptions(LiveSearchOptions{
					Mode: "off",
				}),
			),
		)
		require.NoError(t, err)

		// Ask for current information
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What's the latest news in technology?"},
				},
			},
		}

		// Send request
		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should have system fingerprint
		assert.NotEmpty(t, response.SystemFingerprint)

		// Should have content (but based on training data, not live search)
		assert.NotEmpty(t, response.Content)

		// Should not have citations since Live Search is off
		assert.Empty(t, response.Citations)

		t.Logf("Live Search off response: %s", response.Content)
	})
}

func TestXAIProvider_LiveSearchIntegrationDetailed(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("basic live search integration", func(t *testing.T) {
		// Create provider with default Live Search
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

		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What are the latest AI developments in 2025?"},
				},
			},
		}

		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Validate response structure
		assert.NotEmpty(t, response.Content)
		assert.NotEmpty(t, response.SystemFingerprint)

		t.Logf("Basic Live Search Response:")
		t.Logf("  Response length: %d characters", len(response.Content))
		t.Logf("  Citations: %d", len(response.Citations))
		t.Logf("  System fingerprint: %s", response.SystemFingerprint)

		if len(response.Citations) > 0 {
			t.Logf("  Citations received:")
			for i, citation := range response.Citations {
				assert.NotEmpty(t, citation)
				t.Logf("    %d. %s", i+1, citation)
			}
		}
	})

	t.Run("custom live search parameters integration", func(t *testing.T) {
		// Create provider with custom Live Search options
		maxResults := 5
		returnCitations := true

		customOpts := LiveSearchOptions{
			Mode:             "on",
			MaxSearchResults: &maxResults,
			ReturnCitations:  &returnCitations,
			Sources: []LiveSearchSource{
				{Type: "web"},
				{Type: "news"},
			},
		}

		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(300),
			WithSystemMessage("You are a helpful assistant."),
			WithXAIOptions(
				WithLiveSearchOptions(customOpts),
			),
		)
		require.NoError(t, err)

		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What's happening in the tech industry today?"},
				},
			},
		}

		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Validate response structure
		assert.NotEmpty(t, response.Content)
		assert.NotEmpty(t, response.SystemFingerprint)

		t.Logf("Custom Live Search Response:")
		t.Logf("  Response length: %d characters", len(response.Content))
		t.Logf("  Citations: %d", len(response.Citations))
		t.Logf("  System fingerprint: %s", response.SystemFingerprint)

		// Should have citations since we enabled them and mode is "on"
		if returnCitations && len(response.Citations) > 0 {
			t.Logf("  Citations (as expected):")
			for i, citation := range response.Citations {
				assert.NotEmpty(t, citation)
				t.Logf("    %d. %s", i+1, citation)
			}
		}
	})

	t.Run("live search with web search tool integration", func(t *testing.T) {
		// Create provider with Live Search
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(500),
			WithSystemMessage("You are a helpful assistant. Use web search when you need current information."),
			WithXAIOptions(
				WithLiveSearch(),
			),
		)
		require.NoError(t, err)

		// Create web search tool
		webSearchTool := &tools.WebSearchTool{}

		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Search for recent news about artificial intelligence breakthroughs."},
				},
			},
		}

		response, err := provider.SendMessages(context.Background(), messages, []tools.BaseTool{webSearchTool})
		require.NoError(t, err)
		require.NotNil(t, response)

		// Validate response structure
		assert.NotEmpty(t, response.SystemFingerprint)
		// Should have either content or tool calls
		assert.True(t, response.Content != "" || len(response.ToolCalls) > 0)

		t.Logf("Tool-based Live Search Response:")
		t.Logf("  Response length: %d characters", len(response.Content))
		t.Logf("  Tool calls: %d", len(response.ToolCalls))
		t.Logf("  Citations: %d", len(response.Citations))
		t.Logf("  System fingerprint: %s", response.SystemFingerprint)

		if len(response.ToolCalls) > 0 {
			t.Logf("  Tool calls made:")
			for i, call := range response.ToolCalls {
				assert.Equal(t, "web_search", call.Name)
				assert.NotEmpty(t, call.Input)
				t.Logf("    %d. %s", i+1, call.Name)

				// Validate tool call input contains Live Search parameters
				var params map[string]interface{}
				err := json.Unmarshal([]byte(call.Input), &params)
				assert.NoError(t, err)
				assert.Contains(t, params, "query")
				t.Logf("       Query: %v", params["query"])
			}
		}

		// Validate citations if present
		for i, citation := range response.Citations {
			assert.NotEmpty(t, citation)
			t.Logf("  Citation %d: %s", i+1, citation)
		}
	})

	t.Run("live search comprehensive feature test", func(t *testing.T) {
		// Test with multiple advanced features
		customOpts := LiveSearchOptions{
			Mode:             "auto",
			MaxSearchResults: intPtr(3),
			ReturnCitations:  boolPtr(true),
			Sources: []LiveSearchSource{
				{
					Type:             "x",
					IncludedXHandles: []string{"xai"},
				},
				{
					Type:    "web",
					Country: stringPtr("US"),
				},
			},
		}

		// Combine with other xAI features
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
			WithMaxTokens(400),
			WithSystemMessage("You are a research assistant with access to current information."),
			WithXAIOptions(
				WithLiveSearchOptions(customOpts),
				WithMaxConcurrentRequests(2),
			),
		)
		require.NoError(t, err)

		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What are the latest updates from xAI and recent AI research?"},
				},
			},
		}

		response, err := provider.SendMessages(context.Background(), messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Validate comprehensive response
		assert.NotEmpty(t, response.Content)
		assert.NotEmpty(t, response.SystemFingerprint)

		t.Logf("Comprehensive Live Search Response:")
		t.Logf("  Response length: %d characters", len(response.Content))
		t.Logf("  Citations: %d", len(response.Citations))
		t.Logf("  System fingerprint: %s", response.SystemFingerprint)
		t.Logf("  First 200 chars: %s...",
			func() string {
				if len(response.Content) > 200 {
					return response.Content[:200]
				}
				return response.Content
			}())

		// Log citations for verification
		if len(response.Citations) > 0 {
			t.Logf("  Citations from X and web sources:")
			for i, citation := range response.Citations {
				assert.NotEmpty(t, citation)
				t.Logf("    %d. %s", i+1, citation)
			}
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
