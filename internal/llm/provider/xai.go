package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
)

const (
	// defaultXAITimeout is the standard timeout for regular API requests
	defaultXAITimeout = 30 * time.Second
	// reasoningXAITimeout is an extended timeout for reasoning models
	// which can take several minutes to process complex requests
	reasoningXAITimeout = 5 * time.Minute
	// defaultXAIBaseURL is the default base URL for xAI API
	defaultXAIBaseURL = "https://api.x.ai/v1"
)

// FingerprintRecord tracks system fingerprint information for auditing and compliance purposes.
// It helps monitor xAI system changes and optimize caching performance.
type FingerprintRecord struct {
	Fingerprint string     `json:"fingerprint"`
	Timestamp   time.Time  `json:"timestamp"`
	Model       string     `json:"model"`
	TokensUsed  TokenUsage `json:"tokens_used"`
}

// xaiClient wraps the OpenAI client with xAI-specific functionality.
// It provides enhanced features like deferred completions, concurrent request handling,
// Live Search integration, and comprehensive fingerprint tracking for monitoring.
type xaiClient struct {
	openaiClient
	mu                 sync.Mutex
	lastFingerprint    string
	fingerprintHistory []FingerprintRecord // For compliance and auditing
	concurrent         *ConcurrentClient   // Optional concurrent request handler
	deferredEnabled    bool                // Enable deferred completions
	deferredOptions    DeferredOptions     // Options for deferred completions
	liveSearchEnabled  bool                // Enable Live Search
	liveSearchOptions  LiveSearchOptions   // Options for Live Search

	// New architectural components
	reasoningHandler *ReasoningHandler // Handles reasoning content processing
	httpClient       *XAIHTTPClient    // Custom HTTP client for xAI API
}

type XAIClient ProviderClient

// XAIOption represents xAI-specific configuration options
type XAIOption func(*xaiClient)

// WithMaxConcurrentRequests configures the maximum number of concurrent requests
func WithMaxConcurrentRequests(maxConcurrent int64) XAIOption {
	return func(x *xaiClient) {
		x.concurrent = NewConcurrentClient(x, maxConcurrent)
		// Set up callback to track fingerprints from concurrent requests
		x.concurrent.onResponse = func(resp *ProviderResponse) {
			if resp != nil && resp.SystemFingerprint != "" {
				x.trackFingerprint(resp.SystemFingerprint, resp.Usage)
			}
		}
	}
}

// WithDeferredCompletion enables deferred completion mode
func WithDeferredCompletion() XAIOption {
	return func(x *xaiClient) {
		x.deferredEnabled = true
		x.deferredOptions = DefaultDeferredOptions()
	}
}

// WithDeferredOptions configures deferred completion options
func WithDeferredOptions(timeout, interval time.Duration) XAIOption {
	return func(x *xaiClient) {
		x.deferredOptions = DeferredOptions{
			Timeout:  timeout,
			Interval: interval,
		}
	}
}

// WithLiveSearch enables Live Search with default parameters
func WithLiveSearch() XAIOption {
	return func(x *xaiClient) {
		x.liveSearchEnabled = true
		x.liveSearchOptions = DefaultLiveSearchOptions()
	}
}

// WithLiveSearchOptions enables Live Search with custom parameters
func WithLiveSearchOptions(opts LiveSearchOptions) XAIOption {
	return func(x *xaiClient) {
		x.liveSearchEnabled = true
		x.liveSearchOptions = opts
	}
}

func newXAIClient(opts providerClientOptions) XAIClient {
	// Create base OpenAI client with xAI-specific settings
	opts.openaiOptions = append(opts.openaiOptions,
		WithOpenAIBaseURL(defaultXAIBaseURL),
	)

	baseClient := newOpenAIClient(opts)
	openaiClientImpl, ok := baseClient.(*openaiClient)
	if !ok {
		// This is a programming error - xAI client extends openAI client
		panic("internal error: xAI client requires openaiClient implementation")
	}

	xClient := &xaiClient{
		openaiClient:       *openaiClientImpl,
		fingerprintHistory: make([]FingerprintRecord, 0),
	}

	// Initialize new architectural components
	xClient.reasoningHandler = NewReasoningHandler(xClient)

	// Use the base URL from OpenAI client options if set, otherwise default to xAI API
	baseURL := defaultXAIBaseURL
	if openaiClientImpl.options.baseURL != "" {
		baseURL = openaiClientImpl.options.baseURL
	}

	// Configure HTTP client with appropriate timeout based on model capabilities
	timeout := selectTimeoutForModel(opts.model)

	xClient.httpClient = NewXAIHTTPClient(HTTPClientConfig{
		BaseURL:   baseURL,
		APIKey:    opts.apiKey,
		UserAgent: "opencode/1.0",
		Timeout:   timeout,
	})

	// Apply xAI-specific options if any
	for _, opt := range opts.xaiOptions {
		opt(xClient)
	}

	return xClient
}

