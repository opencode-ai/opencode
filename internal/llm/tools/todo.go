package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opencode-ai/opencode/internal/session"
)

type TodoTool struct {
	sessions session.Service
}

func NewTodoTool(sessions session.Service) BaseTool {
	return &TodoTool{
		sessions: sessions,
	}
}

func (t *TodoTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "todo_write",
		Description: "Create or update a TODO list for the current session. The TODO list is a simple newline-delimited list of items starting with checkboxes.",
		Parameters: map[string]any{
			"todos": map[string]any{
				"type":        "array",
				"description": "List of TODO items",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Unique identifier for the TODO item",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The content/description of the TODO item",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "The current status of the TODO item: 'todo' for incomplete, 'in-progress' for currently working on, 'completed' for finished",
							"enum":        []string{"todo", "in-progress", "completed"},
						},
						"priority": map[string]any{
							"type":        "string",
							"description": "The priority level of the TODO item",
							"enum":        []string{"low", "medium", "high"},
						},
					},
					"required": []string{"id", "content", "status", "priority"},
				},
			},
		},
		Required: []string{"todos"},
	}
}

type TodoItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

func (t *TodoTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	sessionID, _ := GetContextValues(ctx)
	if sessionID == "" {
		return NewTextErrorResponse("No session ID found in context"), nil
	}

	// Parse the todos parameter
	var input struct {
		Todos []TodoItem `json:"todos"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to parse todo items: %v. Please ensure you're providing valid JSON with the required fields (id, content, status, priority).", err)), nil
	}

	// Get current session
	sess, err := t.sessions.Get(ctx, sessionID)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to access session data: %v", err)), nil
	}

	// Convert todos to simple checkbox format
	todoLines := make([]string, len(input.Todos))
	for i, todo := range input.Todos {
		checkbox := "- [ ]"
		if todo.Status == "completed" {
			checkbox = "- [x]"
		} else if todo.Status == "in-progress" {
			checkbox = "- [~]"
		}
		
		priorityIndicator := ""
		if todo.Priority == "high" {
			priorityIndicator = " (!)"
		} else if todo.Priority == "medium" {
			priorityIndicator = " (~)"
		}
		
		todoLines[i] = fmt.Sprintf("%s %s%s", checkbox, todo.Content, priorityIndicator)
	}

	// Update session with new todos
	sess.Todos = strings.Join(todoLines, "\n")
	_, err = t.sessions.Save(ctx, sess)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to save todo list: %v", err)), nil
	}

	// Format the response to show the current todo list
	response := fmt.Sprintf("✓ Todo list updated successfully!\n\n%s", sess.Todos)
	
	return NewTextResponse(response), nil
}

type TodoReadTool struct {
	sessions session.Service
}

func NewTodoReadTool(sessions session.Service) BaseTool {
	return &TodoReadTool{
		sessions: sessions,
	}
}

func (t *TodoReadTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "todo_read",
		Description: "Read the current TODO list for the session",
		Parameters:  map[string]any{},
		Required:    []string{},
	}
}

func (t *TodoReadTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	sessionID, _ := GetContextValues(ctx)
	if sessionID == "" {
		return NewTextErrorResponse("No session ID found in context"), nil
	}

	// Get current session
	sess, err := t.sessions.Get(ctx, sessionID)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to access session data: %v", err)), nil
	}

	if sess.Todos == "" {
		return NewTextResponse("📋 No todo items found for this session. Use todo_write to add new tasks."), nil
	}

	response := fmt.Sprintf("📋 Current Todo List\n\n%s", sess.Todos)
	return NewTextResponse(response), nil
}
