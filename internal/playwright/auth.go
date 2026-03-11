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
	cfg, _ := config.Load()
	home, _ := os.UserHomeDir()
	profileDir := filepath.Join(home, ".super-agent", "browser-profile")

	if cfg != nil && cfg.Facebook.BrowserProfilePath != "" {
		profileDir = expandTilde(cfg.Facebook.BrowserProfilePath)
		fmt.Printf("📂 Using custom browser profile: %s\n", profileDir)
	}
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	fmt.Println("🚀 Starting Browser for Facebook Authentication...")

	err := playwright.Install(&playwright.RunOptions{
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

	// Launch persistent context
	context, err := pw.Chromium.LaunchPersistentContext(profileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless:  playwright.Bool(false),
		Viewport:  &playwright.Size{Width: 1280, Height: 720},
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer context.Close()

	pages := context.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = context.NewPage()
		if err != nil {
			return fmt.Errorf("could not create page: %w", err)
		}
	}

	fmt.Println("🌐 Navigating to Facebook. Please log in if you haven't already.")
	fmt.Println("   *** DO NOT close the browser window. Waiting for session... ***")

	if _, err := page.Goto("https://www.facebook.com/?sk=pages"); err != nil {
		return fmt.Errorf("could not navigate to Facebook: %w", err)
	}

	// We wait until we detect they are logged in.
	fmt.Println("⏳ Waiting for successful login (timeout in 5 minutes)....")

	// Wait for an element that only appears when logged in.
	_, err = page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(300000), // 5 minutes
	})
	if err != nil {
		return fmt.Errorf("login timed out or failed to detect navigation bar: %w", err)
	}

	// Give it an extra few seconds to make sure cookies drop
	time.Sleep(3 * time.Second)

	fmt.Println("✅ Login detected! Your session is securely saved in the browser profile.")
	fmt.Println("🔒 Super-Agent can now post on your behalf.")
	return nil
}

// LoginWithCredentials automates Facebook login using credentials from config.
// It handles both the full login page AND inline modal popups.
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

	return fillLoginForm(page, email, password)
}

// LoginViaModal fills credentials into an inline Facebook login modal/popup
// that appears on the same page (e.g. private group posts showing "See more on Facebook").
func LoginViaModal(page playwright.Page, cfg *config.Config) error {
	email := cfg.Facebook.Email
	password := cfg.Facebook.Password

	if email == "" || password == "" {
		return fmt.Errorf("no Facebook credentials in config. Run 'super-agent setting' to set email/password, or run 'super-agent auth facebook' for manual login")
	}

	fmt.Println("🔐 Logging in via Facebook popup modal...")

	return fillLoginForm(page, email, password)
}