// selectTimeoutForModel returns the appropriate timeout based on model capabilities
func selectTimeoutForModel(model models.Model) time.Duration {
	if model.CanReason {
		return reasoningXAITimeout
	}
	return defaultXAITimeout
}

// shouldApplyReasoningEffort overrides the base implementation for xAI-specific logic
func (x *xaiClient) shouldApplyReasoningEffort() bool {
	// Grok-4 has internal reasoning capabilities but doesn't accept the reasoning_effort parameter
	// or expose its reasoning process. Other xAI thinking models accept reasoning_effort with values "low" or "high"
	if x.providerOptions.model.ID == models.XAIGrok4 {
		return false
	}
	return true
}

// trackFingerprint records fingerprint for monitoring, security, and compliance
func (x *xaiClient) trackFingerprint(fingerprint string, usage TokenUsage) {
	if fingerprint == "" {
		return
	}

	x.mu.Lock()
	defer x.mu.Unlock()

	// Record for audit trail
	record := FingerprintRecord{
		Fingerprint: fingerprint,
		Timestamp:   time.Now(),
		Model:       string(x.providerOptions.model.ID),
		TokensUsed:  usage,
	}
	x.fingerprintHistory = append(x.fingerprintHistory, record)

	// Log for monitoring system changes
	if x.lastFingerprint != "" && x.lastFingerprint != fingerprint {
		// System configuration changed - important for debugging and performance optimization
		logging.Info("xAI system configuration changed",
			"previous", x.lastFingerprint,
			"current", fingerprint,
			"model", x.providerOptions.model.ID,
			"timestamp", record.Timestamp.Format(time.RFC3339))
	}

	// Calculate caching efficiency
	totalPromptTokens := usage.InputTokens + usage.CacheReadTokens
	cacheHitRate := float64(0)
	if totalPromptTokens > 0 {
		cacheHitRate = float64(usage.CacheReadTokens) / float64(totalPromptTokens) * 100
	}

	// Log enhanced metrics including caching information
	logFields := []interface{}{
		"fingerprint", fingerprint,
		"model", x.providerOptions.model.ID,
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"cache_read_tokens", usage.CacheReadTokens,
		"cache_creation_tokens", usage.CacheCreationTokens,
		"total_prompt_tokens", totalPromptTokens,
		"timestamp", record.Timestamp.Format(time.RFC3339),
	}

	// Add cache efficiency metrics if caching is happening
	if usage.CacheReadTokens > 0 {
		logFields = append(logFields,
			"cache_hit_rate_percent", cacheHitRate,
			"cache_cost_savings", x.calculateCacheCostSavings(usage))

		logging.Info("xAI prompt caching active", logFields...)
	} else {
		logging.Debug("xAI API response tracked", logFields...)
	}

	x.lastFingerprint = fingerprint
}

// calculateCacheCostSavings estimates cost savings from prompt caching
func (x *xaiClient) calculateCacheCostSavings(usage TokenUsage) float64 {
	// Get model pricing (cost per 1M tokens)
	model := x.providerOptions.model
	costPer1MIn := model.CostPer1MIn
	costPer1MInCached := model.CostPer1MInCached

	// If cached pricing isn't set, assume significant savings (typically 50% discount)
	if costPer1MInCached == 0 {
		costPer1MInCached = costPer1MIn * 0.5
	}

	// Calculate savings: (regular_cost - cached_cost) * tokens / 1M
	if usage.CacheReadTokens > 0 {
		regularCost := (costPer1MIn * float64(usage.CacheReadTokens)) / 1_000_000
		cachedCost := (costPer1MInCached * float64(usage.CacheReadTokens)) / 1_000_000
		return regularCost - cachedCost
	}

	return 0
}

func (x *xaiClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	// Use deferred completion if enabled
	if x.deferredEnabled {
		logging.Debug("Using deferred completion")
		return x.SendDeferred(ctx, messages, tools, x.deferredOptions)
	}

	// Use custom HTTP client for Live Search in regular completions
	if x.liveSearchEnabled {
		logging.Debug("Using live search")
		return x.sendWithLiveSearch(ctx, messages, tools)
	}

	// Use reasoning handler for models with reasoning capability
	if x.reasoningHandler.ShouldUseReasoning() {
		logging.Debug("Using reasoning handler for model",
			"model", x.providerOptions.model.ID,
			"reasoning_effort", x.options.reasoningEffort)
		return x.sendWithReasoningSupport(ctx, messages, tools)
	}

	// Use concurrent client if configured
	if x.concurrent != nil {
		logging.Debug("Using concurrent client")
		return x.concurrent.send(ctx, messages, tools)
	}

	// Call the base OpenAI implementation
	logging.Debug("Using base OpenAI implementation")
	response, err := x.openaiClient.send(ctx, messages, tools)
	if err != nil {
		return nil, err
	}

	// Track fingerprint for monitoring, security, and compliance
	if response.SystemFingerprint != "" {
		x.trackFingerprint(response.SystemFingerprint, response.Usage)
	}

	return response, nil
}

