package playwright

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/zinwaishine/super-agent/internal/config"
)

type DownloadOptions struct {
	URL      string
	SavePath string
	Headless bool
}

func DownloadImages(opts DownloadOptions) error {
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	// Facebook detects headless browsers and blocks gallery navigation.
	// ALWAYS use visible mode. Use --start-minimized to hide the window.
	fmt.Println("⚙️  Launching browser (visible mode required for Facebook)...")
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args:     []string{"--start-minimized"},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	cfg, _ := config.Load()
	sessionPath := ""
	if cfg != nil && cfg.Facebook.SessionPath != "" {
		sessionPath = expandTilde(cfg.Facebook.SessionPath)
	} else {
		home, _ := os.UserHomeDir()
		sessionPath = filepath.Join(home, ".super-agent", "fb-session.json")
	}

	// If no saved session exists, login first BEFORE navigating to the post
	if _, err := os.Stat(sessionPath); err != nil {
		fmt.Println("🔐 No saved Facebook session found. Logging in first...")

		// Create a temporary context to login
		loginCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{
			Viewport:  &playwright.Size{Width: 1280, Height: 720},
			UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		})
		if err != nil {
			return fmt.Errorf("could not create login context: %w", err)
		}

		loginPage, err := loginCtx.NewPage()
		if err != nil {
			loginCtx.Close()
			return fmt.Errorf("could not create login page: %w", err)
		}

		if err := LoginWithCredentials(loginPage, cfg); err != nil {
			loginCtx.Close()
			return fmt.Errorf("auto-login failed: %w", err)
		}

		// Save session after successful login
		sessionDir := filepath.Dir(sessionPath)
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			loginCtx.Close()
			return fmt.Errorf("could not create session directory: %w", err)
		}
		if _, err := loginCtx.StorageState(sessionPath); err != nil {
			loginCtx.Close()
			return fmt.Errorf("could not save session: %w", err)
		}
		fmt.Println("🔒 Session saved! You won't need to login again:", sessionPath)
		loginCtx.Close()
	} else {
		fmt.Println("✅ Loaded saved Facebook session. No login needed.")
	}

	// Now create the main context with the saved session loaded
	contextOpts := playwright.BrowserNewContextOptions{
		Viewport:         &playwright.Size{Width: 1920, Height: 1080},
		UserAgent:        playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		StorageStatePath: playwright.String(sessionPath),
	}

	context, err := browser.NewContext(contextOpts)
	if err != nil {
		return fmt.Errorf("could not create context: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}

	fmt.Printf("🌐 Navigating to: %s\n", opts.URL)
	if _, err := page.Goto(opts.URL); err != nil {
		return fmt.Errorf("could not navigate: %w", err)
	}
	time.Sleep(5 * time.Second)

	if err := os.MkdirAll(opts.SavePath, os.ModePerm); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	// PHASE 1: DISCOVER ALL PHOTO URLs VIA THEATER NAVIGATION
	fmt.Println("📜 Phase 1: Discovering all photo URLs via theater gallery...")

	// Open theater mode
	entrySelectors := []string{
		`a[href*="/photo/"] img`,
		`a[href*="/photo.php"] img`,
		`a[href*="/photos/"] img`,
	}
	for _, sel := range entrySelectors {
		loc := page.Locator(sel).First()
		if c, _ := loc.Count(); c > 0 {
			if v, _ := loc.IsVisible(); v {
				fmt.Println("   📸 Opening theater gallery...")
				loc.Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)})
				time.Sleep(5 * time.Second)
				break
			}
		}
	}

	uniqueURLs := make(map[string]bool)
	var urlList []string
	staleRuns := 0
	lastCount := 0

	for i := 0; i < 70; i++ {
		// Safe dialog focus
		if i%5 == 0 {
			page.Evaluate(`(function(){ const d = document.querySelector('div[role="dialog"]'); if(d) d.focus(); })()`)
		}

		currURL := page.URL()
		if !uniqueURLs[currURL] && (strings.Contains(currURL, "/photo") || strings.Contains(currURL, "fbid=")) {
			uniqueURLs[currURL] = true
			urlList = append(urlList, currURL)
			fmt.Printf("   🔍 [%d]: %s\n", len(urlList), currURL)
		}

		// Track stale state
		if len(urlList) == lastCount {
			staleRuns++
		} else {
			staleRuns = 0
			lastCount = len(urlList)
		}

		// Heavy-duty JS click approach
		page.Evaluate(`
			(function() {
				// 1. Force focus
				const d = document.querySelector('div[role="dialog"]');
				if (d) d.focus();

				// 2. Try native Next button
				const next = document.querySelector('div[aria-label="Next photo"]') || 
				             document.querySelector('div[aria-label="Next"]') || 
				             document.querySelector('[aria-label*="Next"]');
				if (next) {
					console.log("Clicking Next...");
					['mousedown', 'mouseup', 'click'].forEach(t => {
						next.dispatchEvent(new MouseEvent(t, {bubbles: true, cancelable: true, view: window}));
					});
				} else {
					console.log("Next button missing. Scrolling right...");
					// 3. Fallback: Scroll the dialog right/down to force React to render
					if (d) {
						const scrolls = Array.from(d.querySelectorAll('*')).filter(el => {
							const s = window.getComputedStyle(el);
							return s.overflowX === 'auto' || s.overflowY === 'auto' || s.overflowX === 'scroll' || s.overflowY === 'scroll';
						});
						scrolls.forEach(s => {
							s.scrollBy(1000, 1000);
						});
					}
				}
			})()
		`)

		page.Keyboard().Press("ArrowRight")
		time.Sleep(2500 * time.Millisecond) // Slightly longer wait for JS execution

		// Wrap-around
		if len(urlList) > 1 && currURL == urlList[0] {
			fmt.Println("   ✅ Wrap-around. Discovery complete.")
			break
		}
		if staleRuns >= 8 {
			fmt.Printf("   🏁 Stale for %d rounds. Stopping.\n", staleRuns)
			break
		}
	}

	// PHASE 2: DOWNLOAD EACH PHOTO
	fmt.Printf("\n🚀 Phase 2: Downloading %d discovered photos...\n", len(urlList))

	downloadedSrcs := make(map[string]bool)
	count := 0

	for idx, fullURL := range urlList {
		fmt.Printf("[%d/%d] 🌐 %s...\n", idx+1, len(urlList), fullURL)

		_, err := page.Goto(fullURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
		if err != nil {
			fmt.Printf("   ❌ %v\n", err)
			continue
		}
		time.Sleep(4 * time.Second)

		if strings.Contains(page.URL(), "login") || strings.Contains(page.URL(), "two_factor") {
			fmt.Println("   🛑 Login wall detected during download. Re-authenticating...")
			if err := EnsureLoggedIn(page, context, fullURL, cfg); err != nil {
				fmt.Printf("   ❌ Auto-login failed: %v\n", err)
				break
			}
			// Re-navigate to the photo after login
			if _, err := page.Goto(fullURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
				fmt.Printf("   ❌ %v\n", err)
				continue
			}
			time.Sleep(4 * time.Second)
		}

		src := ""
		for _, sel := range []string{`img[data-visualcompletion="media-vc-image"]`, `img.x1ey2m1z`} {
			loc := page.Locator(sel).First()
			if c, _ := loc.Count(); c > 0 {
				s, _ := loc.GetAttribute("src")
				if s != "" && strings.Contains(s, "fbcdn") {
					src = s
					break
				}
			}
		}
		if src == "" {
			all, _ := page.Locator("img").All()
			for _, img := range all {
				s, _ := img.GetAttribute("src")
				if strings.Contains(s, "fbcdn") && len(s) > 250 {
					src = s
					break
				}
			}
		}

		if src != "" && !downloadedSrcs[src] {
			fmt.Printf("   📥 %s...\n", src[:60])
			fileName := fmt.Sprintf("property_image_%d.jpg", count+1)
			filePath := filepath.Join(opts.SavePath, fileName)
			if err := downloadFile(src, filePath); err != nil {
				fmt.Printf("   ⚠️ %v\n", err)
			} else {
				fmt.Printf("   ✅ %s\n", fileName)
				downloadedSrcs[src] = true
				count++
			}
		} else if src == "" {
			fmt.Println("   ❌ No source.")
		} else {
			fmt.Println("   ⏭️ Duplicate.")
		}
	}

	if count == 0 {
		fmt.Println("⚠️ No images downloaded.")
	} else {
		fmt.Printf("\n🎉 MISSION COMPLETE: %d images downloaded to %s\n", count, opts.SavePath)
	}
	return nil
}

func downloadFile(url, filePath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
