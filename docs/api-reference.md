# OpenCode API Reference

## Overview

This document provides a comprehensive reference for OpenCode's internal APIs, interfaces, and key components. While OpenCode is primarily a CLI application, understanding its internal architecture is crucial for development and extension.

## Core Interfaces

### App Service Interface

The main application orchestrator that coordinates all services.

```go
type App struct {
    Sessions    session.Service
    Messages    message.Service
    History     history.Service
    Permissions permission.Service

    CoderAgent agent.Service

    LSPClients map[string]*lsp.Client

    clientsMutex sync.RWMutex

    watcherCancelFuncs []context.CancelFunc
    cancelFuncsMutex   sync.Mutex
    watcherWG          sync.WaitGroup
}

func New(ctx context.Context, conn *sql.DB) (*App, error)
func (a *App) RunNonInteractive(ctx context.Context, prompt string, outputFormat string, quiet bool) error
func (a *App) Shutdown()
```

**Usage Example:**
```go
app, err := app.New(ctx, db)
if err != nil {
    return err
}
defer app.Shutdown()

// Non-interactive mode
err = app.RunNonInteractive(ctx, "Explain Go contexts", "json", false)
```

## Configuration API

### Config Structure

```go
type Config struct {
    Data         Data                                    `json:"data"`
    WD           string                                  `json:"wd"`
    Debug        bool                                    `json:"debug"`
    DebugLSP     bool                                    `json:"debugLSP"`
    ContextPaths []string                               `json:"contextPaths"`
    TUI          TUIConfig                              `json:"tui"`
    MCPServers   map[string]MCPServer                   `json:"mcpServers"`
    Providers    map[models.ModelProvider]Provider      `json:"providers"`
    Agents       map[AgentName]Agent                    `json:"agents"`
    LSP          map[string]LSPConfig                   `json:"lsp"`
    Shell        ShellConfig                            `json:"shell"`
    AutoCompact  bool                                   `json:"autoCompact"`
}

type Agent struct {
    Model           string `json:"model"`
    MaxTokens       int    `json:"maxTokens"`
    ReasoningEffort string `json:"reasoningEffort,omitempty"`
}

type Provider struct {
    APIKey   string `json:"apiKey"`
    Disabled bool   `json:"disabled"`
}

type LSPConfig struct {
    Disabled bool     `json:"disabled"`
    Command  string   `json:"command"`
    Args     []string `json:"args,omitempty"`
    Options  any      `json:"options,omitempty"`
}
```

### Configuration Functions

```go
func Load(workingDir string, debug bool) (*Config, error)
func (c *Config) GetAgent(name AgentName) (Agent, bool)
func (c *Config) GetProvider(provider models.ModelProvider) (Provider, bool)
func (c *Config) IsProviderEnabled(provider models.ModelProvider) bool
func (c *Config) GetPreferredProvider() models.ModelProvider
```

## Session Management API

### Session Service Interface

```go
type Service interface {
    pubsub.Suscriber[Session]
    Create(ctx context.Context, title string) (Session, error)
    CreateTitleSession(ctx context.Context, parentSessionID string) (Session, error)
    CreateTaskSession(ctx context.Context, toolCallID, parentSessionID, title string) (Session, error)
    Get(ctx context.Context, id string) (Session, error)
    List(ctx context.Context) ([]Session, error)
    Save(ctx context.Context, session Session) (Session, error)
    Delete(ctx context.Context, id string) error
}

type Session struct {
    ID               string
    ParentSessionID  string
    Title            string
    MessageCount     int64
    PromptTokens     int64
    CompletionTokens int64
    SummaryMessageID string
    Cost             float64
    CreatedAt        int64
    UpdatedAt        int64
}
```

**Usage Example:**
```go
session, err := sessionService.CreateSession(ctx, "Debug Session", nil)
if err != nil {
    return err
}

sessions, err := sessionService.ListSessions(ctx, 10, 0)
if err != nil {
    return err
}
```

## Message Management API

### Message Service Interface

```go
type Service interface {
    pubsub.Suscriber[Message]
    Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error)
    Update(ctx context.Context, message Message) error
    Get(ctx context.Context, id string) (Message, error)
    List(ctx context.Context, sessionID string) ([]Message, error)
    Delete(ctx context.Context, id string) error
    DeleteSessionMessages(ctx context.Context, sessionID string) error
}

type Message struct {
    ID           string        `json:"id"`
    SessionID    string        `json:"sessionId"`
    Role         Role          `json:"role"`
    Content      []ContentPart `json:"content"`
    TokenCount   int           `json:"tokenCount"`
    FinishReason *string       `json:"finishReason,omitempty"`
    CreatedAt    time.Time     `json:"createdAt"`
    UpdatedAt    time.Time     `json:"updatedAt"`
}

type Role string
const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleSystem    Role = "system" 
)
```

