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

	// Use a persistent browser profile so login is remembered between runs
	home, _ := os.UserHomeDir()
	profileDir := filepath.Join(home, ".super-agent", "browser-profile")

	// Load configuration to check for custom profile path
	cfg, _ := config.Load()
	if cfg != nil && cfg.Facebook.BrowserProfilePath != "" {
		profileDir = expandTilde(cfg.Facebook.BrowserProfilePath)
		fmt.Printf("📂 Using custom browser profile: %s\n", profileDir)
	}

	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("could not create browser profile directory: %w", err)
	}

	fmt.Println("⚙️  Launching browser (visible mode required for Facebook)...")
	context, err := pw.Chromium.LaunchPersistentContext(profileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless:  playwright.Bool(false),
		Args:      []string{"--start-minimized"},
		Viewport:  &playwright.Size{Width: 1920, Height: 1080},
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer context.Close()

	// Use the first page or create a new one
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

	// Check if we need to login first by navigating to Facebook
	fmt.Println("🔍 Checking Facebook login status...")
	if _, err := page.Goto("https://www.facebook.com/"); err != nil {
		return fmt.Errorf("could not navigate to Facebook: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Check if already logged in (navigation bar = logged in)
	navBar := page.Locator(`div[role="navigation"]`).First()
	navCount, _ := navBar.Count()
	isLoggedIn := navCount > 0

	if !isLoggedIn {
		fmt.Println("═══════════════════════════════════════════════")
		fmt.Println("🔐 Not logged in to Facebook.")
		fmt.Println("   Please log in manually in the browser window.")
		fmt.Println("   Your login will be remembered for all future runs!")
		fmt.Println("═══════════════════════════════════════════════")

		// Wait for user to login (detect navigation bar)
		fmt.Println("⏳ Waiting for you to log in... (timeout: 5 minutes)")
		_, err = page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(300000), // 5 minutes
		})
		if err != nil {
			return fmt.Errorf("login timed out: %w", err)
		}

		time.Sleep(3 * time.Second)
		fmt.Println("✅ Login successful! Your session is saved in the browser profile.")
		fmt.Println("   You won't need to login again! 🎉")
		fmt.Println("═══════════════════════════════════════════════")
	} else {
		fmt.Println("✅ Already logged in to Facebook!")
	}

	// Now navigate to the target post URL
	fmt.Printf("🌐 Navigating to: %s\n", opts.URL)
	if _, err := page.Goto(opts.URL); err != nil {
		return fmt.Errorf("could not navigate: %w", err)
	}
	time.Sleep(5 * time.Second)

	// RE-CHECK: Facebook often shows an overlay/modal on the specific post page
	// even if the root was "logged in". We check for login indicators again.
	if count, _ := page.Locator(`input[type="password"]`).Count(); count > 0 {
		fmt.Println("═══════════════════════════════════════════════")
		fmt.Println("🛑 Login Wall detected on the Post Page!")
		fmt.Println("   Please finish logging in in the browser window.")
		fmt.Println("   The bot will wait for you...")
		fmt.Println("═══════════════════════════════════════════════")

		_, err = page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(300000),
		})
		if err != nil {
			return fmt.Errorf("login failed or timed out: %w", err)
		}
		fmt.Println("✅ Login successful! Continuing...")
		time.Sleep(3 * time.Second)
	}

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
		// Recovery: If we are failing to find photos, check if we were logged out/kicked to login
		if staleRuns > 3 && (strings.Contains(page.URL(), "login") || strings.Contains(page.URL(), "checkpoint")) {
			fmt.Println("   ⚠️  Detected login wall during discovery. Please log in again...")
			page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
				Timeout: playwright.Float(300000),
			})
			fmt.Println("   ✅ Resuming...")
			staleRuns = 0
		}

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
			fmt.Println("   🛑 Login wall detected. Session may have expired.")
			break
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