// fillLoginForm fills email and password into any visible Facebook login form
// (works for both full login page and inline modals).
func fillLoginForm(page playwright.Page, email, password string) error {
	// Fill email — try multiple selectors for both modal and full-page login
	emailSelectors := []string{
		`input[name="email"]`,
		`input#email`,
		`input[type="email"]`,
		`input[aria-label="Email address or phone number"]`,
		`input[placeholder*="Email"]`,
		`input[placeholder*="email"]`,
		`input[placeholder*="phone"]`,
	}

	var emailFilled bool
	for _, sel := range emailSelectors {
		loc := page.Locator(sel).First()
		if c, _ := loc.Count(); c > 0 {
			if v, _ := loc.IsVisible(); v {
				loc.Click()
				time.Sleep(500 * time.Millisecond)
				loc.Fill(email)
				fmt.Println("   ✅ Email entered")
				emailFilled = true
				break
			}
		}
	}
	if !emailFilled {
		return fmt.Errorf("could not find email input field")
	}

	// Fill password
	passSelectors := []string{
		`input[name="pass"]`,
		`input#pass`,
		`input[type="password"]`,
		`input[aria-label="Password"]`,
		`input[placeholder*="Password"]`,
		`input[placeholder*="password"]`,
	}

	var passFilled bool
	for _, sel := range passSelectors {
		loc := page.Locator(sel).First()
		if c, _ := loc.Count(); c > 0 {
			if v, _ := loc.IsVisible(); v {
				loc.Click()
				time.Sleep(500 * time.Millisecond)
				loc.Fill(password)
				fmt.Println("   ✅ Password entered")
				passFilled = true
				break
			}
		}
	}
	if !passFilled {
		return fmt.Errorf("could not find password input field")
	}

	// Click login button — try multiple selectors
	loginSelectors := []string{
		`button[name="login"]`,
		`button[data-testid="royal_login_button"]`,
		`button#loginbutton`,
		`div[aria-label="Log in"]`,
		`div[aria-label="Log In"]`,
		`button:has-text("Log in")`,
		`button:has-text("Log In")`,
		`input[type="submit"][value*="Log"]`,
	}

	var clicked bool
	for _, sel := range loginSelectors {
		loc := page.Locator(sel).First()
		if c, _ := loc.Count(); c > 0 {
			if v, _ := loc.IsVisible(); v {
				loc.Click()
				fmt.Println("   🔄 Clicking Log In...")
				clicked = true
				break
			}
		}
	}
	if !clicked {
		// Fallback: press Enter
		page.Keyboard().Press("Enter")
		fmt.Println("   🔄 Submitting login form via Enter...")
	}

	// Wait for login to process
	time.Sleep(8 * time.Second)

	currentURL := page.URL()
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
	}

	fmt.Println("✅ Facebook login successful!")
	return nil
}

// EnsureLoggedIn checks if the current page shows a login wall or inline login modal.
// If detected, it auto-logs in using config credentials and navigates back to the target URL.
// It also saves the session state after successful login.
func EnsureLoggedIn(page playwright.Page, context playwright.BrowserContext, targetURL string, cfg *config.Config) error {
	currentURL := page.URL()

	// Case 1: Full redirect to login page
	isRedirectLogin := strings.Contains(currentURL, "/login") ||
		strings.Contains(currentURL, "two_factor") ||
		strings.Contains(currentURL, "checkpoint")

	// Case 2: Inline modal popup ("See more on Facebook" popup with email/password)
	// Facebook shows this on private group posts without changing the URL
	isModalLogin := false
	if !isRedirectLogin {
		// Check for visible password input (strong indicator of login form)
		passInput := page.Locator(`input[type="password"]`).First()
		if c, _ := passInput.Count(); c > 0 {
			if v, _ := passInput.IsVisible(); v {
				isModalLogin = true
			}
		}
	}

	if !isRedirectLogin && !isModalLogin {
		return nil // Already logged in, no login wall
	}

	if isModalLogin {
		fmt.Println("🛑 Facebook login modal detected! Logging in directly...")

		// Fill credentials directly in the modal popup
		if err := LoginViaModal(page, cfg); err != nil {
			return err
		}

		// After modal login, Facebook usually reloads the page or removes the modal
		time.Sleep(5 * time.Second)

		// Save session
		sessionPath := expandTilde(cfg.Facebook.SessionPath)
		sessionDir := filepath.Dir(sessionPath)
		if err := os.MkdirAll(sessionDir, 0755); err == nil {
			if _, err := context.StorageState(sessionPath); err == nil {
				fmt.Println("🔒 Session saved for future use:", sessionPath)
			}
		}

		// Check if we need to re-navigate (modal might have redirected)
		newURL := page.URL()
		if newURL != targetURL && !strings.Contains(newURL, "permalink") && !strings.Contains(newURL, "photo") {
			fmt.Printf("🌐 Navigating back to: %s\n", targetURL)
			if _, err := page.Goto(targetURL); err != nil {
				return fmt.Errorf("could not navigate back to target: %w", err)
			}
			time.Sleep(5 * time.Second)
		}
	} else {
		// Full redirect login
		fmt.Println("🛑 Login redirect detected! Attempting auto-login...")

		if err := LoginWithCredentials(page, cfg); err != nil {
			return err
		}

		// Save session
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
	}

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
