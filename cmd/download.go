package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
	"github.com/zinwaishine/super-agent/internal/playwright"
)

var downloadCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"dl"},
	Short:   "Download images from a Facebook post link",
	Long:    `Downloads all high-quality images from a specific Facebook post URL to a local folder.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		path, _ := cmd.Flags().GetString("path")
		headless, _ := cmd.Flags().GetBool("headless")

		reader := bufio.NewReader(os.Stdin)

		autoPost, _ := cmd.Flags().GetBool("post")

		// Load config for default/previous path
		cfg, _ := config.Load()
		defaultPath := "./downloads/"
		if cfg != nil && cfg.App.ImagePath != "" {
			defaultPath = cfg.App.ImagePath
		}

		// 1. Interactive Path Selection
		if path == "./downloads" { // default value passed by flag
			if autoPost {
				// Skip prompt, use default automatically
				path = defaultPath
			} else {
				fmt.Println("\n📂 Step 1: Destination Folder")
				fmt.Println("   Where should the images be saved on your Mac?")
				fmt.Printf("👉 Enter path (Press ENTER for default %s): ", defaultPath)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)
				if input != "" {
					path = input
				} else {
					path = defaultPath
				}
			}

			// Save the path for next time if it changed
			if cfg != nil && path != cfg.App.ImagePath {
				cfg.App.ImagePath = path
				_ = config.Save(cfg)
			}
		}

		// 2. Interactive URL Selection
		if url == "" {
			fmt.Println("\n🔗 Step 2: Facebook Post Link")
			fmt.Println("   Paste the full URL of the post you want to scrape.")
			fmt.Print("👉 Link: ")
			input, _ := reader.ReadString('\n')
			url = strings.TrimSpace(input)
		}

		if url == "" {
			return fmt.Errorf("no URL provided. Download cancelled")
		}

		fmt.Printf("\n🚀 Starting download from: %s\n", url)
		fmt.Printf("📂 Savings to: %s\n", path)

		err := playwright.DownloadImages(playwright.DownloadOptions{
			URL:      url,
			SavePath: path,
			Headless: headless,
		})
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		if autoPost {
			fmt.Println("\n=============================================")
			fmt.Println("🚀 AUTO-POST ACTIVATED")
			fmt.Println("=============================================")
			return RunAutoPostFlow(path, headless, false)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringP("url", "u", "", "The Facebook post URL to scrape")
	downloadCmd.Flags().StringP("path", "d", "./downloads", "Local folder path to save images")
	downloadCmd.Flags().Bool("headless", true, "Run in headless mode (background)")
	downloadCmd.Flags().BoolP("post", "p", false, "Auto-post downloaded images to Facebook immediately")
}
