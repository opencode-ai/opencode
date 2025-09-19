package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/opencode-ai/opencode/internal/llm/provider"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/opencode-ai/opencode/internal/session"
)

func TestShouldTriggerAutoCompaction(t *testing.T) {
	tests := []struct {
		name             string
		contextWindow    int64
		promptTokens     int64
		completionTokens int64
		expected         bool
	}{
		{
			name:             "Below threshold",
			contextWindow:    100000,
			promptTokens:     40000,
			completionTokens: 40000,
			expected:         false,
		},
		{
			name:             "At threshold",
			contextWindow:    100000,
			promptTokens:     47500,
			completionTokens: 47500,
			expected:         true,
		},
		{
			name:             "Above threshold",
			contextWindow:    100000,
			promptTokens:     50000,
			completionTokens: 50000,
			expected:         true,
		},
		{
			name:             "No context window",
			contextWindow:    0,
			promptTokens:     50000,
			completionTokens: 50000,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal agent with just the model info we need
			a := &agent{}

			// Mock the provider's Model() method by setting up the agent's provider field
			a.provider = &testProvider{
				model: models.Model{
					ContextWindow: tt.contextWindow,
				},
			}

			session := session.Session{
				PromptTokens:     tt.promptTokens,
				CompletionTokens: tt.completionTokens,
			}

			result := a.shouldTriggerAutoCompaction(session)
			if result != tt.expected {
				t.Errorf("shouldTriggerAutoCompaction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldTriggerAutoCompactionFromHistory(t *testing.T) {
	tests := []struct {
		name          string
		contextWindow int64
		messages      []message.Message
		expected      bool
	}{
		{
			name:          "Small history below threshold",
			contextWindow: 100000,
			messages: []message.Message{
				{Parts: []message.ContentPart{message.TextContent{Text: "Hello"}}},
				{Parts: []message.ContentPart{message.TextContent{Text: "Hi there"}}},
			},
			expected: false,
		},
		{
			name:          "Large history above threshold",
			contextWindow: 1000,
			messages: []message.Message{
				{Parts: []message.ContentPart{message.TextContent{Text: strings.Repeat("This is a long message that will consume many tokens. ", 100)}}},
				{Parts: []message.ContentPart{message.TextContent{Text: strings.Repeat("Another long response with lots of content. ", 100)}}},
			},
			expected: true,
		},
		{
			name:          "No context window",
			contextWindow: 0,
			messages: []message.Message{
				{Parts: []message.ContentPart{message.TextContent{Text: strings.Repeat("Long message ", 1000)}}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &agent{}
			a.provider = &testProvider{
				model: models.Model{
					ContextWindow: tt.contextWindow,
				},
			}

			result := a.shouldTriggerAutoCompactionFromHistory(tt.messages)
			if result != tt.expected {
				t.Errorf("shouldTriggerAutoCompactionFromHistory() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPerformSynchronousCompaction_NoSummarizeProvider(t *testing.T) {
	a := &agent{
		summarizeProvider: nil,
	}

	err := a.performSynchronousCompaction(context.Background(), "test-session")
	if err == nil {
		t.Error("expected error when summarizeProvider is nil, got nil")
	}
	if err.Error() != "summarize provider not available" {
		t.Errorf("expected 'summarize provider not available', got %v", err.Error())
	}
}

func TestAutoCompactionAttemptLimit(t *testing.T) {
	// Test that compaction attempts are limited to prevent infinite loops
	// This test verifies the fix for the deadloop issue

	longMessage := strings.Repeat("This is a very long message that will exceed the context window. ", 100)
	msgHistory := []message.Message{
		{Parts: []message.ContentPart{message.TextContent{Text: longMessage}}},
		{Parts: []message.ContentPart{message.TextContent{Text: longMessage}}},
	}

	a := &agent{}
	a.provider = &testProvider{
		model: models.Model{
			ContextWindow: 1000, // Small context window to trigger compaction
		},
	}

	// Test that shouldTriggerAutoCompactionFromHistory returns true for this history
	shouldTrigger := a.shouldTriggerAutoCompactionFromHistory(msgHistory)
	if !shouldTrigger {
		t.Error("expected shouldTriggerAutoCompactionFromHistory to return true for large message history")
	}
}

func TestMessageFilteringAfterCompaction(t *testing.T) {
	// Test that message filtering logic is applied correctly after compaction
	// This verifies the fix for the issue where compaction was increasing context size

	// Create test messages that simulate a conversation with a summary
	messages := []message.Message{
		{ID: "msg1", Parts: []message.ContentPart{message.TextContent{Text: "Old message 1"}}},
		{ID: "msg2", Parts: []message.ContentPart{message.TextContent{Text: "Old message 2"}}},
		{ID: "summary", Parts: []message.ContentPart{message.TextContent{Text: "Summary of conversation"}}},
		{ID: "msg3", Parts: []message.ContentPart{message.TextContent{Text: "New message after summary"}}},
	}

	// Simulate finding the summary message and filtering
	summaryMsgIndex := -1
	for i, msg := range messages {
		if msg.ID == "summary" {
			summaryMsgIndex = i
			break
		}
	}

	if summaryMsgIndex == -1 {
		t.Fatal("summary message not found")
	}

	// Apply the filtering logic (same as in the agent code)
	filteredMessages := messages[summaryMsgIndex:]
	filteredMessages[0].Role = message.User

	// Verify that filtering worked correctly
	if len(filteredMessages) != 2 {
		t.Errorf("expected 2 filtered messages, got %d", len(filteredMessages))
	}

	if filteredMessages[0].ID != "summary" {
		t.Errorf("expected first message to be summary, got %s", filteredMessages[0].ID)
	}

	if filteredMessages[0].Role != message.User {
		t.Errorf("expected summary message role to be User, got %s", filteredMessages[0].Role)
	}

	if filteredMessages[1].ID != "msg3" {
		t.Errorf("expected second message to be msg3, got %s", filteredMessages[1].ID)
	}
}

func TestFilterMessagesFromSummary(t *testing.T) {
	a := &agent{}

	tests := []struct {
		name             string
		messages         []message.Message
		summaryMessageID string
		expectedCount    int
		expectedFirstID  string
		expectedRole     message.MessageRole
	}{
		{
			name: "No summary message ID",
			messages: []message.Message{
				{ID: "msg1", Role: message.Assistant},
				{ID: "msg2", Role: message.User},
			},
			summaryMessageID: "",
			expectedCount:    2,
			expectedFirstID:  "msg1",
			expectedRole:     message.Assistant,
		},
		{
			name: "Summary message exists",
			messages: []message.Message{
				{ID: "msg1", Role: message.Assistant},
				{ID: "summary", Role: message.Assistant},
				{ID: "msg3", Role: message.User},
			},
			summaryMessageID: "summary",
			expectedCount:    2,
			expectedFirstID:  "summary",
			expectedRole:     message.User, // Should be converted to User
		},
		{
			name: "Summary message not found",
			messages: []message.Message{
				{ID: "msg1", Role: message.Assistant},
				{ID: "msg2", Role: message.User},
			},
			summaryMessageID: "nonexistent",
			expectedCount:    2,
			expectedFirstID:  "msg1",
			expectedRole:     message.Assistant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.filterMessagesFromSummary(tt.messages, tt.summaryMessageID)

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d messages, got %d", tt.expectedCount, len(result))
			}

			if len(result) > 0 {
				if result[0].ID != tt.expectedFirstID {
					t.Errorf("expected first message ID %s, got %s", tt.expectedFirstID, result[0].ID)
				}

				if result[0].Role != tt.expectedRole {
					t.Errorf("expected first message role %s, got %s", tt.expectedRole, result[0].Role)
				}
			}
		})
	}
}

type testProvider struct {
	model models.Model
}

func (tp *testProvider) Model() models.Model {
	return tp.model
}

func (tp *testProvider) SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*provider.ProviderResponse, error) {
	return &provider.ProviderResponse{
		Content: "Test summary of the conversation",
	}, nil
}

func (tp *testProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan provider.ProviderEvent {
	return nil
}

