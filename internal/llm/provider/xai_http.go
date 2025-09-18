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

	"github.com/opencode-ai/opencode/internal/logging"
)

// HTTPClientConfig holds configuration for HTTP requests
type HTTPClientConfig struct {
	BaseURL   string
	APIKey    string
	Timeout   time.Duration
	UserAgent string
}

// XAIHTTPClient handles HTTP communication with xAI API
type XAIHTTPClient struct {
	config HTTPClientConfig
	client *http.Client
}

// NewXAIHTTPClient creates a new XAI HTTP client
func NewXAIHTTPClient(config HTTPClientConfig) *XAIHTTPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.x.ai/v1"
	}

	// Ensure HTTPS
	if strings.HasPrefix(config.BaseURL, "http://") {
		config.BaseURL = strings.Replace(config.BaseURL, "http://", "https://", 1)
		logging.Debug("Converted HTTP to HTTPS", "url", config.BaseURL)
	}

	return &XAIHTTPClient{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// SendCompletionRequest sends a chat completion request to xAI API
func (c *XAIHTTPClient) SendCompletionRequest(ctx context.Context, reqBody map[string]interface{}) (*DeferredResult, error) {
	// Marshal request body
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	url := c.config.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setRequestHeaders(req, len(jsonBody))

	// Log request details (with masked API key)
	c.logRequest(url, len(jsonBody))

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		logging.Error("HTTP request failed",
			"status", resp.StatusCode,
			"body", string(body))
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	logging.Debug("HTTP response received", "status", resp.StatusCode, "body_size", len(body))

	// Parse response
	var result DeferredResult
	if err := json.Unmarshal(body, &result); err != nil {
		logging.Error("Failed to parse response",
			"error", err,
			"body", string(body))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Log the parsed result
	c.logResponse(&result)

	return &result, nil
}

// setRequestHeaders sets standard headers for xAI API requests
func (c *XAIHTTPClient) setRequestHeaders(req *http.Request, bodySize int) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	if c.config.UserAgent != "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}
}

// logRequest logs request details with masked API key
func (c *XAIHTTPClient) logRequest(url string, bodySize int) {
	maskedKey := c.getMaskedAPIKey()
	logging.Debug("Sending HTTP request",
		"url", url,
		"body_size", bodySize,
		"api_key_masked", maskedKey)
}

// logResponse logs response details
func (c *XAIHTTPClient) logResponse(result *DeferredResult) {
	if len(result.Choices) > 0 {
		choice := result.Choices[0]
		logging.Debug("XAI HTTP response parsed",
			"citations", len(result.Citations),
			"content_length", len(choice.Message.Content),
			"reasoning_length", len(choice.Message.ReasoningContent),
			"has_content", choice.Message.Content != "",
			"has_reasoning", choice.Message.ReasoningContent != "",
			"finish_reason", choice.FinishReason)
	} else {
		logging.Debug("No choices in HTTP response")
	}
}

// getMaskedAPIKey returns a masked version of the API key for logging
func (c *XAIHTTPClient) getMaskedAPIKey() string {
	if len(c.config.APIKey) <= 6 {
		return "***"
	}
	return c.config.APIKey[:3] + "***" + c.config.APIKey[len(c.config.APIKey)-3:]
}