### Content Types

```go
type ContentPart interface {
    isPart()
}

type TextContent struct {
    Text string `json:"text"`
}

type ImageURLContent struct {
    ImageURL ImageURL `json:"image_url"`
}

type ToolCall struct {
    ID       string `json:"id"`
    Type     string `json:"type"`
    Function struct {
        Name      string `json:"name"`
        Arguments string `json:"arguments"`
    } `json:"function"`
}

type ToolResult struct {
    ToolCallID string `json:"tool_call_id"`
    Content    string `json:"content"`
    Success    bool   `json:"success"`
}
```

## AI Agent API

### Agent Service Interface

```go
type Service interface {
    pubsub.Suscriber[AgentEvent]
    Model() models.Model
    Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
    Cancel(sessionID string)
    IsSessionBusy(sessionID string) bool
    IsBusy() bool
    Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error)
    Summarize(ctx context.Context, sessionID string) error
}

type AgentEvent struct {
    Type    AgentEventType
    Message message.Message
    Error   error

    // When summarizing
    SessionID string
    Progress  string
    Done      bool
}

type AgentEventType string
const (
    AgentEventTypeError     AgentEventType = "error"
    AgentEventTypeResponse  AgentEventType = "response"
    AgentEventTypeSummarize AgentEventType = "summarize"
)
```

**Usage Example:**
```go
events, err := agentService.Run(ctx, sessionID, "Help me debug this function", attachments...)
if err != nil {
    return err
}

for event := range events {
    switch event.Type {
    case AgentEventContent:
        fmt.Print(event.Content)
    case AgentEventToolCall:
        fmt.Printf("Executing tool: %s\n", event.ToolCall.Function.Name)
    case AgentEventError:
        fmt.Printf("Error: %v\n", event.Error)
    case AgentEventFinish:
        fmt.Println("Agent finished")
        return
    }
}
```

## LLM Provider API

### Provider Interface

```go
type Provider interface {
    SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error)
    StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent
    Model() models.Model
}

type GenerationParams struct {
    Model            models.ModelID    `json:"model"`
    Messages         []ProviderMessage `json:"messages"`
    MaxTokens        int               `json:"max_tokens,omitempty"`
    Temperature      float64           `json:"temperature,omitempty"`
    Tools            []ToolDefinition  `json:"tools,omitempty"`
    ReasoningEffort  string            `json:"reasoning_effort,omitempty"`
}

type ProviderEvent struct {
    Type EventType

    Content  string
    Thinking string
    Response *ProviderResponse
    ToolCall *message.ToolCall
    Error    error
}

type EventType string
const (
    EventContentStart  EventType = "content_start"
    EventToolUseStart  EventType = "tool_use_start"
    EventToolUseDelta  EventType = "tool_use_delta"
    EventToolUseStop   EventType = "tool_use_stop"
    EventContentDelta  EventType = "content_delta"
    EventThinkingDelta EventType = "thinking_delta"
    EventContentStop   EventType = "content_stop"
    EventComplete      EventType = "complete"
    EventError         EventType = "error"
    EventWarning       EventType = "warning"
)
```

### Supported Providers

```go
type ModelProvider string
const (
    ProviderOpenAI     ModelProvider = "openai"
    ProviderAnthropic  ModelProvider = "anthropic"
    ProviderGemini     ModelProvider = "gemini"
    ProviderGroq       ModelProvider = "groq"
    ProviderBedrock    ModelProvider = "bedrock"
    ProviderAzure      ModelProvider = "azure"
    ProviderVertexAI   ModelProvider = "vertexai"
    ProviderOpenRouter ModelProvider = "openrouter"
    ProviderCopilot    ModelProvider = "copilot"
    ProviderLocal      ModelProvider = "local"
    ProviderXAI        ModelProvider = "xai"
)
```

## Tool System API

### Tool Interface

```go
type BaseTool interface {
    Info() ToolInfo
    Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

type ToolInfo struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
    Required    []string               `json:"required"`
}

type ToolCall struct {
    ID         string                 `json:"id"`
    Name       string                 `json:"name"`
    Parameters map[string]interface{} `json:"parameters"`
}

type ToolResponse struct {
    Type     toolResponseType `json:"type"`
    Content  string           `json:"content"`
    Metadata string           `json:"metadata,omitempty"`
    IsError  bool             `json:"is_error"`
}
```

