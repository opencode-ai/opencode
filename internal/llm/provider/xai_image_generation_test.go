package provider

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_ImageGeneration(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("model capability discovery", func(t *testing.T) {
		// Create provider with image generation model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Image]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Test capability discovery
		ctx := context.Background()
		caps, err := xaiClient.DiscoverModelCapabilities(ctx, "grok-2-image")
		require.NoError(t, err)
		require.NotNil(t, caps)

		assert.True(t, caps.SupportsText, "Image generation models should support text prompts")
		assert.True(t, caps.SupportsImageOutput, "Image generation models should support image output")
		assert.False(t, caps.SupportsImageInput, "Image generation models don't typically support image input")

		t.Logf("Image generation model capabilities: %+v", caps)
	})

	t.Run("single image generation", func(t *testing.T) {
		// Create provider with image generation model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Image]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Generate a simple image
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		image, err := xaiClient.GenerateImage(ctx, "A simple red circle on white background")
		require.NoError(t, err)
		require.NotNil(t, image)

		// Check that we got either a URL or base64 data
		assert.True(t, image.URL != "" || image.Base64 != "", "Should have either URL or base64 data")
		assert.Equal(t, "image/jpeg", image.ContentType)

		t.Logf("Generated image: URL=%v, has_base64=%v", image.URL != "", image.Base64 != "")
	})

	t.Run("multiple image generation", func(t *testing.T) {
		// Create provider with image generation model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Image]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Generate multiple images
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		req := ImageGenerationRequest{
			Prompt:         "A cat sitting on a tree branch",
			N:              3,
			ResponseFormat: "url",
		}

		resp, err := xaiClient.GenerateImages(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Len(t, resp.Images, 3, "Should generate 3 images")
		assert.NotEmpty(t, resp.RevisedPrompt, "Should have revised prompt")

		// Check each image
		for i, img := range resp.Images {
			assert.NotEmpty(t, img.URL, "Image %d should have URL", i)
			assert.Equal(t, "image/jpeg", img.ContentType)
		}

		t.Logf("Generated %d images", len(resp.Images))
		t.Logf("Original prompt: %s", req.Prompt)
		t.Logf("Revised prompt: %s", resp.RevisedPrompt)
	})

	t.Run("base64 format", func(t *testing.T) {
		// Create provider with image generation model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Image]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		// Generate image in base64 format
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		req := ImageGenerationRequest{
			Prompt:         "A simple blue square",
			N:              1,
			ResponseFormat: "b64_json",
		}

		resp, err := xaiClient.GenerateImages(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Images, 1)

		img := resp.Images[0]
		assert.NotEmpty(t, img.Base64, "Should have base64 data")
		assert.Empty(t, img.URL, "Should not have URL when using b64_json format")

		// Test saving the image
		data, err := xaiClient.SaveGeneratedImage(ctx, &img)
		require.NoError(t, err)
		assert.Greater(t, len(data), 1000, "Image data should be reasonably sized")

		t.Logf("Generated base64 image with %d bytes", len(data))
	})

	t.Run("validation tests", func(t *testing.T) {
		// Create provider with image generation model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Image]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Test empty prompt
		_, err = xaiClient.GenerateImages(ctx, ImageGenerationRequest{
			Prompt: "",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt cannot be empty")

		// Test invalid N
		_, err = xaiClient.GenerateImages(ctx, ImageGenerationRequest{
			Prompt: "test",
			N:      15,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "n must be between 1 and 10")

		// Test invalid response format
		_, err = xaiClient.GenerateImages(ctx, ImageGenerationRequest{
			Prompt:         "test",
			ResponseFormat: "invalid",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "response_format must be")
	})

	t.Run("capability discovery for grok-4", func(t *testing.T) {
		// Test if grok-4 supports image generation
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok4]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()
		caps, err := xaiClient.DiscoverModelCapabilities(ctx, "grok-4")
		if err != nil {
			t.Logf("Could not discover grok-4 capabilities: %v", err)
			return
		}

		t.Logf("Grok-4 capabilities: text=%v, image_input=%v, image_output=%v, web_search=%v",
			caps.SupportsText, caps.SupportsImageInput, caps.SupportsImageOutput, caps.SupportsWebSearch)

		// If grok-4 supports image generation, test it
		if caps.SupportsImageOutput {
			t.Log("Grok-4 supports image generation! Testing...")

			req := ImageGenerationRequest{
				Prompt: "A simple test image",
				Model:  "grok-4",
				N:      1,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			resp, err := xaiClient.GenerateImages(ctx, req)
			if err != nil {
				t.Logf("Grok-4 image generation failed: %v", err)
			} else {
				t.Logf("Grok-4 successfully generated %d images", len(resp.Images))
			}
		}
	})
}
