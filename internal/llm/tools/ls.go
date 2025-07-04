package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
)

type LSParams struct {
	Path     string   `json:"path"`
	MaxDepth int      `json:"max_depth"`
	Ignore   []string `json:"ignore"`
}

type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "directory"
	Children []*TreeNode `json:"children,omitempty"`
}

type LSResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

type lsTool struct{}

const (
	LSToolName    = "ls"
	MaxLSFiles    = 1000
	lsDescription = `Directory listing tool that shows files and subdirectories in a tree structure, helping you explore and understand the project organization.

WHEN TO USE THIS TOOL:
- Use when you need to explore the structure of a directory
- Helpful for understanding the organization of a project
- Good first step when getting familiar with a new codebase

HOW TO USE:
- Provide a path to list (defaults to current working directory)
- Optionally specify glob patterns to ignore
- Results are displayed in a tree structure

FEATURES:
- Displays a hierarchical view of files and directories
- Automatically skips hidden files/directories (starting with '.')
- Skips common system directories like __pycache__
- Can filter out files matching specific patterns

LIMITATIONS:
- Results are limited to 1000 files
- Very large directories will be truncated
- Does not show file sizes or permissions
- Cannot recursively list all directories in a large project

TIPS:
- Use Glob tool for finding files by name patterns instead of browsing
- Use Grep tool for searching file contents
- Combine with other tools for more effective exploration`
)

func NewLsTool() *lsTool {
	return &lsTool{}
}

func (l *lsTool) Info() ToolInfo {
	return ToolInfo{
		Name:        LSToolName,
		Description: lsDescription,
		Parameters: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the directory to list (defaults to current working directory)",
			},
			"max_depth": map[string]any{
				// json schema integer: https://json-schema.org/understanding-json-schema/reference/numeric
				"type":        "integer",
				"description": "Limit the maximum depth to traversing depths(defaults to no limit, 0 or negative values also treated as no limit)",
			},
			"ignore": map[string]any{
				"type":        "array",
				"description": "List of glob patterns to ignore",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		Required: []string{"path"},
	}
}

func (l *lsTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params LSParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	return l.RunWithParams(params)
}

func (l *lsTool) RunWithParams(params LSParams) (ToolResponse, error) {
	searchPath := params.Path
	if searchPath == "" {
		searchPath = config.WorkingDirectory()
	}

	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(config.WorkingDirectory(), searchPath)
	}

	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return NewTextErrorResponse(fmt.Sprintf("path does not exist: %s", searchPath)), nil
	}

	files, truncated, err := listDirectory(searchPath, params.Ignore, MaxLSFiles, params.MaxDepth)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error listing directory: %w", err)
	}

	tree := createFileTree(files)
	output := printTree(tree, searchPath)

	if truncated {
		output = fmt.Sprintf("There are more than %d files in the directory. Use a more specific path or use the Glob tool to find specific files. The first %d files and directories are included below:\n\n%s", MaxLSFiles, MaxLSFiles, output)
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		LSResponseMetadata{
			NumberOfFiles: len(files),
			Truncated:     truncated,
		},
	), nil
}

// shouldSkipDirDueToDepth checks if a directory should be skipped due to depth limits
// Returns true if the directory should be skipped, false otherwise
func shouldSkipDirDueToDepth(initialPath, currentPath string, maxDepth int) bool {
	if maxDepth <= 0 {
		return false
	}

	relPath, err := filepath.Rel(initialPath, currentPath)
	if err != nil {
		return false // Don't skip on error
	}

	// Calculate depth by counting directory separators
	depth := 0
	if relPath != "." {
		// If there are no separators, it's still depth 1 (direct child)
		// If there are separators, count them and add 1
		depth = strings.Count(relPath, string(filepath.Separator)) + 1
	}

	// Skip directory traversal if we're at or beyond max depth
	return depth >= maxDepth
}