### Built-in Tools

#### File Operations

```go
// View tool - read file contents
type ViewTool struct{}
func (t *ViewTool) Info() ToolInfo {
    return ToolInfo{
        Name:        "view",
        Description: "View the contents of a file",
        Parameters: map[string]interface{}{
            "file_path": map[string]interface{}{
                "type":        "string",
                "description": "Path to the file to view",
            },
            "offset": map[string]interface{}{
                "type":        "integer",
                "description": "Line offset to start reading from (0-based)",
            },
            "limit": map[string]interface{}{
                "type":        "integer", 
                "description": "Maximum number of lines to read",
            },
        },
        Required: []string{"file_path"},
    }
}

// Write tool - write to files
type WriteTool struct{}
func (t *WriteTool) Info() ToolInfo {
    return ToolInfo{
        Name:        "write",
        Description: "Write content to a file",
        Parameters: map[string]interface{}{
            "file_path": map[string]interface{}{
                "type":        "string",
                "description": "Path to the file to write",
            },
            "content": map[string]interface{}{
                "type":        "string",
                "description": "Content to write to the file",
            },
        },
        Required: []string{"file_path", "content"},
    }
}
```

#### Directory Operations

```go
// List tool - directory listing
type LsTool struct{}
func (t *LsTool) Info() ToolInfo {
    return ToolInfo{
        Name:        "ls",
        Description: "List directory contents",
        Parameters: map[string]interface{}{
            "path": map[string]interface{}{
                "type":        "string",
                "description": "Directory path to list (defaults to current directory)",
            },
            "ignore": map[string]interface{}{
                "type":        "array",
                "description": "Patterns to ignore",
                "items": map[string]interface{}{
                    "type": "string",
                },
            },
        },
        Required: []string{},
    }
}

// Glob tool - pattern matching
type GlobTool struct{}
func (t *GlobTool) Info() ToolInfo {
    return ToolInfo{
        Name:        "glob",
        Description: "Find files matching a pattern",
        Parameters: map[string]interface{}{
            "pattern": map[string]interface{}{
                "type":        "string",
                "description": "Glob pattern to match files (e.g., '*.go', '**/*.js')",
            },
            "path": map[string]interface{}{
                "type":        "string",
                "description": "Base path to search from (defaults to current directory)",
            },
        },
        Required: []string{"pattern"},
    }
}
```

#### Shell Operations

```go
// Bash tool - command execution
type BashTool struct{}
func (t *BashTool) Info() ToolInfo {
    return ToolInfo{
        Name:        "bash",
        Description: "Execute a bash command",
        Parameters: map[string]interface{}{
            "command": map[string]interface{}{
                "type":        "string",
                "description": "The bash command to execute",
            },
            "timeout": map[string]interface{}{
                "type":        "integer",
                "description": "Timeout in seconds (default: 30)",
            },
        },
        Required: []string{"command"},
    }
}
```

## LSP Integration API

### LSP Client Interface

```go
type Client struct {
    serverType   ServerType
    command      string
    args         []string
    conn         *Connection
    capabilities ServerCapabilities
    openFiles    map[string]*OpenFile
    diagnostics  map[string][]Diagnostic
}

func NewClient(language string, config LSPConfig) (*Client, error)
func (c *Client) Initialize(ctx context.Context, rootPath string) error
func (c *Client) OpenFile(ctx context.Context, filePath string) error
func (c *Client) CloseFile(ctx context.Context, filePath string) error
func (c *Client) NotifyFileChange(ctx context.Context, filePath, content string) error
func (c *Client) GetDiagnostics(ctx context.Context, filePath string) ([]Diagnostic, error)
func (c *Client) Shutdown(ctx context.Context) error
```

### LSP Types

```go
type Diagnostic struct {
    Range    Range    `json:"range"`
    Severity int      `json:"severity"`
    Message  string   `json:"message"`
    Source   string   `json:"source,omitempty"`
    Code     any      `json:"code,omitempty"`
}

type Range struct {
    Start Position `json:"start"`
    End   Position `json:"end"`
}

type Position struct {
    Line      int `json:"line"`      // 0-based
    Character int `json:"character"` // 0-based
}

type ServerCapabilities struct {
    TextDocumentSync          int  `json:"textDocumentSync"`
    CompletionProvider        bool `json:"completionProvider"`
    HoverProvider            bool `json:"hoverProvider"`
    SignatureHelpProvider    bool `json:"signatureHelpProvider"`
    DefinitionProvider       bool `json:"definitionProvider"`
    DiagnosticProvider       bool `json:"diagnosticProvider"`
}
```

