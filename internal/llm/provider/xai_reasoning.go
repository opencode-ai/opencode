package provider

import (
	"context"
	"strings"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
)

// ReasoningConfig holds configuration for reasoning requests
type ReasoningConfig struct {
	Model           string
	ReasoningEffort string
	MaxTokens       int
	Tools           []tools.BaseTool
}

// ReasoningHandler manages XAI reasoning content processing
type ReasoningHandler struct {
	client *xaiClient
}

// NewReasoningHandler creates a new reasoning handler
func NewReasoningHandler(client *xaiClient) *ReasoningHandler {
	return &ReasoningHandler{
		client: client,
	}
}

// ShouldUseReasoning determines if reasoning should be used for a request.
// It checks if the model supports reasoning and handles special cases like Grok-4.
func (r *ReasoningHandler) ShouldUseReasoning() bool {
	model := r.client.providerOptions.model

	// Early return if model doesn't support reasoning
	if !model.CanReason {
		return false
	}

	// Special case: Grok-4 always uses reasoning handler when it can reason
	// even though it doesn't accept reasoning_effort parameter
	if model.ID == models.XAIGrok4 {
		return true
	}

	// For other models, check if reasoning effort is configured
	reasoningEffort := r.client.options.reasoningEffort
	if reasoningEffort == "" {
		return false
	}

	// Check if reasoning should be applied based on client-specific logic
	shouldApply := r.client.shouldApplyReasoningEffort()

	logging.Debug("Reasoning conditions evaluated",
		"model", model.ID,
		"can_reason", model.CanReason,
		"reasoning_effort", reasoningEffort,
		"should_apply", shouldApply)

	return shouldApply
}

// normalizeReasoningEffort adjusts reasoning effort values based on model capabilities.
// xAI's thinking models (all except Grok-4) only support "low" or "high", not "medium".
// Grok-4 has internal reasoning but doesn't accept the reasoning_effort parameter or expose reasoning content.
func (r *ReasoningHandler) normalizeReasoningEffort(effort string) string {
	model := r.client.providerOptions.model

	// All xAI thinking models except Grok-4 only support "low" or "high"
	// Grok-4 has internal reasoning but doesn't accept this parameter
	if model.ID != models.XAIGrok4 && effort == "medium" {
		logging.Debug("Normalizing reasoning effort for xAI thinking model",
			"model", model.ID,
			"original", effort,
			"normalized", "high")
		return "high"
	}

	return effort
}

// ProcessReasoningResponse handles reasoning content from API responses
func (r *ReasoningHandler) ProcessReasoningResponse(response *ProviderResponse) []ProviderEvent {
	var events []ProviderEvent

	// Send reasoning content as thinking delta first (if present)
	if response.ReasoningContent != "" {
		sanitizedReasoning := r.sanitizeReasoningContent(response.ReasoningContent)
		events = append(events, ProviderEvent{
			Type:     EventThinkingDelta,
			Thinking: sanitizedReasoning,
		})

		logging.Debug("Reasoning content processed",
			"original_length", len(response.ReasoningContent),
			"sanitized_length", len(sanitizedReasoning))
	}

	// Send regular content as delta if present
	if response.Content != "" {
		events = append(events, ProviderEvent{
			Type:    EventContentDelta,
			Content: response.Content,
		})
	}

	// Clear reasoning content from response before sending complete event
	response.ReasoningContent = ""

	// Send complete event
	events = append(events, ProviderEvent{
		Type:     EventComplete,
		Response: response,
	})

	return events
}

// sanitizeReasoningContent removes control characters that could corrupt terminal display
func (r *ReasoningHandler) sanitizeReasoningContent(content string) string {
	// Define control characters to remove
	replacements := []struct {
		old string
		new string
	}{
		{"\x1b", ""},   // ANSI escape sequences (ESC character)
		{"\r", ""},     // carriage returns (can cause display issues)
		{"\x00", ""},   // null
		{"\x07", ""},   // bell
		{"\x08", ""},   // backspace
		{"\x0c", "\n"}, // form feed - replace with newline to preserve structure
	}

	// Apply all replacements
	for _, repl := range replacements {
		content = strings.ReplaceAll(content, repl.old, repl.new)
	}

	return content
}

// BuildReasoningRequest creates a request body for reasoning models
func (r *ReasoningHandler) BuildReasoningRequest(ctx context.Context, messages []message.Message, tools []tools.BaseTool) map[string]interface{} {
	reqBody := map[string]interface{}{
		"model":      r.client.providerOptions.model.APIModel,
		"messages":   r.client.convertMessagesToAPI(messages),
		"max_tokens": r.client.providerOptions.maxTokens,
		"stream":     false, // Explicitly disable streaming for reasoning requests
	}

	// Add tools if provided
	if len(tools) > 0 {
		reqBody["tools"] = r.client.convertToolsToAPI(tools)
	}

	// Apply reasoning effort parameter for models that support it
	// Grok-4 has internal reasoning but doesn't accept this parameter
	if r.client.options.reasoningEffort != "" && r.client.shouldApplyReasoningEffort() {
		reasoningEffort := r.normalizeReasoningEffort(r.client.options.reasoningEffort)
		reqBody["reasoning_effort"] = reasoningEffort
	}

	// Apply response format if configured
	if r.client.options.responseFormat != nil {
		reqBody["response_format"] = r.client.options.responseFormat
	}

	// Apply tool choice if configured
	if r.client.options.toolChoice != nil {
		reqBody["tool_choice"] = r.client.options.toolChoice
	}

	// Apply parallel tool calls if configured
	if r.client.options.parallelToolCalls != nil {
		reqBody["parallel_tool_calls"] = r.client.options.parallelToolCalls
	}

	return reqBody
}