// sendWithReasoningSupport sends a request using the reasoning handler
func (x *xaiClient) sendWithReasoningSupport(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	// Build request body using reasoning handler
	reqBody := x.reasoningHandler.BuildReasoningRequest(ctx, messages, tools)

	// Log the request for debugging
	logging.Debug("Sending reasoning request",
		"model", reqBody["model"],
		"reasoning_effort", reqBody["reasoning_effort"],
		"messages_count", len(messages))

	// Send the request using HTTP client
	result, err := x.httpClient.SendCompletionRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("reasoning request failed: %w", err)
	}

	// Convert result to ProviderResponse
	response := x.convertDeferredResult(result)

	// Store reasoning content in the response for stream processing
	if len(result.Choices) > 0 && result.Choices[0].Message.ReasoningContent != "" {
		response.ReasoningContent = result.Choices[0].Message.ReasoningContent
	}

	return response, nil
}

// sendWithLiveSearch sends a regular completion request with Live Search parameters
func (x *xaiClient) sendWithLiveSearch(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	// Build request similar to deferred completions but without the deferred flag
	reqBody := map[string]interface{}{
		"model":      x.providerOptions.model.APIModel,
		"messages":   x.convertMessagesToAPI(messages), // Use the deferred method for proper conversion
		"max_tokens": x.providerOptions.maxTokens,      // Don't use pointer
	}

	// Add tools if provided
	if len(tools) > 0 {
		reqBody["tools"] = x.convertToolsToAPI(tools)
	}

	// Temperature is not configurable in the current implementation

	// Apply reasoning effort if applicable
	if x.shouldApplyReasoningEffort() && x.options.reasoningEffort != "" {
		reqBody["reasoning_effort"] = x.options.reasoningEffort
	}

	// Apply response format if configured
	if x.options.responseFormat != nil {
		reqBody["response_format"] = x.options.responseFormat
	}

	// Apply tool choice if configured
	if x.options.toolChoice != nil {
		reqBody["tool_choice"] = x.options.toolChoice
	}

	// Apply parallel tool calls if configured
	if x.options.parallelToolCalls != nil {
		reqBody["parallel_tool_calls"] = x.options.parallelToolCalls
	}

	// Add Live Search parameters only if enabled
	if x.liveSearchEnabled {
		reqBody["search_parameters"] = x.liveSearchOptions
	}

	// Log the request for debugging
	logging.Debug("Sending custom HTTP request",
		"model", reqBody["model"],
		"reasoning_effort", reqBody["reasoning_effort"],
		"messages_count", len(x.convertMessagesToAPI(messages)))

	// Send the request using HTTP client
	result, err := x.httpClient.SendCompletionRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("live search request failed: %w", err)
	}

	// Convert result to ProviderResponse
	return x.convertDeferredResult(result), nil
}

func (x *xaiClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	// Use concurrent client if configured
	if x.concurrent != nil {
		return x.concurrent.stream(ctx, messages, tools)
	}

	// Use reasoning handler for models with reasoning capability
	if x.reasoningHandler.ShouldUseReasoning() {
		return x.streamWithReasoning(ctx, messages, tools)
	}

	// Get the base stream
	baseChan := x.openaiClient.stream(ctx, messages, tools)

	// Create a new channel to intercept and process events
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		for event := range baseChan {
			// If this is a complete event with a response, track the fingerprint
			if event.Type == EventComplete && event.Response != nil && event.Response.SystemFingerprint != "" {
				x.trackFingerprint(event.Response.SystemFingerprint, event.Response.Usage)
			}

			// Forward the event
			eventChan <- event
		}
	}()

	return eventChan
}

