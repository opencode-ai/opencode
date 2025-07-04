package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	toolsPkg "github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/spf13/viper"
)

type copilotOptions struct {
	reasoningEffort string
	extraHeaders    map[string]string
	bearerToken     string
}

type CopilotOption func(*copilotOptions)

type copilotClient struct {
	providerOptions providerClientOptions
	options         copilotOptions
	client          openai.Client
	httpClient      *http.Client
}

type CopilotClient ProviderClient

// CopilotTokenResponse represents the response from GitHub's token exchange endpoint
type CopilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func (c *copilotClient) isAnthropicModel() bool {
	for _, modelId := range models.CopilotAnthropicModels {
		if c.providerOptions.model.ID == modelId {
			return true
		}
	}
	return false
}

// GitHub OAuth device flow response
type GitHubDeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// GitHub OAuth token response
type GitHubTokenResponse struct {
	// Standard OAuth fields 
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	
	// For backward compatibility with any custom formats
	Token string `json:"token,omitempty"`
}

// performDeviceCodeFlow initiates the GitHub device code flow and returns a GitHub token
func (c *copilotClient) performDeviceCodeFlow() (string, error) {
	// Step 1: Get a device code
	data := url.Values{}
	
	// Use the official GitHub Copilot client ID
	const copilotClientID = "Iv1.b507a08c87ecfe98"
	data.Set("client_id", copilotClientID)
	data.Set("scope", "user:email read:user copilot")
	
	// Using the exact URL and headers from VS Code Copilot extension
	req, err := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create device code request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "OpenCode/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("device code request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp GitHubDeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return "", fmt.Errorf("failed to parse device code response: %w", err)
	}

	// Step 2: Print instructions for the user
	fmt.Printf("\n🔑 GitHub Copilot Authentication Required\n\n")
	fmt.Printf("1. Visit: %s\n", deviceResp.VerificationURI)
	fmt.Printf("2. Enter code: %s\n\n", deviceResp.UserCode)
	fmt.Printf("Waiting for authentication... (expires in %d seconds)\n", deviceResp.ExpiresIn)
	fmt.Printf("Please complete authentication in your browser to continue.\n\n")

	// Step 3: Poll for the token
	tokenData := url.Values{}
	tokenData.Set("client_id", copilotClientID)
	tokenData.Set("device_code", deviceResp.DeviceCode)
	tokenData.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	// Add a slight delay before first poll
	time.Sleep(2 * time.Second)

	// Create a context with timeout based on expires_in
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deviceResp.ExpiresIn)*time.Second)
	defer cancel()

	interval := deviceResp.Interval
	if interval < 5 {
		interval = 5 // Ensure minimum polling interval
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	fmt.Printf("Waiting for authorization...\n")
	maxPolls := 60 // Maximum polling attempts
	pollCount := 0
	
	for {
		select {
		case <-ticker.C:
			pollCount++
			if pollCount > maxPolls {
				return "", fmt.Errorf("maximum polling attempts reached, please try again")
			}
			
			// Make a request to check if the user has authorized
			tokenReq, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", 
				strings.NewReader(tokenData.Encode()))
			if err != nil {
				return "", fmt.Errorf("failed to create token request: %w", err)
			}

			tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			tokenReq.Header.Set("User-Agent", "OpenCode/1.0")
			tokenReq.Header.Set("Accept", "application/json")

			tokenResp, err := c.httpClient.Do(tokenReq)
			if err != nil {
				return "", fmt.Errorf("failed token request: %w", err)
			}
			defer tokenResp.Body.Close()

			// Read the full response body so we can log it and also re-analyze it
			bodyBytes, err := io.ReadAll(tokenResp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read token response body: %w", err)
			}
			
			if tokenResp.StatusCode == http.StatusOK {
				// Log the raw response for debugging
				logging.Debug("Token response body", "body", string(bodyBytes))
				
				// Check for empty or invalid responses
				if len(bodyBytes) == 0 {
					logging.Debug("Empty response body from GitHub")
					continue // Continue polling
				}
				
				// First try standard OAuth response format
				var tokenResponse GitHubTokenResponse
				if err := json.Unmarshal(bodyBytes, &tokenResponse); err != nil {
					logging.Debug("Failed to parse as standard OAuth format", "error", err)
					
					// Try alternative format with access_token field
					var altResponse struct {
						AccessToken string `json:"access_token"`
						TokenType   string `json:"token_type"`
						Scope       string `json:"scope"`
					}
					
					if err := json.Unmarshal(bodyBytes, &altResponse); err != nil {
						logging.Debug("Failed to parse in any format", "error", err)
						continue // Continue polling instead of failing
					}
					
					// Use the alternative format
					tokenResponse.AccessToken = altResponse.AccessToken
				}

				// Check which token field was populated
				var finalToken string
				if tokenResponse.AccessToken != "" {
					finalToken = tokenResponse.AccessToken
				} else if tokenResponse.Token != "" {
					finalToken = tokenResponse.Token
				}
				
				// Final token validation
				if finalToken == "" {
					logging.Debug("No token found in response")
					continue // Continue polling instead of failing
				}
				
				if finalToken != "" {
					fmt.Printf("Successfully authenticated with GitHub!\n")
					
					// Save token to standard locations
					saveGitHubToken(finalToken)
					
					return finalToken, nil
				} else {
					// If we got a 200 but no access token, that's strange
					logging.Error("Received HTTP 200 but no access token in response", "body", string(bodyBytes))
					return "", fmt.Errorf("authentication response did not contain an access token")
				}
			} else if tokenResp.StatusCode == http.StatusBadRequest {
				// Check for specific errors
				var errorResp map[string]string
				if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
					if errorResp["error"] == "authorization_pending" {
						// Still waiting for user to authorize - continue
						continue
					} else if errorResp["error"] == "slow_down" {
						// Rate limiting - fail fast
						return "", fmt.Errorf("GitHub rate limit detected, please try again in a few minutes")
					} else if errorResp["error"] == "expired_token" {
						return "", fmt.Errorf("device code expired, please try again")
					}
				}
			}
			
			// Any other error
			return "", fmt.Errorf("token request failed with status %d: %s", 
				tokenResp.StatusCode, string(bodyBytes))

		case <-ctx.Done():
			return "", fmt.Errorf("authentication timed out after %d seconds", deviceResp.ExpiresIn)
		}
	}
}

