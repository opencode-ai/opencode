package provider

import (
	"context"
	"os"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestOllamaProvider(t *testing.T) {
	if os.Getenv("OLLAMA_ENDPOINT") == "" {
		t.Skip("OLLAMA_ENDPOINT not set, skipping test")
	}

	provider, err := NewProvider(
		models.ProviderOllama,
		WithModel(models.SupportedModels[models.OllamaLlama3]),
	)
	require.NoError(t, err)

	_, err = provider.SendMessages(context.Background(), nil, nil)
	require.NoError(t, err)
}
