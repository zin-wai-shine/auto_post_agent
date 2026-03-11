package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Super-Agent configuration",
	Long: `Setup the local configuration file for Super-Agent.

This command will interactively prompt you for:
  • PostgreSQL database URL (with pgvector extension)
  • OpenAI / Claude API keys (BYOK — Bring Your Own Key)
  • Facebook session storage path
  • Ollama endpoint (optional, for local inference)

The configuration is stored at ~/.super-agent.yaml by default.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("db-url")
		openaiKey, _ := cmd.Flags().GetString("openai-key")
		claudeKey, _ := cmd.Flags().GetString("claude-key")
		ollamaURL, _ := cmd.Flags().GetString("ollama-url")

		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: dbURL,
			},
			LLM: config.LLMConfig{
				OpenAIKey: openaiKey,
				ClaudeKey: claudeKey,
				OllamaURL: ollamaURL,
			},
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("✅ Super-Agent initialized successfully!")
		fmt.Println("📁 Config saved to ~/.super-agent.yaml")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String("db-url", "", "PostgreSQL connection URL (e.g., postgres://user:pass@localhost:5432/superagent)")
	initCmd.Flags().String("openai-key", "", "OpenAI API key")
	initCmd.Flags().String("claude-key", "", "Claude/Anthropic API key")
	initCmd.Flags().String("ollama-url", "http://localhost:11434", "Ollama API endpoint")
}
