package provider

import (
	"context"
	"os"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestHuggingFaceProvider(t *testing.T) {
	if os.Getenv("HUGGINGFACE_API_KEY") == "" {
		t.Skip("HUGGINGFACE_API_KEY not set, skipping test")
	}

	provider, err := NewProvider(
		models.ProviderHuggingFace,
		WithModel(models.SupportedModels[models.HuggingFaceMistral7BInstruct]),
		WithAPIKey(os.Getenv("HUGGINGFACE_API_KEY")),
	)
	require.NoError(t, err)

	_, err = provider.SendMessages(context.Background(), nil, nil)
	require.NoError(t, err)
}
