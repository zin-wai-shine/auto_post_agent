package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
	"gopkg.in/yaml.v3"
)

var settingCmd = &cobra.Command{
	Use:     "setting",
	Aliases: []string{"settings", "config"},
	Short:   "Manage your Super-Agent configuration settings",
	Long:    `View, edit, or reset your default configuration settings like Database URL, API Keys, and default paths.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		for {
			fmt.Println("\n⚙️  Super-Agent Settings Menu")
			fmt.Println("   1) 👁️  View Current Settings")
			fmt.Println("   2) ✏️  Edit a Setting")
			fmt.Println("   3) ❌ Exit")
			fmt.Print("👉 Choose an option (1-3): ")

			opt, _ := reader.ReadString('\n')
			opt = strings.TrimSpace(opt)

			switch opt {
			case "1":
				viewSettings()
			case "2":
				editSettings(reader)
			case "3", "exit", "q":
				fmt.Println("👋 Exiting settings.")
				return nil
			default:
				fmt.Println("⚠️  Invalid choice.")
			}
		}
	},
}

func viewSettings() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("\n⚠️  Could not load config: %v\n", err)
		return
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Printf("\n⚠️  Failed to format config: %v\n", err)
		return
	}

	fmt.Println("\n=============================================")
	fmt.Println("📄 CURRENT SETTINGS (" + config.DefaultConfigPath() + ")")
	fmt.Println("=============================================")
	fmt.Println(string(data))
	fmt.Println("=============================================")
}

func editSettings(reader *bufio.Reader) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("\n⚠️  Could not load existing config. A new one will be created.")
		cfg = &config.Config{}
	}

	fmt.Println("\n✏️  Which setting would you like to edit?")
	fmt.Println("   1) Database URL")
	fmt.Println("   2) OpenAI API Key")
	fmt.Println("   3) Claude API Key")
	fmt.Println("   4) Default Image Download/Upload Folder")
	fmt.Println("   5) Default Facebook Page ID")
	fmt.Println("   6) Default LLM Model")
	fmt.Println("   7) Default Business Facebook URL")
	fmt.Println("   8) Browser Profile Path (Target your actual Chrome profile)")
	fmt.Println("   0) Cancel")
	fmt.Print("👉 Choose an option: ")

	opt, _ := reader.ReadString('\n')
	opt = strings.TrimSpace(opt)

	switch opt {
	case "1":
		fmt.Printf("   Current Database URL: %s\n", cfg.Database.URL)
		fmt.Print("   New Database URL (leave blank to keep current): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.Database.URL = input
		}
	case "2":
		fmt.Printf("   Current OpenAI Key: %s\n", maskString(cfg.LLM.OpenAIKey))
		fmt.Print("   New OpenAI Key: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.LLM.OpenAIKey = input
		}
	case "3":
		fmt.Printf("   Current Claude Key: %s\n", maskString(cfg.LLM.ClaudeKey))
		fmt.Print("   New Claude Key: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.LLM.ClaudeKey = input
		}
	case "4":
		fmt.Printf("   Current Image Folder: %s\n", cfg.App.ImagePath)
		fmt.Print("   New Image Folder: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.App.ImagePath = input
		}
	case "5":
		var currentPages string
		if len(cfg.Facebook.PageIDs) > 0 {
			currentPages = strings.Join(cfg.Facebook.PageIDs, ", ")
		}
		fmt.Printf("   Current Facebook Pages: %s\n", currentPages)
		fmt.Print("   New Facebook Page ID (or comma-separated list): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			pages := strings.Split(input, ",")
			for i := range pages {
				pages[i] = strings.TrimSpace(pages[i])
			}
			cfg.Facebook.PageIDs = pages
		}
	case "6":
		fmt.Printf("   Current Default Model: %s\n", cfg.LLM.DefaultModel)
		fmt.Print("   New Default Model (e.g. gpt-4o, claude-3-haiku-20240307): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.LLM.DefaultModel = input
		}
	case "7":
		fmt.Printf("   Current Business Facebook URL: %s\n", cfg.Facebook.BusinessURL)
		fmt.Print("   New Business Facebook URL: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.Facebook.BusinessURL = input
		}
	case "8":
		fmt.Printf("   Current Browser Profile Path: %s\n", cfg.Facebook.BrowserProfilePath)
		fmt.Println("   ⚠️  WARNING: If you point this to your actual Chrome User Data,")
		fmt.Println("      you MUST CLOSE Chrome before running the bot.")
		fmt.Print("   New Browser Profile Path: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.Facebook.BrowserProfilePath = input
		}
	case "0", "":
		return
	default:
		fmt.Println("⚠️  Invalid choice.")
		return
	}

	err = config.Save(cfg)
	if err != nil {
		fmt.Printf("\n❌ Failed to save config: %v\n", err)
	} else {
		fmt.Println("\n✅ Settings saved successfully!")
	}
}

func maskString(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "...." + s[len(s)-4:]
}

func init() {
	rootCmd.AddCommand(settingCmd)
}