## Permission System API

### Permission Service Interface

```go
type Service interface {
    RequestPermission(ctx context.Context, req Request) (bool, error)
    HasPermission(ctx context.Context, tool string) bool
    GrantSessionPermission(ctx context.Context, tool string)
    RevokePermission(ctx context.Context, tool string)
    Subscribe(ctx context.Context) <-chan pubsub.Event[PermissionEvent]
}

type Request struct {
    Tool        string `json:"tool"`
    Description string `json:"description"`
    Sensitive   bool   `json:"sensitive"`
    Parameters  any    `json:"parameters,omitempty"`
}

type PermissionEvent struct {
    Type        PermissionEventType `json:"type"`
    Tool        string              `json:"tool"`
    Granted     bool                `json:"granted"`
    SessionOnly bool                `json:"sessionOnly"`
}

type PermissionEventType string
const (
    PermissionRequested PermissionEventType = "requested"
    PermissionGranted   PermissionEventType = "granted"
    PermissionDenied    PermissionEventType = "denied"
)
```

## Database API

### Generated SQLC Queries

```go
type Queries struct {
    db DBTX
}

// Session operations
func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error)
func (q *Queries) GetSession(ctx context.Context, id string) (Session, error)
func (q *Queries) ListSessions(ctx context.Context, arg ListSessionsParams) ([]Session, error)
func (q *Queries) UpdateSession(ctx context.Context, arg UpdateSessionParams) error
func (q *Queries) DeleteSession(ctx context.Context, id string) error

// Message operations  
func (q *Queries) CreateMessage(ctx context.Context, arg CreateMessageParams) (Message, error)
func (q *Queries) GetMessage(ctx context.Context, id string) (Message, error)
func (q *Queries) ListMessages(ctx context.Context, arg ListMessagesParams) ([]Message, error)
func (q *Queries) UpdateMessage(ctx context.Context, arg UpdateMessageParams) error
func (q *Queries) DeleteMessage(ctx context.Context, id string) error

// File operations
func (q *Queries) CreateFile(ctx context.Context, arg CreateFileParams) (File, error)
func (q *Queries) GetFile(ctx context.Context, path string) (File, error)
func (q *Queries) ListFiles(ctx context.Context) ([]File, error)
func (q *Queries) UpdateFile(ctx context.Context, arg UpdateFileParams) error
```

### Database Models

```go
type Session struct {
    ID            string    `json:"id"`
    Title         string    `json:"title"`
    ParentID      *string   `json:"parent_id"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
    ModelProvider string    `json:"model_provider"`
    ModelID       string    `json:"model_id"`
    TotalTokens   int32     `json:"total_tokens"`
    InputTokens   int32     `json:"input_tokens"`
    OutputTokens  int32     `json:"output_tokens"`
    TotalCost     float64   `json:"total_cost"`
}

