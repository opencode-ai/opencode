package provider

import (
	"context"
	"time"

	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
)

// StreamingMetrics tracks streaming performance for xAI
type StreamingMetrics struct {
	FirstTokenTime    time.Duration
	TotalStreamTime   time.Duration
	TokenCount        int
	ChunkCount        int
	SystemFingerprint string
}

// streamWithMetrics wraps the base streaming with xAI-specific metrics and handling
func (x *xaiClient) streamWithMetrics(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	// Use concurrent client if configured
	if x.concurrent != nil {
		return x.concurrent.stream(ctx, messages, tools)
	}

	// Get the base stream
	baseChan := x.openaiClient.stream(ctx, messages, tools)

	// Create a new channel to intercept and process events
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		startTime := time.Now()
		var firstTokenTime time.Duration
		var metrics StreamingMetrics
		tokenCount := 0
		chunkCount := 0

		for event := range baseChan {
			// Track metrics
			switch event.Type {
			case EventContentDelta:
				chunkCount++
				if tokenCount == 0 {
					firstTokenTime = time.Since(startTime)
					metrics.FirstTokenTime = firstTokenTime
					logging.Debug("xAI streaming first token received",
						"model", x.providerOptions.model.ID,
						"time_to_first_token", firstTokenTime)
				}
				tokenCount += len(event.Content)

			case EventComplete:
				if event.Response != nil {
					metrics.TotalStreamTime = time.Since(startTime)
					metrics.TokenCount = tokenCount
					metrics.ChunkCount = chunkCount
					metrics.SystemFingerprint = event.Response.SystemFingerprint

					// Track fingerprint for monitoring, security, and compliance
					if event.Response.SystemFingerprint != "" {
						x.trackFingerprint(event.Response.SystemFingerprint, event.Response.Usage)
					}

					// Log streaming metrics
					logging.Debug("xAI streaming completed",
						"model", x.providerOptions.model.ID,
						"total_time", metrics.TotalStreamTime,
						"first_token_time", metrics.FirstTokenTime,
						"chunks", metrics.ChunkCount,
						"usage", event.Response.Usage,
						"system_fingerprint", metrics.SystemFingerprint)
				}
			}

			// Forward the event
			eventChan <- event
		}
	}()

	return eventChan
}

// EnhancedStreamOptions provides xAI-specific streaming configuration
type EnhancedStreamOptions struct {
	// TimeoutOverride allows manual timeout override for reasoning models
	TimeoutOverride *time.Duration

	// EnableMetrics enables detailed streaming metrics
	EnableMetrics bool

	// BufferSize controls the event channel buffer size
	BufferSize int
}

// streamWithOptions provides enhanced streaming with xAI-specific options
func (x *xaiClient) streamWithOptions(ctx context.Context, messages []message.Message, tools []tools.BaseTool, opts EnhancedStreamOptions) <-chan ProviderEvent {
	// Apply timeout override if specified (useful for reasoning models)
	if opts.TimeoutOverride != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *opts.TimeoutOverride)
		defer cancel()
	}

	// Use metrics-enabled streaming if requested
	if opts.EnableMetrics {
		return x.streamWithMetrics(ctx, messages, tools)
	}

	// Otherwise use standard streaming
	return x.stream(ctx, messages, tools)
}

// ValidateStreamingSupport checks if the model supports streaming
func (x *xaiClient) ValidateStreamingSupport() error {
	// All xAI chat models support streaming
	// This is a placeholder for future model-specific validation
	return nil
}