// saveGitHubToken saves the GitHub token to the standard location for future use
func saveGitHubToken(token string) {
	// Only save if we have a token
	if token == "" {
		logging.Error("Cannot save empty GitHub token")
		return
	}
	
	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME") // Fallback to HOME environment variable
		if homeDir == "" {
			logging.Error("Failed to determine home directory")
			return
		}
	}
	
	// Set config directory based on platform
	var configDir string
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		configDir = xdgConfig
	} else if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			configDir = localAppData
		} else {
			configDir = filepath.Join(homeDir, "AppData", "Local")
		}
	} else {
		configDir = filepath.Join(homeDir, ".config")
	}

	// Create the directory if it doesn't exist
	copilotDir := filepath.Join(configDir, "github-copilot")
	if err := os.MkdirAll(copilotDir, 0755); err != nil {
		logging.Error("Failed to create github-copilot directory", "error", err)
		return
	}
	
	// Save to both files for maximum compatibility
	
	// 1. Save to hosts.json (VS Code format)
	hostsFile := filepath.Join(copilotDir, "hosts.json")
	hostsData := map[string]map[string]interface{}{
		"github.com": {
			"oauth_token": token,
		},
	}
	hostsJSON, err := json.Marshal(hostsData)
	if err == nil {
		if err := os.WriteFile(hostsFile, hostsJSON, 0600); err != nil {
			logging.Error("Failed to write hosts.json", "error", err)
		}
	}
	
	// Set environment variables for immediate use
	os.Setenv("GITHUB_TOKEN", token)
	os.Setenv("GITHUB_COPILOT_TOKEN", token)
}

