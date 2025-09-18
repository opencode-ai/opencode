package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opencode-ai/opencode/internal/logging"
)

// XAIModelInfo represents the detailed model information from xAI API
type XAIModelInfo struct {
	ID                         string   `json:"id"`
	Fingerprint                string   `json:"fingerprint"`
	Created                    int64    `json:"created"`
	Object                     string   `json:"object"`
	OwnedBy                    string   `json:"owned_by"`
	Version                    string   `json:"version"`
	InputModalities            []string `json:"input_modalities"`
	OutputModalities           []string `json:"output_modalities"`
	PromptTextTokenPrice       int      `json:"prompt_text_token_price"`
	CachedPromptTextTokenPrice int      `json:"cached_prompt_text_token_price"`
	PromptImageTokenPrice      int      `json:"prompt_image_token_price"`
	CompletionTextTokenPrice   int      `json:"completion_text_token_price"`
	SearchPrice                int      `json:"search_price"`
	Aliases                    []string `json:"aliases"`
}

// XAIImageModelInfo represents image generation model information
type XAIImageModelInfo struct {
	ID                       string   `json:"id"`
	Fingerprint              string   `json:"fingerprint"`
	MaxPromptLength          int      `json:"max_prompt_length"`
	Created                  int64    `json:"created"`
	Object                   string   `json:"object"`
	OwnedBy                  string   `json:"owned_by"`
	Version                  string   `json:"version"`
	InputModalities          []string `json:"input_modalities"`
	OutputModalities         []string `json:"output_modalities"`
	ImagePrice               int      `json:"image_price"`
	PromptTextTokenPrice     int      `json:"prompt_text_token_price"`
	PromptImageTokenPrice    int      `json:"prompt_image_token_price"`
	GeneratedImageTokenPrice int      `json:"generated_image_token_price"`
	Aliases                  []string `json:"aliases"`
}

// XAILanguageModelsResponse represents the response from /language-models
type XAILanguageModelsResponse struct {
	Models []XAIModelInfo `json:"models"`
}

// XAIImageModelsResponse represents the response from /image-generation-models
type XAIImageModelsResponse struct {
	Models []XAIImageModelInfo `json:"models"`
}

// ModelCapabilities represents the capabilities of a model
type ModelCapabilities struct {
	SupportsText        bool
	SupportsImageInput  bool
	SupportsImageOutput bool
	SupportsWebSearch   bool
	MaxPromptLength     int
	Aliases             []string
}

// DiscoverModelCapabilities queries the xAI API to discover model capabilities
func (x *xaiClient) DiscoverModelCapabilities(ctx context.Context, modelID string) (*ModelCapabilities, error) {
	// First try language models endpoint
	langCaps, err := x.getLanguageModelCapabilities(ctx, modelID)
	if err == nil && langCaps != nil {
		return langCaps, nil
	}

	// Then try image generation models endpoint
	imgCaps, err := x.getImageModelCapabilities(ctx, modelID)
	if err == nil && imgCaps != nil {
		return imgCaps, nil
	}

	// Fallback to basic model info
	return x.getBasicModelCapabilities(ctx, modelID)
}

// getLanguageModelCapabilities fetches capabilities from language models endpoint
func (x *xaiClient) getLanguageModelCapabilities(ctx context.Context, modelID string) (*ModelCapabilities, error) {
	url := fmt.Sprintf("%s/language-models/%s", x.getBaseURL(), modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Not a language model
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var modelInfo XAIModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
		return nil, err
	}

	caps := &ModelCapabilities{
		Aliases: modelInfo.Aliases,
	}

	// Check input modalities
	for _, mod := range modelInfo.InputModalities {
		switch mod {
		case "text":
			caps.SupportsText = true
		case "image":
			caps.SupportsImageInput = true
		}
	}

	// Check output modalities
	for _, mod := range modelInfo.OutputModalities {
		switch mod {
		case "text":
			// Text output is standard
		case "image":
			caps.SupportsImageOutput = true
		}
	}

	// Web search is available for all language models
	caps.SupportsWebSearch = caps.SupportsText

	logging.Debug("Discovered language model capabilities",
		"model", modelID,
		"text", caps.SupportsText,
		"image_input", caps.SupportsImageInput,
		"image_output", caps.SupportsImageOutput,
		"web_search", caps.SupportsWebSearch)

	return caps, nil
}

// getImageModelCapabilities fetches capabilities from image generation models endpoint
func (x *xaiClient) getImageModelCapabilities(ctx context.Context, modelID string) (*ModelCapabilities, error) {
	url := fmt.Sprintf("%s/image-generation-models/%s", x.getBaseURL(), modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Not an image generation model
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var modelInfo XAIImageModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
		return nil, err
	}

	caps := &ModelCapabilities{
		SupportsText:        true, // Image generation takes text prompts
		SupportsImageOutput: true,
		MaxPromptLength:     modelInfo.MaxPromptLength,
		Aliases:             modelInfo.Aliases,
	}

	logging.Debug("Discovered image generation model capabilities",
		"model", modelID,
		"max_prompt_length", caps.MaxPromptLength)

	return caps, nil
}

// getBasicModelCapabilities fetches basic model info as fallback
func (x *xaiClient) getBasicModelCapabilities(ctx context.Context, modelID string) (*ModelCapabilities, error) {
	url := fmt.Sprintf("%s/models/%s", x.getBaseURL(), modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Basic model info doesn't provide capability details
	// Return minimal capabilities
	return &ModelCapabilities{
		SupportsText: true, // Assume text support for all models
	}, nil
}

// getBaseURL returns the base URL for API requests
func (x *xaiClient) getBaseURL() string {
	if x.options.baseURL != "" {
		return x.options.baseURL
	}
	return "https://api.x.ai/v1"
}

// ListAllModels fetches all available models from xAI API
func (x *xaiClient) ListAllModels(ctx context.Context) ([]XAIModelInfo, []XAIImageModelInfo, error) {
	var languageModels []XAIModelInfo
	var imageModels []XAIImageModelInfo

	// Fetch language models
	langURL := fmt.Sprintf("%s/language-models", x.getBaseURL())
	langReq, err := http.NewRequestWithContext(ctx, "GET", langURL, nil)
	if err != nil {
		return nil, nil, err
	}
	langReq.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	langResp, err := client.Do(langReq)
	if err != nil {
		return nil, nil, err
	}
	defer langResp.Body.Close()

	if langResp.StatusCode == http.StatusOK {
		var langModelsResp XAILanguageModelsResponse
		if err := json.NewDecoder(langResp.Body).Decode(&langModelsResp); err != nil {
			return nil, nil, err
		}
		languageModels = langModelsResp.Models
	}

	// Fetch image generation models
	imgURL := fmt.Sprintf("%s/image-generation-models", x.getBaseURL())
	imgReq, err := http.NewRequestWithContext(ctx, "GET", imgURL, nil)
	if err != nil {
		return languageModels, nil, err
	}
	imgReq.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)

	imgResp, err := client.Do(imgReq)
	if err != nil {
		return languageModels, nil, err
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode == http.StatusOK {
		var imgModelsResp XAIImageModelsResponse
		if err := json.NewDecoder(imgResp.Body).Decode(&imgModelsResp); err != nil {
			return languageModels, nil, err
		}
		imageModels = imgModelsResp.Models
	}

	return languageModels, imageModels, nil
}
