package provider

import (
	"context"
	"os"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestCohereProvider(t *testing.T) {
	if os.Getenv("COHERE_API_KEY") == "" {
		t.Skip("COHERE_API_KEY not set, skipping test")
	}

	provider, err := NewProvider(
		models.ProviderCohere,
		WithModel(models.SupportedModels[models.CohereCommandRPlus]),
		WithAPIKey(os.Getenv("COHERE_API_KEY")),
	)
	require.NoError(t, err)

	_, err = provider.SendMessages(context.Background(), nil, nil)
	require.NoError(t, err)
}
