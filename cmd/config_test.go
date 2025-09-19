package cmd

import (
	"testing"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestConfigCmd(t *testing.T) {
	_, err := config.Load(".", false)
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"config", "--key", "openai", "--value", "test-key"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	cfg := config.Get()
	require.Equal(t, "test-key", cfg.Providers[models.ProviderOpenAI].APIKey)
}
