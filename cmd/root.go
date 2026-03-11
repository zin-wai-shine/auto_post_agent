package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "super-agent",
	Short: "🏠 Super-Agent CLI — Real Estate Listing-to-Social Pipeline",
	Long: `Super-Agent is a high-performance CLI tool that automates the entire
"Listing-to-Social" pipeline for real estate professionals.

Connect your existing website database, prep images with AI,
generate trilingual content, and post to multiple Facebook Pages
— all from a single command.

Features:
  • Universal Database Connector with vector embeddings (pgvector)
  • AI-powered image prep (resize, watermark, hero-shot selection)
  • Trilingual content generation (Thai, English, Myanmar)
  • Automated Facebook Marketplace & Pages posting
  • Natural Language AI Search for your website`,
	Version: "0.1.0",
}

var helpAliasCmd = &cobra.Command{
	Use:    "h",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		newArgs := append([]string{"help"}, args...)
		rootCmd.SetArgs(newArgs)
		rootCmd.Execute()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(helpAliasCmd)
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.super-agent.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
}
