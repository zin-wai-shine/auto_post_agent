package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/playwright"
)

var authCmd = &cobra.Command{
	Use:   "auth [provider]",
	Short: "Authenticate with a target platform (e.g. facebook)",
	Long: `Connect your system to an external platform. Currently supports:
	
  - facebook: Opens a browser window letting you log into your Facebook Page.
              Saves the session securely so Super-Agent can auto-post for you.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		if provider != "facebook" {
			return fmt.Errorf("unsupported provider: %s. Currently only 'facebook' is supported", provider)
		}

		fmt.Println("🔗 Connecting to Facebook...")

		err := playwright.Authenticate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Authentication failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n🎉 Success! Your Facebook Page is now connected to Super-Agent.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
