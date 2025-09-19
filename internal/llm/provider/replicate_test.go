package provider

import (
	"context"
	"os"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestReplicateProvider(t *testing.T) {
	if os.Getenv("REPLICATE_API_KEY") == "" {
		t.Skip("REPLICATE_API_KEY not set, skipping test")
	}

	provider, err := NewProvider(
		models.ProviderReplicate,
		WithModel(models.SupportedModels[models.ReplicateLlama270BChat]),
		WithAPIKey(os.Getenv("REPLICATE_API_KEY")),
	)
	require.NoError(t, err)

	_, err = provider.SendMessages(context.Background(), nil, nil)
	require.NoError(t, err)
}
