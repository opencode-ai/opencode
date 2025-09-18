package agent

import (
	"context"
	"testing"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/history"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/lsp"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/pubsub"
	"github.com/opencode-ai/opencode/internal/session"
)

func TestCoderAgentToolsIncludesWebSearch(t *testing.T) {
	// Setup test configuration
	setupTestConfig(t)

	// Create mock services
	mockPermissions := &mockPermissionService{}
	mockSessions := &mockSessionService{}
	mockMessages := &mockMessageService{}
	mockHistory := &mockHistoryService{}
	lspClients := make(map[string]*lsp.Client)

	// Get coder agent tools
	agentTools := CoderAgentTools(
		mockPermissions,
		mockSessions,
		mockMessages,
		mockHistory,
		lspClients,
	)

	// Check if web_search tool is included
	found := false
	var webSearchTool interface{}
	for _, tool := range agentTools {
		info := tool.Info()
		if info.Name == "web_search" {
			found = true
			webSearchTool = tool
			break
		}
	}

	if !found {
		t.Error("CoderAgentTools should include web_search tool")
	}

	// Additional validation
	if webSearchTool != nil {
		// Type assert to verify it implements the correct interface
		if baseTool, ok := webSearchTool.(interface {
			Info() tools.ToolInfo
			Run(context.Context, tools.ToolCall) (tools.ToolResponse, error)
		}); ok {
			info := baseTool.Info()
			if info.Name != "web_search" {
				t.Errorf("Expected tool name 'web_search', got '%s'", info.Name)
			}
			if info.Description == "" {
				t.Error("Web search tool should have a description")
			}
			// Verify it has properties with query parameter
			properties, ok := info.Parameters["properties"].(map[string]interface{})
			if !ok {
				t.Fatal("Parameters should have 'properties' field")
			}
			if _, hasQuery := properties["query"]; !hasQuery {
				t.Error("Web search tool should have 'query' parameter in properties")
			}
			// Verify required fields
			if len(info.Required) == 0 || info.Required[0] != "query" {
				t.Error("Web search tool should require 'query' parameter")
			}
			
			// Verify provider restrictions
			if len(info.Providers) != 1 || info.Providers[0] != "xai" {
				t.Errorf("Web search tool should be restricted to xai provider, got %v", info.Providers)
			}
		} else {
			t.Error("Web search tool does not implement BaseTool interface correctly")
		}
	}
}

// Mock implementations for testing
type mockPermissionService struct{}

func (m *mockPermissionService) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	ch := make(chan pubsub.Event[permission.PermissionRequest])
	close(ch)
	return ch
}

func (m *mockPermissionService) GrantPersistant(permission permission.PermissionRequest) {}

func (m *mockPermissionService) Grant(permission permission.PermissionRequest) {}

func (m *mockPermissionService) Deny(permission permission.PermissionRequest) {}

func (m *mockPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	return true
}

func (m *mockPermissionService) AutoApproveSession(sessionID string) {}

// Mock session service
type mockSessionService struct{}

func (m *mockSessionService) Subscribe(ctx context.Context) <-chan pubsub.Event[session.Session] {
	ch := make(chan pubsub.Event[session.Session])
	close(ch)
	return ch
}

func (m *mockSessionService) Create(ctx context.Context, title string) (session.Session, error) {
	return session.Session{ID: "test-session", Title: title}, nil
}

func (m *mockSessionService) CreateTitleSession(ctx context.Context, parentSessionID string) (session.Session, error) {
	return session.Session{ID: "test-title-session", ParentSessionID: parentSessionID}, nil
}

func (m *mockSessionService) CreateTaskSession(ctx context.Context, toolCallID, parentSessionID, title string) (session.Session, error) {
	return session.Session{ID: "test-task-session", ParentSessionID: parentSessionID, Title: title}, nil
}

func (m *mockSessionService) Get(ctx context.Context, id string) (session.Session, error) {
	return session.Session{ID: id}, nil
}

func (m *mockSessionService) List(ctx context.Context) ([]session.Session, error) {
	return []session.Session{}, nil
}

func (m *mockSessionService) Save(ctx context.Context, session session.Session) (session.Session, error) {
	return session, nil
}

func (m *mockSessionService) Delete(ctx context.Context, id string) error {
	return nil
}

// Mock message service
type mockMessageService struct{}

func (m *mockMessageService) Subscribe(ctx context.Context) <-chan pubsub.Event[message.Message] {
	ch := make(chan pubsub.Event[message.Message])
	close(ch)
	return ch
}

func (m *mockMessageService) Create(ctx context.Context, sessionID string, params message.CreateMessageParams) (message.Message, error) {
	return message.Message{ID: "test-message", SessionID: sessionID}, nil
}

func (m *mockMessageService) Update(ctx context.Context, msg message.Message) error {
	return nil
}

func (m *mockMessageService) Get(ctx context.Context, id string) (message.Message, error) {
	return message.Message{ID: id}, nil
}

func (m *mockMessageService) List(ctx context.Context, sessionID string) ([]message.Message, error) {
	return []message.Message{}, nil
}

func (m *mockMessageService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMessageService) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	return nil
}

// Mock history service
type mockHistoryService struct{}

func (m *mockHistoryService) Subscribe(ctx context.Context) <-chan pubsub.Event[history.File] {
	ch := make(chan pubsub.Event[history.File])
	close(ch)
	return ch
}

func (m *mockHistoryService) Create(ctx context.Context, sessionID, path, content string) (history.File, error) {
	return history.File{ID: "test-file", SessionID: sessionID, Path: path, Content: content}, nil
}

func (m *mockHistoryService) CreateVersion(ctx context.Context, sessionID, path, content string) (history.File, error) {
	return history.File{ID: "test-file-version", SessionID: sessionID, Path: path, Content: content}, nil
}

func (m *mockHistoryService) Get(ctx context.Context, id string) (history.File, error) {
	return history.File{ID: id}, nil
}

func (m *mockHistoryService) GetByPathAndSession(ctx context.Context, path, sessionID string) (history.File, error) {
	return history.File{Path: path, SessionID: sessionID}, nil
}

func (m *mockHistoryService) ListBySession(ctx context.Context, sessionID string) ([]history.File, error) {
	return []history.File{}, nil
}

func (m *mockHistoryService) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]history.File, error) {
	return []history.File{}, nil
}

func (m *mockHistoryService) Update(ctx context.Context, file history.File) (history.File, error) {
	return file, nil
}

func (m *mockHistoryService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockHistoryService) DeleteSessionFiles(ctx context.Context, sessionID string) error {
	return nil
}

// setupTestConfig initializes a minimal configuration for testing
func setupTestConfig(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	cfg, err := config.Load(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Ensure MCPServers map is initialized to prevent nil pointer
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]config.MCPServer)
	}
}
