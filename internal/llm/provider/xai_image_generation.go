package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/opencode-ai/opencode/internal/logging"
)

// ImageGenerationRequest represents a request to generate images
type ImageGenerationRequest struct {
	Prompt         string
	Model          string
	N              int    // Number of images (1-10)
	ResponseFormat string // "url" or "b64_json"
}

// ImageGenerationResponse represents the response from image generation
type ImageGenerationResponse struct {
	Images        []GeneratedImage
	RevisedPrompt string
	Model         string
	Created       time.Time
}

// GeneratedImage represents a single generated image
type GeneratedImage struct {
	URL         string // For URL format
	Base64      string // For b64_json format
	ContentType string // MIME type
}

// GenerateImages generates one or more images based on a text prompt
func (x *xaiClient) GenerateImages(ctx context.Context, req ImageGenerationRequest) (*ImageGenerationResponse, error) {
	// Validate request
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	if req.N < 1 {
		req.N = 1
	} else if req.N > 10 {
		return nil, fmt.Errorf("n must be between 1 and 10, got %d", req.N)
	}

	if req.ResponseFormat == "" {
		req.ResponseFormat = "url"
	} else if req.ResponseFormat != "url" && req.ResponseFormat != "b64_json" {
		return nil, fmt.Errorf("response_format must be 'url' or 'b64_json', got %s", req.ResponseFormat)
	}

	// Use the model from request or fall back to provider's model
	model := req.Model
	if model == "" {
		model = string(x.providerOptions.model.APIModel)
	}

	// Check if model supports image generation
	caps, err := x.DiscoverModelCapabilities(ctx, model)
	if err != nil {
		logging.Warn("Failed to discover model capabilities, proceeding anyway", "error", err)
	} else if caps != nil && !caps.SupportsImageOutput {
		return nil, fmt.Errorf("model %s does not support image generation", model)
	}

	// Create the image generation request
	params := openai.ImageGenerateParams{
		Model:          openai.ImageModel(model),
		Prompt:         req.Prompt,
		N:              openai.Int(int64(req.N)),
		ResponseFormat: openai.ImageGenerateParamsResponseFormat(req.ResponseFormat),
		// xAI doesn't support quality, size, or style parameters
	}

	logging.Debug("Generating images",
		"model", model,
		"prompt_length", len(req.Prompt),
		"n", req.N,
		"format", req.ResponseFormat)

	// Make the API call
	startTime := time.Now()
	result, err := x.client.Images.Generate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("image generation failed: %w", err)
	}

	elapsed := time.Since(startTime)
	logging.Debug("Image generation completed",
		"model", model,
		"n", req.N,
		"elapsed", elapsed)

	// Convert response
	response := &ImageGenerationResponse{
		Model:   model,
		Created: time.Unix(result.Created, 0),
		Images:  make([]GeneratedImage, 0, len(result.Data)),
	}

	// Extract revised prompt if available
	if len(result.Data) > 0 && result.Data[0].RevisedPrompt != "" {
		response.RevisedPrompt = result.Data[0].RevisedPrompt
	}

	// Process each generated image
	for i, imgData := range result.Data {
		img := GeneratedImage{
			ContentType: "image/jpeg", // xAI generates JPEGs
		}

		if req.ResponseFormat == "url" {
			img.URL = imgData.URL
		} else {
			// b64_json format
			img.Base64 = imgData.B64JSON
		}

		response.Images = append(response.Images, img)

		logging.Debug("Processed generated image",
			"index", i,
			"has_url", img.URL != "",
			"has_base64", img.Base64 != "")
	}

	// Track fingerprint if available
	if x.providerOptions.model.Provider == "xai" {
		// Note: Image generation responses don't include system fingerprint in the same way
		// but we can still track the generation event
		logging.Debug("Image generation completed",
			"model", model,
			"images_generated", len(response.Images))
	}

	return response, nil
}

// GenerateImage is a convenience method to generate a single image
func (x *xaiClient) GenerateImage(ctx context.Context, prompt string) (*GeneratedImage, error) {
	req := ImageGenerationRequest{
		Prompt:         prompt,
		N:              1,
		ResponseFormat: "url",
	}

	resp, err := x.GenerateImages(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Images) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	return &resp.Images[0], nil
}

// SaveGeneratedImage downloads and returns the image data from a URL-based response
func (x *xaiClient) SaveGeneratedImage(ctx context.Context, image *GeneratedImage) ([]byte, error) {
	if image.Base64 != "" {
		// Already have base64 data, decode it
		// Remove data URL prefix if present
		b64Data := image.Base64
		if strings.HasPrefix(b64Data, "data:") {
			parts := strings.SplitN(b64Data, ",", 2)
			if len(parts) == 2 {
				b64Data = parts[1]
			}
		}

		return base64.StdEncoding.DecodeString(b64Data)
	}

	if image.URL == "" {
		return nil, fmt.Errorf("no image data available")
	}

	// Download from URL
	req, err := http.NewRequestWithContext(ctx, "GET", image.URL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ConvertToDataURL converts image data to a data URL
func ConvertImageToDataURL(data []byte, contentType string) string {
	if contentType == "" {
		contentType = "image/jpeg"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}
