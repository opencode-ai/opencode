package provider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProviderClient implements ProviderClient for testing
type mockProviderClient struct {
	sendCount    int32
	streamCount  int32
	sendDelay    time.Duration
	streamDelay  time.Duration
	sendError    error
	streamError  error
	responseFunc func() *ProviderResponse
}

func (m *mockProviderClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	atomic.AddInt32(&m.sendCount, 1)
	if m.sendDelay > 0 {
		time.Sleep(m.sendDelay)
	}
	if m.sendError != nil {
		return nil, m.sendError
	}
	if m.responseFunc != nil {
		return m.responseFunc(), nil
	}
	return &ProviderResponse{
		Content:      "test response",
		FinishReason: message.FinishReasonEndTurn,
		Usage: TokenUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}, nil
}

func (m *mockProviderClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	atomic.AddInt32(&m.streamCount, 1)
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		if m.streamDelay > 0 {
			time.Sleep(m.streamDelay)
		}

		if m.streamError != nil {
			eventChan <- ProviderEvent{Type: EventError, Error: m.streamError}
			return
		}

		// Send some content deltas
		eventChan <- ProviderEvent{Type: EventContentDelta, Content: "test "}
		eventChan <- ProviderEvent{Type: EventContentDelta, Content: "response"}

		// Send complete event
		resp := &ProviderResponse{
			Content:      "test response",
			FinishReason: message.FinishReasonEndTurn,
			Usage: TokenUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}
		if m.responseFunc != nil {
			resp = m.responseFunc()
		}
		eventChan <- ProviderEvent{Type: EventComplete, Response: resp}
	}()

	return eventChan
}

func TestConcurrentClient_Send(t *testing.T) {
	tests := []struct {
		name          string
		maxConcurrent int64
		numRequests   int
		requestDelay  time.Duration
		expectError   bool
	}{
		{
			name:          "single request",
			maxConcurrent: 1,
			numRequests:   1,
			requestDelay:  0,
			expectError:   false,
		},
		{
			name:          "multiple concurrent requests within limit",
			maxConcurrent: 5,
			numRequests:   5,
			requestDelay:  10 * time.Millisecond,
			expectError:   false,
		},
		{
			name:          "requests exceed concurrent limit",
			maxConcurrent: 2,
			numRequests:   10,
			requestDelay:  50 * time.Millisecond,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockProviderClient{
				sendDelay: tt.requestDelay,
			}

			concurrentClient := NewConcurrentClient(mockClient, tt.maxConcurrent)

			ctx := context.Background()
			messages := []message.Message{}
			tools := []tools.BaseTool{}

			var wg sync.WaitGroup
			results := make([]*ProviderResponse, tt.numRequests)
			errors := make([]error, tt.numRequests)

			start := time.Now()

			for i := 0; i < tt.numRequests; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					resp, err := concurrentClient.send(ctx, messages, tools)
					results[idx] = resp
					errors[idx] = err
				}(i)
			}

			wg.Wait()
			elapsed := time.Since(start)

			// Check all requests completed
			for i, err := range errors {
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, results[i])
					assert.Equal(t, "test response", results[i].Content)
				}
			}

			// Verify semaphore worked by checking timing
			if tt.numRequests > int(tt.maxConcurrent) && tt.requestDelay > 0 {
				expectedMinTime := tt.requestDelay * time.Duration(tt.numRequests/int(tt.maxConcurrent))
				assert.GreaterOrEqual(t, elapsed, expectedMinTime)
			}

			// Verify all requests were made
			assert.Equal(t, int32(tt.numRequests), atomic.LoadInt32(&mockClient.sendCount))
		})
	}
}

