package playwright

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// LoginWithCredentials automates Facebook login using credentials from config.
// It types email/password into the login form and waits for navigation to confirm login.
func LoginWithCredentials(page playwright.Page, cfg *config.Config) error {
	email := cfg.Facebook.Email
	password := cfg.Facebook.Password

	if email == "" || password == "" {
		return fmt.Errorf("no Facebook credentials in config. Run 'super-agent setting' to set email/password, or run 'super-agent auth facebook' for manual login")
	}

	fmt.Println("🔐 Auto-logging in to Facebook...")

	// Navigate to Facebook login page
	if _, err := page.Goto("https://www.facebook.com/login"); err != nil {
		return fmt.Errorf("could not navigate to login page: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Fill email
	emailInput := page.Locator(`input[name="email"], input#email`)
	if c, _ := emailInput.Count(); c > 0 {
		emailInput.Click()
		time.Sleep(500 * time.Millisecond)
		emailInput.Fill(email)
		fmt.Println("   ✅ Email entered")
	} else {
		return fmt.Errorf("could not find email input field")
	}

	// Fill password
	passInput := page.Locator(`input[name="pass"], input#pass`)
	if c, _ := passInput.Count(); c > 0 {
		passInput.Click()
		time.Sleep(500 * time.Millisecond)
		passInput.Fill(password)
		fmt.Println("   ✅ Password entered")
	} else {
		return fmt.Errorf("could not find password input field")
	}

	// Click login button
	loginBtn := page.Locator(`button[name="login"], button[data-testid="royal_login_button"], button#loginbutton, input[type="submit"]`).First()
	if c, _ := loginBtn.Count(); c > 0 {
		loginBtn.Click()
		fmt.Println("   🔄 Clicking Login...")
	} else {
		// Fallback: press Enter
		page.Keyboard().Press("Enter")
		fmt.Println("   🔄 Submitting login form...")
	}

	// Wait for navigation (successful login redirects away from /login)
	time.Sleep(8 * time.Second)

	currentURL := page.URL()
	if strings.Contains(currentURL, "login") || strings.Contains(currentURL, "checkpoint") {
		// Check for 2FA / checkpoint
		if strings.Contains(currentURL, "checkpoint") || strings.Contains(currentURL, "two_factor") {
			fmt.Println("⚠️  Two-factor authentication detected!")
			fmt.Println("   Please complete 2FA in the browser window...")
			// Wait up to 2 minutes for 2FA
			for i := 0; i < 24; i++ {
				time.Sleep(5 * time.Second)
				url := page.URL()
				if !strings.Contains(url, "checkpoint") && !strings.Contains(url, "two_factor") && !strings.Contains(url, "login") {
					fmt.Println("   ✅ 2FA completed!")
					break
				}
				if i == 23 {
					return fmt.Errorf("2FA timed out after 2 minutes")
				}
			}
		} else {
			return fmt.Errorf("login may have failed — still on login page. Check credentials")
		}
	}

	fmt.Println("✅ Facebook login successful!")
	return nil
}

// EnsureLoggedIn checks if the current page shows a login wall.
// If detected, it auto-logs in using config credentials and navigates back to the target URL.
// It also saves the session state after successful login.
func EnsureLoggedIn(page playwright.Page, context playwright.BrowserContext, targetURL string, cfg *config.Config) error {
	currentURL := page.URL()

	// Check if we hit a login wall
	isLoginWall := strings.Contains(currentURL, "/login") ||
		strings.Contains(currentURL, "two_factor") ||
		strings.Contains(currentURL, "checkpoint")

	// Also check for login form presence on the page (Facebook sometimes shows inline login)
	if !isLoginWall {
		loginForm, _ := page.Locator(`form#login_form, form[action*="login"]`).Count()
		emailField, _ := page.Locator(`input[name="email"]#email`).Count()
		if loginForm > 0 && emailField > 0 {
			isLoginWall = true
		}
	}

	if !isLoginWall {
		return nil // Already logged in
	}

	fmt.Println("🛑 Login wall detected! Attempting auto-login...")

	if err := LoginWithCredentials(page, cfg); err != nil {
		return err
	}

	// Save session after successful login
	sessionPath := expandTilde(cfg.Facebook.SessionPath)
	sessionDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(sessionDir, 0755); err == nil {
		if _, err := context.StorageState(sessionPath); err == nil {
			fmt.Println("🔒 Session saved for future use:", sessionPath)
		}
	}

	// Navigate back to the target URL
	fmt.Printf("🌐 Navigating back to: %s\n", targetURL)
	if _, err := page.Goto(targetURL); err != nil {
		return fmt.Errorf("could not navigate back to target: %w", err)
	}
	time.Sleep(5 * time.Second)

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
