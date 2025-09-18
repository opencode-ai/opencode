package provider

import (
	"strings"

	"github.com/opencode-ai/opencode/internal/llm/tools"
)

// FilterToolsByProvider filters tools based on provider compatibility.
// If a tool has no provider restrictions (empty Providers field), it's available to all providers.
// Otherwise, it's only available to providers listed in the Providers field.
// The provider name comparison is case-insensitive.
func FilterToolsByProvider(inputTools []tools.BaseTool, providerName string) []tools.BaseTool {
	if len(inputTools) == 0 {
		return nil
	}

	// Pre-allocate slice with capacity to avoid reallocation
	// In most cases, most tools will be available
	filteredTools := make([]tools.BaseTool, 0, len(inputTools))

	for _, tool := range inputTools {
		if isToolAvailableForProvider(tool, providerName) {
			filteredTools = append(filteredTools, tool)
		}
	}

	return filteredTools
}

// isToolAvailableForProvider checks if a tool is available for the given provider.
// Returns true if the tool has no provider restrictions or if the provider is in the allowed list.
// The comparison is case-insensitive to handle variations in provider name casing.
func isToolAvailableForProvider(tool tools.BaseTool, providerName string) bool {
	info := tool.Info()

	// If no providers specified, tool is universally available
	if len(info.Providers) == 0 {
		return true
	}

	// Check if this provider is in the allowed list (case-insensitive)
	for _, allowedProvider := range info.Providers {
		if strings.EqualFold(allowedProvider, providerName) {
			return true
		}
	}

	return false
}
