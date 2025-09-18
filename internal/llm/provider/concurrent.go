package provider

import (
	"context"
	"sync"

	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"golang.org/x/sync/semaphore"
)

// ConcurrentClient wraps a ProviderClient to add concurrent request handling with rate limiting.
// It uses a semaphore to control the maximum number of concurrent requests and provides
// optional response tracking for monitoring and compliance purposes.
type ConcurrentClient struct {
	client      ProviderClient
	semaphore   *semaphore.Weighted
	maxInFlight int64
	mu          sync.RWMutex

	// Optional callback for tracking responses (e.g., xAI fingerprints)
	onResponse func(*ProviderResponse)
}

// NewConcurrentClient creates a new concurrent client wrapper with the specified max concurrent requests.
// If maxConcurrent is <= 0, it defaults to 10 concurrent requests.
func NewConcurrentClient(client ProviderClient, maxConcurrent int64) *ConcurrentClient {
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default to 10 concurrent requests
	}

	return &ConcurrentClient{
		client:      client,
		semaphore:   semaphore.NewWeighted(maxConcurrent),
		maxInFlight: maxConcurrent,
	}
}

// Send implements ProviderClient interface with semaphore-based rate limiting
func (c *ConcurrentClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	// Acquire semaphore
	if err := c.semaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer c.semaphore.Release(1)

	// Forward to underlying client
	resp, err := c.client.send(ctx, messages, tools)

	// Call callback if configured
	if c.onResponse != nil && resp != nil {
		c.onResponse(resp)
	}

	return resp, err
}

// Stream implements ProviderClient interface with semaphore-based rate limiting
func (c *ConcurrentClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		// Acquire semaphore
		if err := c.semaphore.Acquire(ctx, 1); err != nil {
			eventChan <- ProviderEvent{Type: EventError, Error: err}
			return
		}
		defer c.semaphore.Release(1)

		// Forward events from underlying client
		for event := range c.client.stream(ctx, messages, tools) {
			// Call callback for complete events if configured
			if c.onResponse != nil && event.Type == EventComplete && event.Response != nil {
				c.onResponse(event.Response)
			}
			eventChan <- event
		}
	}()

	return eventChan
}

// SetMaxConcurrent updates the maximum concurrent requests allowed.
// If max is <= 0, it defaults to 10.
func (c *ConcurrentClient) SetMaxConcurrent(max int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if max <= 0 {
		max = 10
	}

	c.maxInFlight = max
	c.semaphore = semaphore.NewWeighted(max)
}

// GetMaxConcurrent returns the current maximum concurrent requests setting
func (c *ConcurrentClient) GetMaxConcurrent() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxInFlight
}

// BatchRequest represents a single request in a batch
type BatchRequest struct {
	Messages []message.Message
	Tools    []tools.BaseTool
}

// BatchResponse represents the response for a batch request
type BatchResponse struct {
	Response *ProviderResponse
	Error    error
	Index    int
}

// SendBatch processes multiple requests concurrently respecting the semaphore limit
func (c *ConcurrentClient) SendBatch(ctx context.Context, requests []BatchRequest) []BatchResponse {
	responses := make([]BatchResponse, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request BatchRequest) {
			defer wg.Done()

			resp, err := c.send(ctx, request.Messages, request.Tools)
			responses[index] = BatchResponse{
				Response: resp,
				Error:    err,
				Index:    index,
			}
		}(i, req)
	}

	wg.Wait()
	return responses
}

// StreamBatch processes multiple streaming requests concurrently
func (c *ConcurrentClient) StreamBatch(ctx context.Context, requests []BatchRequest) []<-chan ProviderEvent {
	channels := make([]<-chan ProviderEvent, len(requests))

	for i, req := range requests {
		channels[i] = c.stream(ctx, req.Messages, req.Tools)
	}

	return channels
}
