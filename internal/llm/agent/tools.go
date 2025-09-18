package agent

import (
	"context"

	"github.com/opencode-ai/opencode/internal/history"
	"github.com/opencode-ai/opencode/internal/llm/tools"
	"github.com/opencode-ai/opencode/internal/lsp"
	"github.com/opencode-ai/opencode/internal/message"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/session"
)

// CoderAgentTools returns the complete set of tools available to the coder agent.
// This includes file manipulation, code search, LSP integration, and web search capabilities.
func CoderAgentTools(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	lspClients map[string]*lsp.Client,
) []tools.BaseTool {
	ctx := context.Background()

	// Base tools available to all coder agents
	baseTools := []tools.BaseTool{
		tools.NewBashTool(permissions),
		tools.NewEditTool(lspClients, permissions, history),
		tools.NewFetchTool(permissions),
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewSourcegraphTool(),
		tools.NewViewTool(lspClients),
		tools.NewPatchTool(lspClients, permissions, history),
		tools.NewWriteTool(lspClients, permissions, history),
		NewAgentTool(sessions, messages, lspClients),
		&tools.WebSearchTool{}, // Enables web search for compatible providers (e.g., xAI)
	}

	// Add MCP tools if available
	mcpTools := GetMcpTools(ctx, permissions)

	// Add diagnostics tool if LSP clients are configured
	if len(lspClients) > 0 {
		mcpTools = append(mcpTools, tools.NewDiagnosticsTool(lspClients))
	}

	return append(baseTools, mcpTools...)
}

func TaskAgentTools(lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewSourcegraphTool(),
		tools.NewViewTool(lspClients),
	}
}