// exchangeGitHubToken exchanges a GitHub token for a Copilot bearer token
func (c *copilotClient) exchangeGitHubToken(githubToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/copilot_internal/v2/token", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token exchange request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+githubToken)
	req.Header.Set("User-Agent", "OpenCode/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange GitHub token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp CopilotTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.Token, nil
}

func newCopilotClient(opts providerClientOptions) CopilotClient {
	copilotOpts := copilotOptions{
		reasoningEffort: "medium",
	}
	// Apply copilot-specific options
	for _, o := range opts.copilotOptions {
		o(&copilotOpts)
	}

	// Create HTTP client for token exchange
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	var bearerToken string

	// If bearer token is already provided, use it
	if copilotOpts.bearerToken != "" {
		bearerToken = copilotOpts.bearerToken
	} else {
		// Try to get GitHub token from multiple sources
		var githubToken string

		// 1. Check environment variables first (fastest)
		for _, envVar := range []string{"GITHUB_TOKEN", "GITHUB_COPILOT_TOKEN", "GH_COPILOT_TOKEN"} {
			if token := os.Getenv(envVar); token != "" {
				githubToken = token
				break
			}
		}

		// 2. API key from options
		if githubToken == "" && opts.apiKey != "" {
			githubToken = opts.apiKey
		}

		// 3. Standard GitHub CLI/Copilot locations
		if githubToken == "" {
			var err error
			fmt.Printf("🔍 Looking for GitHub Copilot token in standard locations...\n")
			githubToken, err = config.LoadGitHubToken()
			if err != nil {
				// Check if we need to use device flow
				if err.Error() == "no_copilot_token" && !viper.GetBool("non_interactive") && viper.GetString("prompt") == "" {
					logging.Info("No GitHub token found, starting device code flow")
					fmt.Printf("🔑 No GitHub token found, starting authentication flow...\n")
					
					// Create temporary client for auth flow
					tempClient := &copilotClient{
						providerOptions: opts,
						options:         copilotOpts,
						httpClient:      httpClient,
					}
					
					var authErr error
					githubToken, authErr = tempClient.performDeviceCodeFlow()
					if authErr != nil {
						logging.Error("Device code authentication failed", "error", authErr)
						fmt.Printf("❌ Authentication failed: %v\n", authErr)
						return &copilotClient{
							providerOptions: opts,
							options:         copilotOpts,
							httpClient:      httpClient,
						}
					}
					
					// Double-check token after device flow
					if githubToken != "" {
						fmt.Printf("✅ Successfully obtained GitHub token\n")
						// Set it directly in opts.apiKey
						opts.apiKey = githubToken
					}
				} else {
					logging.Debug("Failed to load GitHub token from standard locations", "error", err)
					fmt.Printf("⚠️ Failed to load GitHub token: %v\n", err)
				}
			} else if githubToken != "" {
				fmt.Printf("✅ Found existing GitHub token (length: %d)\n", len(githubToken))
			}
		}

		if githubToken == "" {
			logging.Error("GitHub token is required for Copilot provider. Set GITHUB_TOKEN environment variable, configure it in opencode.json, or ensure GitHub CLI/Copilot is properly authenticated.")
			return &copilotClient{
				providerOptions: opts,
				options:         copilotOpts,
				httpClient:      httpClient,
			}
		}

		// Create a temporary client for token exchange
		tempClient := &copilotClient{
			providerOptions: opts,
			options:         copilotOpts,
			httpClient:      httpClient,
		}

		// Exchange GitHub token for bearer token
		var err error
		fmt.Printf("🔄 Exchanging GitHub token for Copilot bearer token...\n")
		bearerToken, err = tempClient.exchangeGitHubToken(githubToken)
		if err != nil {
			logging.Error("Failed to exchange GitHub token for Copilot bearer token", "error", err)
			fmt.Printf("❌ Failed to exchange GitHub token: %v\n", err)
			return &copilotClient{
				providerOptions: opts,
				options:         copilotOpts,
				httpClient:      httpClient,
			}
		} else if bearerToken != "" {
			fmt.Printf("✅ Successfully obtained Copilot bearer token\n")
		}
	}

	copilotOpts.bearerToken = bearerToken

	// GitHub Copilot API base URL
	baseURL := "https://api.githubcopilot.com"

	openaiClientOptions := []option.RequestOption{
		option.WithBaseURL(baseURL),
		option.WithAPIKey(bearerToken), // Use bearer token as API key
	}

	// Add GitHub Copilot specific headers
	openaiClientOptions = append(openaiClientOptions,
		option.WithHeader("Editor-Version", "OpenCode/1.0"),
		option.WithHeader("Editor-Plugin-Version", "OpenCode/1.0"),
		option.WithHeader("Copilot-Integration-Id", "vscode-chat"),
	)

	// Add any extra headers
	if copilotOpts.extraHeaders != nil {
		for key, value := range copilotOpts.extraHeaders {
			openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
		}
	}

	client := openai.NewClient(openaiClientOptions...)
	// logging.Debug("Copilot client created", "opts", opts, "copilotOpts", copilotOpts, "model", opts.model)
	return &copilotClient{
		providerOptions: opts,
		options:         copilotOpts,
		client:          client,
		httpClient:      httpClient,
	}
}

func (c *copilotClient) convertMessages(messages []message.Message) (copilotMessages []openai.ChatCompletionMessageParamUnion) {
	// Add system message first
	copilotMessages = append(copilotMessages, openai.SystemMessage(c.providerOptions.systemMessage))

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})

			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String(models.ProviderCopilot)}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}

			copilotMessages = append(copilotMessages, openai.UserMessage(content))

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

			copilotMessages = append(copilotMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

		case message.Tool:
			for _, result := range msg.ToolResults() {
				copilotMessages = append(copilotMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

func (c *copilotClient) convertTools(tools []toolsPkg.BaseTool) []openai.ChatCompletionToolParam {
	copilotTools := make([]openai.ChatCompletionToolParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		copilotTools[i] = openai.ChatCompletionToolParam{
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

	return copilotTools
}

func (c *copilotClient) finishReason(reason string) message.FinishReason {
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

func (c *copilotClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.providerOptions.model.APIModel),
		Messages: messages,
		Tools:    tools,
	}

	if c.providerOptions.model.CanReason == true {
		params.MaxCompletionTokens = openai.Int(c.providerOptions.maxTokens)
		switch c.options.reasoningEffort {
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "medium":
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		}
	} else {
		params.MaxTokens = openai.Int(c.providerOptions.maxTokens)
	}

	return params
}

func (c *copilotClient) send(ctx context.Context, messages []message.Message, tools []toolsPkg.BaseTool) (response *ProviderResponse, err error) {
	params := c.preparedParams(c.convertMessages(messages), c.convertTools(tools))
	cfg := config.Get()
	var sessionId string
	requestSeqId := (len(messages) + 1) / 2
	if cfg.Debug {
		// jsonData, _ := json.Marshal(params)
		// logging.Debug("Prepared messages", "messages", string(jsonData))
		if sid, ok := ctx.Value(toolsPkg.SessionIDContextKey).(string); ok {
			sessionId = sid
		}
		jsonData, _ := json.Marshal(params)
		if sessionId != "" {
			filepath := logging.WriteRequestMessageJson(sessionId, requestSeqId, params)
			logging.Debug("Prepared messages", "filepath", filepath)
		} else {
			logging.Debug("Prepared messages", "messages", string(jsonData))
		}
	}

	attempts := 0
	for {
		attempts++
		copilotResponse, err := c.client.Chat.Completions.New(
			ctx,
			params,
		)

		// If there is an error we are going to see if we can retry the call
		if err != nil {
			retry, after, retryErr := c.shouldRetry(attempts, err)
			if retryErr != nil {
				return nil, retryErr
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
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
		if copilotResponse.Choices[0].Message.Content != "" {
			content = copilotResponse.Choices[0].Message.Content
		}

		toolCalls := c.toolCalls(*copilotResponse)
		finishReason := c.finishReason(string(copilotResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        c.usage(*copilotResponse),
			FinishReason: finishReason,
		}, nil
	}
}

func (c *copilotClient) stream(ctx context.Context, messages []message.Message, tools []toolsPkg.BaseTool) <-chan ProviderEvent {
	params := c.preparedParams(c.convertMessages(messages), c.convertTools(tools))
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	cfg := config.Get()
	var sessionId string
	requestSeqId := (len(messages) + 1) / 2
	if cfg.Debug {
		if sid, ok := ctx.Value(toolsPkg.SessionIDContextKey).(string); ok {
			sessionId = sid
		}
		jsonData, _ := json.Marshal(params)
		if sessionId != "" {
			filepath := logging.WriteRequestMessageJson(sessionId, requestSeqId, params)
			logging.Debug("Prepared messages", "filepath", filepath)
		} else {
			logging.Debug("Prepared messages", "messages", string(jsonData))
		}

	}

	attempts := 0
	eventChan := make(chan ProviderEvent)

	go func() {
		for {
			attempts++
			copilotStream := c.client.Chat.Completions.NewStreaming(
				ctx,
				params,
			)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)

			var currentToolCallId string
			var currentToolCall openai.ChatCompletionMessageToolCall
			var msgToolCalls []openai.ChatCompletionMessageToolCall
			for copilotStream.Next() {
				chunk := copilotStream.Current()
				acc.AddChunk(chunk)

				if cfg.Debug {
					logging.AppendToStreamSessionLogJson(sessionId, requestSeqId, chunk)
				}

				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						eventChan <- ProviderEvent{
							Type:    EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}

				if c.isAnthropicModel() {
					// Monkeypatch adapter for Sonnet-4 multi-tool use
					for _, choice := range chunk.Choices {
						if choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0 {
							toolCall := choice.Delta.ToolCalls[0]
							// Detect tool use start
							if currentToolCallId == "" {
								if toolCall.ID != "" {
									currentToolCallId = toolCall.ID
									currentToolCall = openai.ChatCompletionMessageToolCall{
										ID:   toolCall.ID,
										Type: "function",
										Function: openai.ChatCompletionMessageToolCallFunction{
											Name:      toolCall.Function.Name,
											Arguments: toolCall.Function.Arguments,
										},
									}
								}
							} else {
								// Delta tool use
								if toolCall.ID == "" {
									currentToolCall.Function.Arguments += toolCall.Function.Arguments
								} else {
									// Detect new tool use
									if toolCall.ID != currentToolCallId {
										msgToolCalls = append(msgToolCalls, currentToolCall)
										currentToolCallId = toolCall.ID
										currentToolCall = openai.ChatCompletionMessageToolCall{
											ID:   toolCall.ID,
											Type: "function",
											Function: openai.ChatCompletionMessageToolCallFunction{
												Name:      toolCall.Function.Name,
												Arguments: toolCall.Function.Arguments,
											},
										}
									}
								}
							}
						}
						if choice.FinishReason == "tool_calls" {
							msgToolCalls = append(msgToolCalls, currentToolCall)
							acc.ChatCompletion.Choices[0].Message.ToolCalls = msgToolCalls
						}
					}
				}
			}

			err := copilotStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				if cfg.Debug {
					respFilepath := logging.WriteChatResponseJson(sessionId, requestSeqId, acc.ChatCompletion)
					logging.Debug("Chat completion response", "filepath", respFilepath)
				}
				// Stream completed successfully
				finishReason := c.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, c.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        c.usage(acc.ChatCompletion),
						FinishReason: finishReason,
					},
				}
				close(eventChan)
				return
			}

			// If there is an error we are going to see if we can retry the call
			retry, after, retryErr := c.shouldRetry(attempts, err)
			if retryErr != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
				close(eventChan)
				return
			}
			// shouldRetry is not catching the max retries...
			// TODO: Figure out why
			if attempts > maxRetries {
				logging.Warn("Maximum retry attempts reached for rate limit", "attempts", attempts, "max_retries", maxRetries)
				retry = false
			}
			if retry {
				logging.WarnPersist(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d (paused for %d ms)", attempts, maxRetries, after), logging.PersistTimeArg, time.Millisecond*time.Duration(after+100))
				select {
				case <-ctx.Done():
					// context cancelled
					if ctx.Err() == nil {
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

func (c *copilotClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	var apierr *openai.Error
	if !errors.As(err, &apierr) {
		return false, 0, err
	}

	// Check for token expiration (401 Unauthorized)
	if apierr.StatusCode == 401 {
		// Try to refresh the bearer token
		var githubToken string

		// 1. Environment variable
		githubToken = os.Getenv("GITHUB_TOKEN")

		// 2. API key from options
		if githubToken == "" {
			githubToken = c.providerOptions.apiKey
		}

		// 3. Standard GitHub CLI/Copilot locations
		if githubToken == "" {
			var err error
			githubToken, err = config.LoadGitHubToken()
			if err != nil {
				logging.Debug("Failed to load GitHub token from standard locations during retry", "error", err)
			}
		}

		if githubToken != "" {
			newBearerToken, tokenErr := c.exchangeGitHubToken(githubToken)
			if tokenErr == nil {
				c.options.bearerToken = newBearerToken
				// Update the client with the new token
				// Note: This is a simplified approach. In a production system,
				// you might want to recreate the entire client with the new token
				logging.Info("Refreshed Copilot bearer token")
				return true, 1000, nil // Retry immediately with new token
			}
			logging.Error("Failed to refresh Copilot bearer token", "error", tokenErr)
		}
		return false, 0, fmt.Errorf("authentication failed: %w", err)
	}
	logging.Debug("Copilot API Error", "status", apierr.StatusCode, "headers", apierr.Response.Header, "body", apierr.RawJSON())

	if apierr.StatusCode != 429 && apierr.StatusCode != 500 {
		return false, 0, err
	}

	if apierr.StatusCode == 500 {
		logging.Warn("Copilot API returned 500 error, retrying", "error", err)
	}

	if attempts > maxRetries {
		return false, 0, fmt.Errorf("maximum retry attempts reached for rate limit: %d retries", maxRetries)
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

func (c *copilotClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
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

func (c *copilotClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0, // GitHub Copilot doesn't provide this directly
		CacheReadTokens:     cachedTokens,
	}
}

func WithCopilotReasoningEffort(effort string) CopilotOption {
	return func(options *copilotOptions) {
		defaultReasoningEffort := "medium"
		switch effort {
		case "low", "medium", "high":
			defaultReasoningEffort = effort
		default:
			logging.Warn("Invalid reasoning effort, using default: medium")
		}
		options.reasoningEffort = defaultReasoningEffort
	}
}

func WithCopilotExtraHeaders(headers map[string]string) CopilotOption {
	return func(options *copilotOptions) {
		options.extraHeaders = headers
	}
}

func WithCopilotBearerToken(bearerToken string) CopilotOption {
	return func(options *copilotOptions) {
		options.bearerToken = bearerToken
	}
}