type Message struct {
    ID           string    `json:"id"`
    SessionID    string    `json:"session_id"`
    Role         string    `json:"role"`
    Content      string    `json:"content"`      // JSON serialized ContentPart[]
    TokenCount   int32     `json:"token_count"`
    FinishReason *string   `json:"finish_reason"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type File struct {
    Path      string    `json:"path"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

## Event System API

### PubSub Interface

```go
type Event[T any] struct {
    Type EventType `json:"type"`
    Data T         `json:"data"`
}

type Broker[T any] struct {
    subscribers map[string]chan Event[T]
    mu          sync.RWMutex
}

func NewBroker[T any]() *Broker[T]
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T]
func (b *Broker[T]) Publish(event Event[T])
func (b *Broker[T]) Close()
```

### Event Types

```go
// Session events
type SessionEvent struct {
    SessionID string       `json:"sessionId"`
    Action    SessionAction `json:"action"`
    Session   *Session     `json:"session,omitempty"`
}

// Message events
type MessageEvent struct {
    MessageID string        `json:"messageId"`
    SessionID string        `json:"sessionId"`
    Action    MessageAction `json:"action"`
    Message   *Message      `json:"message,omitempty"`
}

// Agent events
type AgentEvent struct {
    SessionID  string         `json:"sessionId"`
    Type       AgentEventType `json:"type"`
    Content    string         `json:"content,omitempty"`
    ToolCall   *ToolCall     `json:"toolCall,omitempty"`
    ToolResult *ToolResult   `json:"toolResult,omitempty"`
    Error      error         `json:"error,omitempty"`
}
```

## TUI Component API

### Base Component Interface

```go
type Component interface {
    tea.Model
    SetSize(width, height int)
    Focus()
    Blur()
    Focused() bool
}

type BaseComponent struct {
    width   int
    height  int
    focused bool
    theme   theme.Theme
}

func (c *BaseComponent) SetSize(width, height int) {
    c.width, c.height = width, height
}

func (c *BaseComponent) Focus() {
    c.focused = true
}

func (c *BaseComponent) Blur() {
    c.focused = false
}

func (c *BaseComponent) Focused() bool {
    return c.focused
}
```

### Theme Interface

```go
type Theme interface {
    Name() string
    Colors() ThemeColors
}

type ThemeColors struct {
    Background    lipgloss.Color
    Text          lipgloss.Color
    Primary       lipgloss.Color
    Secondary     lipgloss.Color
    Success       lipgloss.Color
    Warning       lipgloss.Color
    Error         lipgloss.Color
    Border        lipgloss.Color
    BorderFocused lipgloss.Color
    Highlight     lipgloss.Color
    Muted         lipgloss.Color
}
```

## Error Types

### Common Errors

```go
var (
    ErrNotFound          = errors.New("not found")
    ErrInvalidInput      = errors.New("invalid input")
    ErrPermissionDenied  = errors.New("permission denied")
    ErrProviderNotFound  = errors.New("provider not found")
    ErrModelNotSupported = errors.New("model not supported")
    ErrToolNotFound      = errors.New("tool not found")
    ErrSessionNotFound   = errors.New("session not found")
    ErrMessageNotFound   = errors.New("message not found")
)

// Wrapped errors provide context
type ProviderError struct {
    Provider models.ModelProvider
    Err      error
}

func (e *ProviderError) Error() string {
    return fmt.Sprintf("provider %s: %v", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error {
    return e.Err
}
```

## Usage Examples

### Complete Agent Interaction

```go
func ExampleAgentInteraction() error {
    // Setup
    ctx := context.Background()
    db, err := db.Connect()
    if err != nil {
        return err
    }
    defer db.Close()

    app, err := app.New(ctx, db)
    if err != nil {
        return err
    }
    defer app.Shutdown()

    // Create session
    session, err := app.Sessions.CreateSession(ctx, "API Example", nil)
    if err != nil {
        return err
    }

    // Run agent
    events, err := app.CoderAgent.Run(ctx, session.ID, "Explain Go interfaces")
    if err != nil {
        return err
    }

    // Process events
    for event := range events {
        switch event.Type {
        case AgentEventContent:
            fmt.Print(event.Content)
        case AgentEventToolCall:
            fmt.Printf("\n[Tool: %s]\n", event.ToolCall.Function.Name)
        case AgentEventFinish:
            fmt.Println("\n[Finished]")
            return nil
        case AgentEventError:
            return event.Error
        }
    }

    return nil
}
```

### Custom Tool Implementation

```go
func ExampleCustomTool() {
    type CustomTool struct {
        permissions permission.Service
    }

    func (t *CustomTool) Info() ToolInfo {
        return ToolInfo{
            Name:        "custom_tool",
            Description: "A custom tool example",
            Parameters: map[string]interface{}{
                "input": map[string]interface{}{
                    "type":        "string",
                    "description": "Input parameter",
                },
            },
            Required: []string{"input"},
        }
    }

    func (t *CustomTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
        // Permission check
        permitted, err := t.permissions.RequestPermission(ctx, permission.Request{
            Tool:        "custom_tool",
            Description: "Execute custom operation",
            Sensitive:   false,
        })
        if err != nil {
            return ToolResponse{}, err
        }
        if !permitted {
            return ToolResponse{}, ErrPermissionDenied
        }

        // Extract parameters
        input, ok := params.Parameters["input"].(string)
        if !ok {
            return ToolResponse{}, fmt.Errorf("input must be a string")
        }

        // Tool logic
        result := fmt.Sprintf("Processed: %s", input)

        return ToolResponse{
            Success: true,
            Content: result,
        }, nil
    }
}
```

This API reference provides the foundation for understanding and extending OpenCode's capabilities. Each interface is designed for extensibility while maintaining type safety and clear contracts.
