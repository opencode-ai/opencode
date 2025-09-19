package cmd

import (
	"fmt"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure opencode CLI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return configure(cmd, args)
	},
}

func configure(cmd *cobra.Command, args []string) error {
	key, _ := cmd.Flags().GetString("key")
	value, _ := cmd.Flags().GetString("value")

	if key == "" || value == "" {
		return fmt.Errorf("key and value are required")
	}

	return config.UpdateProviderAPIKey(models.ModelProvider(key), value)
}

func init() {
	configCmd.Flags().StringP("key", "k", "", "Configuration key")
	configCmd.Flags().StringP("value", "v", "", "Configuration value")
	rootCmd.AddCommand(configCmd)
}
