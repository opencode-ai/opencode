package provider

import (
	"context"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/stretchr/testify/require"
)

func TestMockProvider(t *testing.T) {
	mockClient := &MockClient{
		SendMessagesFunc: func(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
			return &ProviderResponse{
				Content: "Hello, world!",
			}, nil
		},
	}

	provider, err := NewProvider(
		models.ProviderMock,
		WithMockClient(mockClient),
	)
	require.NoError(t, err)

	resp, err := provider.SendMessages(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Equal(t, "Hello, world!", resp.Content)
}
