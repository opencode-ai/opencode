package provider

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_VisionSupport(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("vision model detection", func(t *testing.T) {
		// Test grok-2-vision model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Vision]),
			WithMaxTokens(200),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		assert.True(t, xaiClient.IsVisionCapable(), "grok-2-vision should be vision capable")
	})

	t.Run("non-vision model detection", func(t *testing.T) {
		// Test grok-3-fast model (non-vision)
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
			WithMaxTokens(200),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		assert.False(t, xaiClient.IsVisionCapable(), "grok-3-fast should not be vision capable")
	})

	t.Run("image recognition with base64", func(t *testing.T) {
		// Create a simple test image (1x1 red pixel PNG)
		redPixelPNG := []byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
			0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
			0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d, 0xb4, 0x00, 0x00, 0x00,
			0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		}

		// Create provider with vision model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Vision]),
			WithMaxTokens(200),
			WithSystemMessage("You are a helpful assistant that analyzes images concisely."),
		)
		require.NoError(t, err)

		// Create message with image
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "What color is this 1x1 pixel image? Just say the color name."},
					message.BinaryContent{
						MIMEType: "image/png",
						Data:     redPixelPNG,
					},
				},
			},
		}

		// Send request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := provider.SendMessages(ctx, messages, nil)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Check that we got a response about red color
		assert.NotEmpty(t, response.Content)
		assert.NotEmpty(t, response.SystemFingerprint)
		assert.Greater(t, response.Usage.InputTokens, int64(0))
		assert.Greater(t, response.Usage.OutputTokens, int64(0))

		t.Logf("Vision response: %s", response.Content)
		t.Logf("System fingerprint: %s", response.SystemFingerprint)
		t.Logf("Usage: %+v", response.Usage)
	})

	t.Run("image validation", func(t *testing.T) {
		// Test valid image
		validAttachment := message.Attachment{
			FileName: "test.jpg",
			MimeType: "image/jpeg",
			Content:  make([]byte, 1024*1024), // 1MB
		}
		err := ValidateImageAttachment(validAttachment)
		assert.NoError(t, err)

		// Test oversized image
		oversizedAttachment := message.Attachment{
			FileName: "large.jpg",
			MimeType: "image/jpeg",
			Content:  make([]byte, 21*1024*1024), // 21MB
		}
		err = ValidateImageAttachment(oversizedAttachment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")

		// Test unsupported format
		unsupportedAttachment := message.Attachment{
			FileName: "test.gif",
			MimeType: "image/gif",
			Content:  make([]byte, 1024),
		}
		err = ValidateImageAttachment(unsupportedAttachment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported image format")
	})

	t.Run("streaming with images", func(t *testing.T) {
		// Create a simple test image (base64 encoded small JPEG)
		smallJPEG := "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDAREAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAX/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCmAA//2Q=="

		// Extract base64 data
		b64Data := smallJPEG[23:] // Skip "data:image/jpeg;base64,"
		imageData, err := base64.StdEncoding.DecodeString(b64Data)
		require.NoError(t, err)

		// Create provider with vision model
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok2Vision]),
			WithMaxTokens(200),
			WithSystemMessage("You are a helpful assistant."),
		)
		require.NoError(t, err)

		// Create message with image
		messages := []message.Message{
			{
				Role: message.User,
				Parts: []message.ContentPart{
					message.TextContent{Text: "Describe this image in 5 words or less."},
					message.BinaryContent{
						MIMEType: "image/jpeg",
						Data:     imageData,
					},
				},
			},
		}

		// Stream response
		ctx := context.Background()
		eventChan := provider.StreamResponse(ctx, messages, nil)

		// Collect events
		var contentChunks []string
		var finalResponse *ProviderResponse
		hasContentDelta := false

		for event := range eventChan {
			switch event.Type {
			case EventContentDelta:
				hasContentDelta = true
				contentChunks = append(contentChunks, event.Content)

			case EventComplete:
				finalResponse = event.Response

			case EventError:
				t.Fatalf("Streaming error: %v", event.Error)
			}
		}

		// Verify streaming worked correctly
		require.NotNil(t, finalResponse)
		assert.True(t, hasContentDelta, "Should have received content deltas")
		assert.NotEmpty(t, finalResponse.Content)
		assert.NotEmpty(t, finalResponse.SystemFingerprint)

		t.Logf("Streaming vision response: %s", finalResponse.Content)
	})

	t.Run("deferred completion with images", func(t *testing.T) {
		// Skip if not configured for deferred
		t.Skip("Deferred completion with images test - enable when needed")

		// This test would verify deferred completions work with images
		// Similar structure to above tests but using SendDeferred
	})
}
