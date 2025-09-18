package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
)

// DeferredCompletionRequest represents the request body for deferred completions
type DeferredCompletionRequest struct {
	Model             string                   `json:"model"`
	Messages          []map[string]interface{} `json:"messages"`
	MaxTokens         *int64                   `json:"max_tokens,omitempty"`
	Temperature       *float64                 `json:"temperature,omitempty"`
	Tools             []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice        interface{}              `json:"tool_choice,omitempty"`
	ResponseFormat    interface{}              `json:"response_format,omitempty"`
	Deferred          bool                     `json:"deferred"`
	ReasoningEffort   string                   `json:"reasoning_effort,omitempty"`
	ParallelToolCalls *bool                    `json:"parallel_tool_calls,omitempty"`
	SearchParameters  *LiveSearchOptions       `json:"search_parameters,omitempty"`
}

// DeferredCompletionResponse represents the initial response with request_id
type DeferredCompletionResponse struct {
	RequestID string `json:"request_id"`
}

// DeferredResult represents the final deferred completion result
type DeferredResult struct {
	ID                string           `json:"id"`
	Object            string           `json:"object"`
	Created           int64            `json:"created"`
	Model             string           `json:"model"`
	Choices           []DeferredChoice `json:"choices"`
	Usage             DeferredUsage    `json:"usage"`
	SystemFingerprint string           `json:"system_fingerprint"`
	Citations         []string         `json:"citations,omitempty"`
}

// DeferredChoice represents a choice in the deferred result
type DeferredChoice struct {
	Index        int             `json:"index"`
	Message      DeferredMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// DeferredMessage represents a message in the deferred result
type DeferredMessage struct {
	Role             string             `json:"role"`
	Content          string             `json:"content,omitempty"`
	ReasoningContent string             `json:"reasoning_content,omitempty"`
	ToolCalls        []DeferredToolCall `json:"tool_calls,omitempty"`
}

// DeferredToolCall represents a tool call in the deferred result
type DeferredToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function DeferredToolFunction `json:"function"`
}

// DeferredToolFunction represents a tool function call
type DeferredToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// DeferredUsage represents token usage in the deferred result
type DeferredUsage struct {
	PromptTokens            int64                            `json:"prompt_tokens"`
	CompletionTokens        int64                            `json:"completion_tokens"`
	TotalTokens             int64                            `json:"total_tokens"`
	PromptTokensDetails     *DeferredPromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *DeferredCompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	NumSourcesUsed          int64                            `json:"num_sources_used,omitempty"`
}

// DeferredPromptTokensDetails represents detailed prompt token usage
type DeferredPromptTokensDetails struct {
	TextTokens   int64 `json:"text_tokens"`
	AudioTokens  int64 `json:"audio_tokens"`
	ImageTokens  int64 `json:"image_tokens"`
	CachedTokens int64 `json:"cached_tokens"`
}