func listDirectory(initialPath string, ignorePatterns []string, limit int, maxDepth int) ([]string, bool, error) {
	var results []string
	truncated := false

	err := filepath.Walk(initialPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we don't have permission to access
		}

		if shouldSkip(path, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we should skip due to depth limits (only for directories)
		if info.IsDir() && shouldSkipDirDueToDepth(initialPath, path, maxDepth) {
			// Still include this directory in results if it's not the initial path
			if path != initialPath {
				path = path + string(filepath.Separator)
				results = append(results, path)
			}
			return filepath.SkipDir
		}

		if path != initialPath {
			if info.IsDir() {
				path = path + string(filepath.Separator)
			}
			results = append(results, path)
		}

		if len(results) >= limit {
			truncated = true
			return filepath.SkipAll
		}

		return nil
	})
	if err != nil {
		return nil, truncated, err
	}

	return results, truncated, nil
}

func shouldSkip(path string, ignorePatterns []string) bool {
	base := filepath.Base(path)

	if base != "." && strings.HasPrefix(base, ".") {
		return true
	}

	commonIgnored := []string{
		"__pycache__",
		"node_modules",
		"dist",
		"build",
		"target",
		"vendor",
		"bin",
		"obj",
		".git",
		".idea",
		".vscode",
		".DS_Store",
		"*.pyc",
		"*.pyo",
		"*.pyd",
		"*.so",
		"*.dll",
		"*.exe",
	}

	if strings.Contains(path, filepath.Join("__pycache__", "")) {
		return true
	}

	for _, ignored := range commonIgnored {
		if strings.HasSuffix(ignored, "/") {
			if strings.Contains(path, filepath.Join(ignored[:len(ignored)-1], "")) {
				return true
			}
		} else if strings.HasPrefix(ignored, "*.") {
			if strings.HasSuffix(base, ignored[1:]) {
				return true
			}
		} else {
			if base == ignored {
				return true
			}
		}
	}

	for _, pattern := range ignorePatterns {
		matched, err := filepath.Match(pattern, base)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func createFileTree(sortedPaths []string) []*TreeNode {
	root := []*TreeNode{}
	pathMap := make(map[string]*TreeNode)

	for _, path := range sortedPaths {
		parts := strings.Split(path, string(filepath.Separator))
		currentPath := ""
		var parentPath string

		var cleanParts []string
		for _, part := range parts {
			if part != "" {
				cleanParts = append(cleanParts, part)
			}
		}
		parts = cleanParts

		if len(parts) == 0 {
			continue
		}

		for i, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = filepath.Join(currentPath, part)
			}

			if _, exists := pathMap[currentPath]; exists {
				parentPath = currentPath
				continue
			}

			isLastPart := i == len(parts)-1
			isDir := !isLastPart || strings.HasSuffix(path, string(filepath.Separator))
			nodeType := "file"
			if isDir {
				nodeType = "directory"
			}
			newNode := &TreeNode{
				Name:     part,
				Path:     currentPath,
				Type:     nodeType,
				Children: []*TreeNode{},
			}

			pathMap[currentPath] = newNode

			if i > 0 && parentPath != "" {
				if parent, ok := pathMap[parentPath]; ok {
					parent.Children = append(parent.Children, newNode)
				}
			} else {
				root = append(root, newNode)
			}

			parentPath = currentPath
		}
	}

	return root
}

func printTree(tree []*TreeNode, rootPath string) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("- %s%s\n", rootPath, string(filepath.Separator)))

	for _, node := range tree {
		printNode(&result, node, 1)
	}

	return result.String()
}

func printNode(builder *strings.Builder, node *TreeNode, level int) {
	indent := strings.Repeat("  ", level)

	nodeName := node.Name
	if node.Type == "directory" {
		nodeName += string(filepath.Separator)
	}

	fmt.Fprintf(builder, "%s- %s\n", indent, nodeName)

	if node.Type == "directory" && len(node.Children) > 0 {
		for _, child := range node.Children {
			printNode(builder, child, level+1)
		}
	}
}