// streamWithReasoning handles streaming for reasoning models
func (x *xaiClient) streamWithReasoning(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	logging.Debug("Using reasoning handler for stream",
		"model", x.providerOptions.model.ID,
		"reasoning_effort", x.options.reasoningEffort)

	// Create a channel to return events
	eventChan := make(chan ProviderEvent)

	go func() {
		defer close(eventChan)

		defer func() {
			if r := recover(); r != nil {
				logging.Error("Panic in reasoning stream", "panic", r)
				eventChan <- ProviderEvent{
					Type:  EventError,
					Error: fmt.Errorf("panic in reasoning stream: %v", r),
				}
			}
		}()

		// Check context first
		select {
		case <-ctx.Done():
			logging.Debug("Context cancelled before reasoning request", "error", ctx.Err())
			eventChan <- ProviderEvent{
				Type:  EventError,
				Error: ctx.Err(),
			}
			return
		default:
		}

		logging.Debug("Starting reasoning request")

		// Get response using reasoning support
		response, err := x.sendWithReasoningSupport(ctx, messages, tools)
		if err != nil {
			logging.Error("Reasoning request failed", "error", err)
			eventChan <- ProviderEvent{
				Type:  EventError,
				Error: err,
			}
			return
		}

		// Process response using reasoning handler
		events := x.reasoningHandler.ProcessReasoningResponse(response)

		// Send all events
		for _, event := range events {
			eventChan <- event
		}
	}()

	return eventChan
}

// GetFingerprintHistory returns the fingerprint history for auditing and compliance
func (x *xaiClient) GetFingerprintHistory() []FingerprintRecord {
	x.mu.Lock()
	defer x.mu.Unlock()

	// Return a copy to prevent external modification
	history := make([]FingerprintRecord, len(x.fingerprintHistory))
	copy(history, x.fingerprintHistory)
	return history
}

// GetCurrentFingerprint returns the current system fingerprint
func (x *xaiClient) GetCurrentFingerprint() string {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.lastFingerprint
}

// SendBatch processes multiple requests concurrently if concurrent client is configured
func (x *xaiClient) SendBatch(ctx context.Context, requests []BatchRequest) []BatchResponse {
	if x.concurrent != nil {
		return x.concurrent.SendBatch(ctx, requests)
	}

	// Fallback to sequential processing if no concurrent client
	responses := make([]BatchResponse, len(requests))
	for i, req := range requests {
		resp, err := x.send(ctx, req.Messages, req.Tools)
		responses[i] = BatchResponse{
			Response: resp,
			Error:    err,
			Index:    i,
		}
	}
	return responses
}

// StreamBatch processes multiple streaming requests concurrently if configured
func (x *xaiClient) StreamBatch(ctx context.Context, requests []BatchRequest) []<-chan ProviderEvent {
	if x.concurrent != nil {
		return x.concurrent.StreamBatch(ctx, requests)
	}

	// Fallback to sequential processing if no concurrent client
	channels := make([]<-chan ProviderEvent, len(requests))
	for i, req := range requests {
		channels[i] = x.stream(ctx, req.Messages, req.Tools)
	}
	return channels
}

// convertMessages overrides the base implementation to support xAI-specific image handling
func (x *xaiClient) convertMessages(messages []message.Message) (openaiMessages []openai.ChatCompletionMessageParamUnion) {
	// Add system message first
	openaiMessages = append(openaiMessages, openai.SystemMessage(x.providerOptions.systemMessage))

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam

			// Add text content if present
			if msg.Content().String() != "" {
				textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})
			}

			// Add binary content (base64 encoded images)
			for _, binaryContent := range msg.BinaryContent() {
				// xAI expects data URLs in format: data:image/jpeg;base64,<base64_string>
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{
					URL:    binaryContent.String(models.ProviderOpenAI), // This already formats as data URL
					Detail: "high",                                      // Default to high detail for better recognition
				}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}

			// Add image URL content (web URLs)
			for _, imageURLContent := range msg.ImageURLContent() {
				detail := imageURLContent.Detail
				if detail == "" {
					detail = "auto" // Default to auto if not specified
				}
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{
					URL:    imageURLContent.URL,
					Detail: detail,
				}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}

			openaiMessages = append(openaiMessages, openai.UserMessage(content))

		case message.Assistant:
			// Use base implementation for assistant messages
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}

			if msg.Content().String() != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content().String()),
				}
			}

			if len(msg.ToolCalls()) > 0 {
				assistantMsg.ToolCalls = make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls()))
				for i, call := range msg.ToolCalls() {
					assistantMsg.ToolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID:   call.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Input,
						},
					}
				}
			}

			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

		case message.Tool:
			for _, result := range msg.ToolResults() {
				openaiMessages = append(openaiMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

// IsVisionCapable returns true if the current model supports image input
func (x *xaiClient) IsVisionCapable() bool {
	return x.providerOptions.model.SupportsAttachments
}

// SetMaxConcurrentRequests updates the maximum concurrent requests at runtime
func (x *xaiClient) SetMaxConcurrentRequests(maxConcurrent int64) {
	if x.concurrent == nil {
		x.concurrent = NewConcurrentClient(x, maxConcurrent)
	} else {
		x.concurrent.SetMaxConcurrent(maxConcurrent)
	}
}
