package provider

import (
	"context"

	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
)

type MockClient struct {
	SendMessagesFunc   func(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)
	StreamResponseFunc func(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
}

func (m *MockClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	if m.SendMessagesFunc != nil {
		return m.SendMessagesFunc(ctx, messages, tools)
	}
	return nil, nil
}

func (m *MockClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	if m.StreamResponseFunc != nil {
		return m.StreamResponseFunc(ctx, messages, tools)
	}
	return nil
}

func newMockClient(opts providerClientOptions) *MockClient {
	return opts.mockClient
}