// DeferredCompletionTokensDetails represents detailed completion token usage
type DeferredCompletionTokensDetails struct {
	ReasoningTokens          int64 `json:"reasoning_tokens"`
	AudioTokens              int64 `json:"audio_tokens"`
	AcceptedPredictionTokens int64 `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int64 `json:"rejected_prediction_tokens"`
}

// DeferredOptions represents options for deferred completions
type DeferredOptions struct {
	Timeout  time.Duration
	Interval time.Duration
}

// DefaultDeferredOptions returns default options for deferred completions
func DefaultDeferredOptions() DeferredOptions {
	return DeferredOptions{
		Timeout:  10 * time.Minute,
		Interval: 10 * time.Second,
	}
}

// LiveSearchOptions represents options for Live Search
type LiveSearchOptions struct {
	Mode             string             `json:"mode,omitempty"`               // "auto", "on", "off"
	MaxSearchResults *int               `json:"max_search_results,omitempty"` // 1-20, default 20
	FromDate         *string            `json:"from_date,omitempty"`          // YYYY-MM-DD
	ToDate           *string            `json:"to_date,omitempty"`            // YYYY-MM-DD
	ReturnCitations  *bool              `json:"return_citations,omitempty"`   // default true
	Sources          []LiveSearchSource `json:"sources,omitempty"`            // Data sources
}

// LiveSearchSource represents a data source for Live Search
type LiveSearchSource struct {
	Type              string   `json:"type"`                          // "web", "x", "news", "rss"
	Country           *string  `json:"country,omitempty"`             // ISO alpha-2 (web, news)
	ExcludedWebsites  []string `json:"excluded_websites,omitempty"`   // max 5 (web, news)
	AllowedWebsites   []string `json:"allowed_websites,omitempty"`    // max 5 (web only)
	SafeSearch        *bool    `json:"safe_search,omitempty"`         // default true (web, news)
	IncludedXHandles  []string `json:"included_x_handles,omitempty"`  // max 10 (x only)
	ExcludedXHandles  []string `json:"excluded_x_handles,omitempty"`  // max 10 (x only)
	PostFavoriteCount *int     `json:"post_favorite_count,omitempty"` // min favorites (x only)
	PostViewCount     *int     `json:"post_view_count,omitempty"`     // min views (x only)
	Links             []string `json:"links,omitempty"`               // RSS URLs, max 1 (rss only)
}

// DefaultLiveSearchOptions returns default Live Search options
func DefaultLiveSearchOptions() LiveSearchOptions {
	returnCitations := true
	return LiveSearchOptions{
		Mode:            "auto",
		ReturnCitations: &returnCitations,
		Sources: []LiveSearchSource{
			{Type: "web"},
			{Type: "x"},
		},
	}
}

// sendDeferred sends a deferred completion request to xAI
func (x *xaiClient) sendDeferred(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (string, error) {
	// Convert messages to the format expected by the API
	apiMessages := x.convertMessagesToAPI(messages)

	// Convert tools to the format expected by the API
	apiTools := x.convertToolsToAPI(tools)

	// Prepare request body
	reqBody := DeferredCompletionRequest{
		Model:     x.providerOptions.model.APIModel,
		Messages:  apiMessages,
		MaxTokens: &x.providerOptions.maxTokens,
		Deferred:  true,
		Tools:     apiTools,
	}

	// Apply reasoning effort if applicable
	if x.shouldApplyReasoningEffort() {
		reqBody.ReasoningEffort = x.options.reasoningEffort
	}

	// Apply response format if configured
	if x.options.responseFormat != nil {
		reqBody.ResponseFormat = x.options.responseFormat
	}

	// Apply tool choice if configured
	if x.options.toolChoice != nil {
		reqBody.ToolChoice = x.options.toolChoice
	}

	// Apply parallel tool calls if configured
	if x.options.parallelToolCalls != nil {
		reqBody.ParallelToolCalls = x.options.parallelToolCalls
	}

	// Apply Live Search parameters if enabled
	if x.liveSearchEnabled {
		reqBody.SearchParameters = &x.liveSearchOptions
	}

	// Marshal request body
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Get base URL (default to xAI API if not set)
	baseURL := "https://api.x.ai/v1"
	if x.openaiClient.options.baseURL != "" {
		baseURL = x.openaiClient.options.baseURL
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var deferredResp DeferredCompletionResponse
	if err := json.Unmarshal(body, &deferredResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if deferredResp.RequestID == "" {
		return "", fmt.Errorf("no request_id in response")
	}

	logging.Debug("Created deferred completion", "request_id", deferredResp.RequestID)

	return deferredResp.RequestID, nil
}

// pollDeferredResult polls for the deferred completion result
func (x *xaiClient) pollDeferredResult(ctx context.Context, requestID string, opts DeferredOptions) (*DeferredResult, error) {
	// Get base URL (default to xAI API if not set)
	baseURL := "https://api.x.ai/v1"
	if x.openaiClient.options.baseURL != "" {
		baseURL = x.openaiClient.options.baseURL
	}

	url := fmt.Sprintf("%s/chat/deferred-completion/%s", baseURL, requestID)

	// Create HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Start polling
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	timeout := time.After(opts.Timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for deferred completion after %v", opts.Timeout)
		case <-ticker.C:
			// Create request
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create polling request: %w", err)
			}

			// Set headers
			req.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

			// Send request
			resp, err := client.Do(req)
			if err != nil {
				logging.Debug("Error polling deferred result", "error", err)
				continue // Retry on network errors
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode == http.StatusAccepted {
				// 202 Accepted means still processing
				logging.Debug("Deferred completion still processing", "request_id", requestID)
				continue
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read polling response body: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("polling failed with status %d: %s", resp.StatusCode, string(body))
			}

			// Parse result
			var result DeferredResult
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("failed to parse deferred result: %w", err)
			}

			logging.Debug("Deferred completion ready", "request_id", requestID)

			return &result, nil
		}
	}
}

// SendDeferred sends a deferred completion request and polls for the result
func (x *xaiClient) SendDeferred(ctx context.Context, messages []message.Message, tools []tools.BaseTool, opts DeferredOptions) (*ProviderResponse, error) {
	// Send deferred request
	requestID, err := x.sendDeferred(ctx, messages, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to send deferred request: %w", err)
	}

	// Poll for result
	result, err := x.pollDeferredResult(ctx, requestID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get deferred result: %w", err)
	}

	// Convert result to ProviderResponse
	return x.convertDeferredResult(result), nil
}

// convertDeferredResult converts a DeferredResult to ProviderResponse
func (x *xaiClient) convertDeferredResult(result *DeferredResult) *ProviderResponse {
	if result == nil || len(result.Choices) == 0 {
		return &ProviderResponse{
			FinishReason: message.FinishReasonUnknown,
		}
	}

	choice := result.Choices[0]

	// Convert tool calls
	var toolCalls []message.ToolCall
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, message.ToolCall{
			ID:       tc.ID,
			Name:     tc.Function.Name,
			Input:    tc.Function.Arguments,
			Finished: true,
		})
	}

	// Determine finish reason
	finishReason := x.finishReason(choice.FinishReason)
	if len(toolCalls) > 0 {
		finishReason = message.FinishReasonToolUse
	}

	// Calculate cached tokens and actual input tokens
	var cachedTokens int64
	var inputTokens int64

	if result.Usage.PromptTokensDetails != nil {
		cachedTokens = result.Usage.PromptTokensDetails.CachedTokens
	}
	inputTokens = result.Usage.PromptTokens - cachedTokens

	// Handle content and reasoning_content separately to maintain proper data structure
	content := choice.Message.Content
	reasoningContent := choice.Message.ReasoningContent

	// Create response
	resp := &ProviderResponse{
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolCalls:        toolCalls,
		FinishReason:     finishReason,
		Usage: TokenUsage{
			InputTokens:         inputTokens,
			OutputTokens:        result.Usage.CompletionTokens,
			CacheCreationTokens: 0, // Not provided in deferred responses
			CacheReadTokens:     cachedTokens,
		},
		SystemFingerprint: result.SystemFingerprint,
		Citations:         result.Citations,
	}

	// Track fingerprint
	if resp.SystemFingerprint != "" {
		x.trackFingerprint(resp.SystemFingerprint, resp.Usage)
	}

	return resp
}

// convertMessagesToAPI converts internal messages to API format
func (x *xaiClient) convertMessagesToAPI(messages []message.Message) []map[string]interface{} {
	var apiMessages []map[string]interface{}

	// Add system message first
	apiMessages = append(apiMessages, map[string]interface{}{
		"role":    "system",
		"content": x.providerOptions.systemMessage,
	})

	for _, msg := range messages {
		apiMsg := map[string]interface{}{
			"role": string(msg.Role),
		}

		// Convert content based on message type
		switch msg.Role {
		case message.User, message.System:
			// Handle potential multipart content
			var content []map[string]interface{}
			textParts := []string{}

			for _, part := range msg.Parts {
				switch p := part.(type) {
				case message.TextContent:
					textParts = append(textParts, p.Text)
				case message.BinaryContent:
					// xAI expects images in a specific format
					content = append(content, map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": fmt.Sprintf("data:%s;base64,%s", p.MIMEType, p.Data),
						},
					})
				}
			}

			// If we have text parts, add them first
			if len(textParts) > 0 {
				content = append([]map[string]interface{}{{
					"type": "text",
					"text": strings.Join(textParts, "\n"),
				}}, content...)
			}

			if len(content) > 0 {
				apiMsg["content"] = content
			} else {
				apiMsg["content"] = msg.Content().String()
			}

		case message.Assistant:
			apiMsg["content"] = msg.Content().String()

			// Add tool calls if present
			if toolCalls := msg.ToolCalls(); len(toolCalls) > 0 {
				var apiToolCalls []map[string]interface{}
				for _, tc := range toolCalls {
					apiToolCalls = append(apiToolCalls, map[string]interface{}{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      tc.Name,
							"arguments": tc.Input,
						},
					})
				}
				apiMsg["tool_calls"] = apiToolCalls
			}

		case message.Tool:
			// Handle tool results
			for _, result := range msg.ToolResults() {
				apiMessages = append(apiMessages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": result.ToolCallID,
					"content":      result.Content,
				})
			}
			continue // Skip adding the message itself
		}

		apiMessages = append(apiMessages, apiMsg)
	}

	return apiMessages
}

// convertToolsToAPI converts internal tools to API format
func (x *xaiClient) convertToolsToAPI(tools []tools.BaseTool) []map[string]interface{} {
	var apiTools []map[string]interface{}

	// Filter tools based on provider compatibility
	providerName := string(x.providerOptions.model.Provider)
	filteredTools := FilterToolsByProvider(tools, providerName)

	for _, tool := range filteredTools {
		info := tool.Info()

		// Check if Parameters already contains the full schema (with "type" and "properties")
		var parameters map[string]interface{}
		params := info.Parameters
		if _, hasType := params["type"]; hasType {
			// Parameters already contains the full schema
			parameters = params
		} else {
			// Parameters only contains properties, wrap them
			parameters = map[string]interface{}{
				"type":       "object",
				"properties": info.Parameters,
				"required":   info.Required,
			}
		}

		apiTools = append(apiTools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        info.Name,
				"description": info.Description,
				"parameters":  parameters,
			},
		})
	}

	return apiTools
}

// sanitizeContent removes control characters that could corrupt terminal display
func sanitizeContent(content string) string {
	// Remove ANSI escape sequences (ESC character)
	content = strings.ReplaceAll(content, "\x1b", "")
	// Remove carriage returns (which can cause display issues)
	content = strings.ReplaceAll(content, "\r", "")
	// Remove other control characters that might cause issues
	content = strings.ReplaceAll(content, "\x00", "") // null
	content = strings.ReplaceAll(content, "\x07", "") // bell
	content = strings.ReplaceAll(content, "\x08", "") // backspace
	// Replace form feed with newline to preserve structure
	content = strings.ReplaceAll(content, "\x0c", "\n")
	return content
}
