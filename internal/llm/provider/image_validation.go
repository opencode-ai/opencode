package provider

import (
	"fmt"
	"strings"

	"github.com/opencode-ai/opencode/internal/message"
)

const (
	// MaxImageSize is the maximum allowed image size for xAI (20MiB)
	MaxImageSize = 20 * 1024 * 1024 // 20 MiB
)

// SupportedImageFormats lists the image formats supported by xAI
var SupportedImageFormats = []string{"image/jpeg", "image/jpg", "image/png"}

// ValidateImageAttachment validates that an image attachment meets xAI requirements
func ValidateImageAttachment(attachment message.Attachment) error {
	// Check file size
	if len(attachment.Content) > MaxImageSize {
		return fmt.Errorf("image size exceeds maximum allowed size of 20MiB (current: %.2fMiB)",
			float64(len(attachment.Content))/(1024*1024))
	}

	// Check MIME type
	mimeType := strings.ToLower(attachment.MimeType)
	supported := false
	for _, format := range SupportedImageFormats {
		if mimeType == format {
			supported = true
			break
		}
	}

	if !supported {
		return fmt.Errorf("unsupported image format: %s (supported: %s)",
			mimeType, strings.Join(SupportedImageFormats, ", "))
	}

	return nil
}

// IsVisionModel checks if a model supports image understanding
func IsVisionModel(modelID string) bool {
	visionModels := []string{
		"grok-2-vision-1212",
		"grok-4-0709", // grok-4 supports vision
	}

	for _, vm := range visionModels {
		if modelID == vm {
			return true
		}
	}

	return false
}
