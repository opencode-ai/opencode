package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
)

type deepSeekOptions struct {
	baseURL      string
	extraHeaders map[string]string
}

type DeepSeekOption func(*deepSeekOptions)

type deepSeekClient struct {
	providerOptions providerClientOptions
	options         deepSeekOptions
	client          openai.Client
}

type DeepSeekClient ProviderClient

func newDeepSeekClient(opts providerClientOptions) DeepSeekClient {
	deepSeekOpts := deepSeekOptions{
		baseURL: "https://api.deepseek.com/v1",
	}
	for _, o := range opts.deepSeekOptions {
		o(&deepSeekOpts)
	}

	clientOptions := []option.RequestOption{}
	if opts.apiKey != "" {
		clientOptions = append(clientOptions, option.WithAPIKey(opts.apiKey))
	}
	if deepSeekOpts.baseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(deepSeekOpts.baseURL))
	}

	if deepSeekOpts.extraHeaders != nil {
		for key, value := range deepSeekOpts.extraHeaders {
			clientOptions = append(clientOptions, option.WithHeader(key, value))
		}
	}

	client := openai.NewClient(clientOptions...)
	return &deepSeekClient{
		providerOptions: opts,
		options:         deepSeekOpts,
		client:          client,
	}
}

func (d *deepSeekClient) convertMessages(messages []message.Message) (deepSeekMessages []openai.ChatCompletionMessageParamUnion) {
	// Add system message first
	deepSeekMessages = append(deepSeekMessages, openai.SystemMessage(d.providerOptions.systemMessage))

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})
			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String(models.ProviderDeepSeek)}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}
			deepSeekMessages = append(deepSeekMessages, openai.UserMessage(content))

		case message.Assistant:
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

			deepSeekMessages = append(deepSeekMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

		case message.Tool:
			for _, result := range msg.ToolResults() {
				deepSeekMessages = append(deepSeekMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

// DeepSeek-specific tool conversion that handles empty tools properly
func (d *deepSeekClient) convertTools(tools []tools.BaseTool) []openai.ChatCompletionToolParam {
	// DeepSeek API doesn't accept empty tools array - return nil instead
	if len(tools) == 0 {
		return nil
	}

	deepSeekTools := make([]openai.ChatCompletionToolParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		deepSeekTools[i] = openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": info.Parameters,
					"required":   info.Required,
				},
			},
		}
	}

	return deepSeekTools
}

func (d *deepSeekClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "stop":
		return message.FinishReasonEndTurn
	case "length":
		return message.FinishReasonMaxTokens
	case "tool_calls":
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (d *deepSeekClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:     openai.ChatModel(d.providerOptions.model.APIModel),
		Messages:  messages,
		MaxTokens: openai.Int(d.providerOptions.maxTokens),
	}

	// Only add tools if they exist (DeepSeek doesn't like empty tools array)
	if len(tools) > 0 {
		params.Tools = tools
	}

	return params
}

func (d *deepSeekClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (response *ProviderResponse, err error) {
	params := d.preparedParams(d.convertMessages(messages), d.convertTools(tools))
	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("DeepSeek prepared messages", "messages", string(jsonData))
	}

	attempts := 0
	for {
		attempts++
		deepSeekResponse, err := d.client.Chat.Completions.New(ctx, params)
		
		// If there is an error we are going to see if we can retry the call
		if err != nil {
			retry, after, retryErr := d.shouldRetry(attempts, err)
			if retryErr != nil {
				return nil, retryErr
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("DeepSeek: Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			return nil, retryErr
		}

		content := ""
		if deepSeekResponse.Choices[0].Message.Content != "" {
			content = deepSeekResponse.Choices[0].Message.Content
		}

		toolCalls := d.toolCalls(*deepSeekResponse)
		finishReason := d.finishReason(string(deepSeekResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        d.usage(*deepSeekResponse),
			FinishReason: finishReason,
		}, nil
	}
}

func (d *deepSeekClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	params := d.preparedParams(d.convertMessages(messages), d.convertTools(tools))
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("DeepSeek prepared messages", "messages", string(jsonData))
	}

	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		for {
			attempts++
			deepSeekStream := d.client.Chat.Completions.NewStreaming(ctx, params)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)

			for deepSeekStream.Next() {
				chunk := deepSeekStream.Current()
				acc.AddChunk(chunk)

				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						eventChan <- ProviderEvent{
							Type:    EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}
			}

			err := deepSeekStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				// Stream completed successfully
				finishReason := d.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, d.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        d.usage(acc.ChatCompletion),
						FinishReason: finishReason,
					},
				}
				close(eventChan)
				return
			}

			// If there is an error we are going to see if we can retry the call
			retry, after, retryErr := d.shouldRetry(attempts, err)
			if retryErr != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
				close(eventChan)
				return
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("DeepSeek: Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					if ctx.Err() != nil {
						eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					}
					close(eventChan)
					return
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
			close(eventChan)
			return
		}
	}()

	return eventChan
}

func (d *deepSeekClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	var apierr *openai.Error
	if !errors.As(err, &apierr) {
		return false, 0, err
	}

	// DeepSeek specific retry logic
	if apierr.StatusCode != http.StatusTooManyRequests && apierr.StatusCode != http.StatusInternalServerError {
		return false, 0, err
	}

	if attempts > maxRetries {
		return false, 0, fmt.Errorf("DeepSeek: maximum retry attempts reached: %d retries", maxRetries)
	}

	retryMs := 0
	retryAfterValues := apierr.Response.Header.Values("Retry-After")

	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs = backoffMs + jitterMs
	if len(retryAfterValues) > 0 {
		if _, err := fmt.Sscanf(retryAfterValues[0], "%d", &retryMs); err == nil {
			retryMs = retryMs * 1000
		}
	}
	return true, int64(retryMs), nil
}

func (d *deepSeekClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(completion.Choices) > 0 && len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCall := message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (d *deepSeekClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := int64(0)
	if completion.Usage.PromptTokensDetails.CachedTokens != 0 {
		cachedTokens = int64(completion.Usage.PromptTokensDetails.CachedTokens)
	}
	inputTokens := int64(completion.Usage.PromptTokens) - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        int64(completion.Usage.CompletionTokens),
		CacheCreationTokens: 0,
		CacheReadTokens:     cachedTokens,
	}
}

func WithDeepSeekBaseURL(baseURL string) DeepSeekOption {
	return func(options *deepSeekOptions) {
		options.baseURL = baseURL
	}
}

func WithDeepSeekExtraHeaders(headers map[string]string) DeepSeekOption {
	return func(options *deepSeekOptions) {
		options.extraHeaders = headers
	}
}