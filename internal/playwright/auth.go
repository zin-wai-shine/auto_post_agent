package playwright

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/zinwaishine/super-agent/internal/config"
)

// Authenticate opens a visible browser for the user to log into Facebook,
// wait for them to log in successfully, and saves their session state to disk.
func Authenticate() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Make sure the directory for the session file exists
	sessionPath := expandTilde(cfg.Facebook.SessionPath)
	sessionDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	fmt.Println("🚀 Starting Browser for Facebook Authentication...")

	err = playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	})
	if err != nil {
		return fmt.Errorf("could not install playwright dependencies: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	// Launch in non-headless mode so the user can see and type
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	// If we already have a session, we can load it to see if they're still logged in.
	// Otherwise, we create a fresh context.
	var context playwright.BrowserContext
	if _, statErr := os.Stat(sessionPath); statErr == nil {
		fmt.Println("♻️  Found existing session, loading it...")
		context, err = browser.NewContext(playwright.BrowserNewContextOptions{
			StorageStatePath: playwright.String(sessionPath),
			Viewport:         &playwright.Size{Width: 1280, Height: 720},
		})
	} else {
		context, err = browser.NewContext(playwright.BrowserNewContextOptions{
			Viewport: &playwright.Size{Width: 1280, Height: 720},
		})
	}

	if err != nil {
		return fmt.Errorf("could not create context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}

	fmt.Println("🌐 Navigating to Facebook. Please log in if you haven't already.")
	fmt.Println("   *** DO NOT close the browser window. Waiting for session... ***")

	if _, err := page.Goto("https://www.facebook.com/?sk=pages"); err != nil {
		return fmt.Errorf("could not navigate to Facebook: %w", err)
	}

	// We wait until we detect they are logged in.
	// A good indicator of being logged in is the presence of the profile picture account switcher or the "Pages" header.
	// We'll give the user 5 minutes to log in.
	fmt.Println("⏳ Waiting for successful login (timeout in 5 minutes)....")

	// Wait for an element that only appears when logged in.
	// We'll wait for the aria-label="Account" or "Profile" or the Facebook nav bar
	_, err = page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(300000), // 5 minutes
	})
	if err != nil {
		return fmt.Errorf("login timed out or failed to detect navigation bar: %w", err)
	}

	// Give it an extra few seconds to make sure cookies drop
	time.Sleep(3 * time.Second)

	fmt.Println("✅ Login detected! Saving your session securely to:", sessionPath)
	if _, err := context.StorageState(sessionPath); err != nil {
		return fmt.Errorf("failed to save session state: %w", err)
	}

	fmt.Println("🔒 Session securely saved. Super-Agent can now post on your behalf.")
	return nil
}

func expandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
