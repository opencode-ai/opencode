package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencode-ai/opencode/internal/tui/util"
	"github.com/stretchr/testify/assert"
)

func TestCreateAgentOsCommands(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "agent_os_commands_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change the current working directory to the temporary directory
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Run the createAgentOsCommands function
	cmd := createAgentOsCommands()
	msg := cmd()

	// Check that the message is not an error
	assert.NotNil(t, msg)
	infoMsg, ok := msg.(util.InfoMsg)
	assert.True(t, ok)
	assert.Equal(t, util.InfoTypeInfo, infoMsg.Type)

	// Check that the commands directory was created
	commandsDir := filepath.Join(tempDir, ".opencode", "commands")
	_, err = os.Stat(commandsDir)
	assert.NoError(t, err)

	// Check that the command files were created
	agentOsCommands := []string{"analyze_product", "create_spec", "execute_tasks", "plan_product"}
	for _, cmd := range agentOsCommands {
		cmdFile := filepath.Join(commandsDir, cmd+".md")
		_, err = os.Stat(cmdFile)
		assert.NoError(t, err)

		// Check the content of the command file
		content, err := os.ReadFile(cmdFile)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("@~/agent_os/instructions/%s.md", cmd), string(content))
	}
}