func TestConcurrentClient_Stream(t *testing.T) {
	mockClient := &mockProviderClient{
		streamDelay: 10 * time.Millisecond,
	}

	concurrentClient := NewConcurrentClient(mockClient, 2)

	ctx := context.Background()
	messages := []message.Message{}
	tools := []tools.BaseTool{}

	// Start multiple streaming requests
	numStreams := 5
	var wg sync.WaitGroup
	results := make([][]ProviderEvent, numStreams)

	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			eventChan := concurrentClient.stream(ctx, messages, tools)
			var events []ProviderEvent

			for event := range eventChan {
				events = append(events, event)
			}

			results[idx] = events
		}(i)
	}

	wg.Wait()

	// Verify all streams completed successfully
	for i, events := range results {
		require.NotEmpty(t, events, "stream %d should have events", i)

		// Should have content deltas and complete event
		var hasContentDelta, hasComplete bool
		for _, event := range events {
			if event.Type == EventContentDelta {
				hasContentDelta = true
			}
			if event.Type == EventComplete {
				hasComplete = true
				assert.NotNil(t, event.Response)
				assert.Equal(t, "test response", event.Response.Content)
			}
		}

		assert.True(t, hasContentDelta, "stream %d should have content delta", i)
		assert.True(t, hasComplete, "stream %d should have complete event", i)
	}

	// Verify all streams were made
	assert.Equal(t, int32(numStreams), atomic.LoadInt32(&mockClient.streamCount))
}

func TestConcurrentClient_Callback(t *testing.T) {
	var callbackCount int32
	var lastResponse *ProviderResponse

	mockClient := &mockProviderClient{
		responseFunc: func() *ProviderResponse {
			return &ProviderResponse{
				Content:           "test",
				SystemFingerprint: "test-fingerprint",
				Usage: TokenUsage{
					InputTokens:  5,
					OutputTokens: 10,
				},
			}
		},
	}

	concurrentClient := NewConcurrentClient(mockClient, 1)
	concurrentClient.onResponse = func(resp *ProviderResponse) {
		atomic.AddInt32(&callbackCount, 1)
		lastResponse = resp
	}

	ctx := context.Background()

	// Test send callback
	resp, err := concurrentClient.send(ctx, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackCount))
	assert.Equal(t, "test-fingerprint", lastResponse.SystemFingerprint)

	// Test stream callback
	eventChan := concurrentClient.stream(ctx, nil, nil)
	var events []ProviderEvent
	for event := range eventChan {
		events = append(events, event)
	}

	assert.Equal(t, int32(2), atomic.LoadInt32(&callbackCount))
	assert.Equal(t, "test-fingerprint", lastResponse.SystemFingerprint)
}

func TestConcurrentClient_BatchRequests(t *testing.T) {
	mockClient := &mockProviderClient{
		sendDelay: 20 * time.Millisecond,
	}

	concurrentClient := NewConcurrentClient(mockClient, 3)

	ctx := context.Background()

	// Create batch requests
	requests := make([]BatchRequest, 10)
	for i := range requests {
		requests[i] = BatchRequest{
			Messages: []message.Message{},
			Tools:    []tools.BaseTool{},
		}
	}

	start := time.Now()
	responses := concurrentClient.SendBatch(ctx, requests)
	elapsed := time.Since(start)

	// Verify all responses
	assert.Len(t, responses, 10)
	for i, resp := range responses {
		assert.NoError(t, resp.Error)
		assert.NotNil(t, resp.Response)
		assert.Equal(t, i, resp.Index)
	}

	// Verify semaphore worked (10 requests / 3 concurrent = at least 4 batches)
	expectedMinTime := 4 * 20 * time.Millisecond
	assert.GreaterOrEqual(t, elapsed, expectedMinTime)
}

func TestConcurrentClient_SetMaxConcurrent(t *testing.T) {
	mockClient := &mockProviderClient{}
	concurrentClient := NewConcurrentClient(mockClient, 2)

	assert.Equal(t, int64(2), concurrentClient.GetMaxConcurrent())

	concurrentClient.SetMaxConcurrent(5)
	assert.Equal(t, int64(5), concurrentClient.GetMaxConcurrent())

	// Test with invalid value
	concurrentClient.SetMaxConcurrent(0)
	assert.Equal(t, int64(10), concurrentClient.GetMaxConcurrent()) // Should default to 10
}
