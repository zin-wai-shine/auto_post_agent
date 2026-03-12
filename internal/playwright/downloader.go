package playwright

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func extractFBID(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	q := u.Query()
	fbid := q.Get("fbid")
	if fbid != "" {
		return fbid
	}

	// Sometimes it's in the path: /photo/?fbid=123... vs /photo/123/
	parts := strings.Split(u.Path, "/")
	for i, p := range parts {
		if (p == "photo" || p == "photos") && i+1 < len(parts) {
			if len(parts[i+1]) > 5 && strings.ContainsAny(parts[i+1], "0123456789") {
				return parts[i+1]
			}
		}
	}
	return ""
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

	fmt.Println("⚙️  Launching browser...")
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

	// DIRECT NAVIGATION: Go straight to the post URL (skip facebook.com homepage)
	fmt.Printf("🌐 Navigating directly to: %s\n", opts.URL)
	if _, err := page.Goto(opts.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("could not navigate: %w", err)
	}
	time.Sleep(3 * time.Second)

	// Check if we need to login (detect login wall on the current page)
	needsLogin := false

	// Check for password input (modal login popup)
	if count, _ := page.Locator(`input[type="password"]`).Count(); count > 0 {
		needsLogin = true
	}
	// Check for redirect to login page
	if strings.Contains(page.URL(), "/login") || strings.Contains(page.URL(), "checkpoint") {
		needsLogin = true
	}
	// Check for navigation bar (means logged in)
	navBar := page.Locator(`div[role="navigation"]`).First()
	navCount, _ := navBar.Count()
	if navCount == 0 && !needsLogin {
		// No nav bar and no login form — might need login
		needsLogin = true
	}

	if needsLogin {
		// Try auto-login with credentials first
		if cfg != nil && cfg.Facebook.Email != "" && cfg.Facebook.Password != "" {
			fmt.Println("🔐 Login required. Attempting auto-login...")
			if err := LoginWithCredentials(page, cfg); err != nil {
				fmt.Printf("   ⚠️ Auto-login failed: %v\n", err)
			} else {
				// Re-navigate to the target URL after login
				fmt.Printf("🌐 Re-navigating to: %s\n", opts.URL)
				if _, err := page.Goto(opts.URL, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
				}); err != nil {
					return fmt.Errorf("could not navigate after login: %w", err)
				}
				time.Sleep(3 * time.Second)
				needsLogin = false
			}
		}

		// If still needs login, wait for manual login
		if needsLogin {
			// Re-check after auto-login attempt
			navBar := page.Locator(`div[role="navigation"]`).First()
			navCount, _ := navBar.Count()
			if navCount == 0 {
				fmt.Println("═══════════════════════════════════════════════")
				fmt.Println("🔐 Not logged in to Facebook.")
				fmt.Println("   Please log in manually in the browser window.")
				fmt.Println("   Your login will be remembered for all future runs!")
				fmt.Println("═══════════════════════════════════════════════")

				fmt.Println("⏳ Waiting for you to log in... (timeout: 5 minutes)")
				_, err = page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
					Timeout: playwright.Float(300000),
				})
				if err != nil {
					return fmt.Errorf("login timed out: %w", err)
				}
				time.Sleep(2 * time.Second)
				fmt.Println("✅ Login successful!")

				// Re-navigate to the post URL after login
				fmt.Printf("🌐 Re-navigating to: %s\n", opts.URL)
				if _, err := page.Goto(opts.URL, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
				}); err != nil {
					return fmt.Errorf("could not navigate after login: %w", err)
				}
				time.Sleep(3 * time.Second)
			}
		}
	} else {
		fmt.Println("✅ Already logged in to Facebook!")
	}

	if err := os.MkdirAll(opts.SavePath, os.ModePerm); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	// Special Handling for Commerce / Marketplace Listings
	isCommerce := func() bool {
		u := page.URL()
		return strings.Contains(u, "/commerce/listing/") || strings.Contains(u, "/marketplace/item/")
	}

	if isCommerce() {
		return downloadCommerceImages(page, opts)
	}

	// Detect if this is a shared post (limited theater gallery)
	isSharedPost := strings.Contains(opts.URL, "/share/p/") || strings.Contains(opts.URL, "/share/")

	// ══════════════════════════════════════════════════════════════
	// STEP 1: FIND THE TARGET POST CONTAINER
	// Facebook pages can show multiple posts. We must scope all
	// photo searches to the CORRECT post to avoid downloading
	// images from the wrong post.
	// ══════════════════════════════════════════════════════════════
	fmt.Println("📜 Step 1: Finding target post on the page...")

	// Scroll to load content
	for scrollAttempt := 0; scrollAttempt < 5; scrollAttempt++ {
		page.Evaluate(`window.scrollBy(0, 800)`)
		time.Sleep(500 * time.Millisecond)
	}
	page.Evaluate(`window.scrollTo(0, 0)`)
	time.Sleep(500 * time.Millisecond)

	// Use JS to identify the right post container and extract photo links from it ONLY
	// We identify the "target post" by finding the article that contains photo links.
	// For single-post views (/share/p/), usually there's one main article.
	// For feed views, we need the article that was linked to (usually focused/highlighted).
	postInfo, _ := page.Evaluate(`
		() => {
			const articles = document.querySelectorAll('div[role="article"]');
			let targetArticle = null;
			let targetIndex = -1;

			// Strategy 1: If the page URL has been redirected to a permalink or post page,
			// the focused post is usually the LAST article with photos
			// (Facebook renders feed above, target post below)
			
			// Collect articles that have photo links
			const articlesWithPhotos = [];
			for (let i = 0; i < articles.length; i++) {
				const art = articles[i];
				const photoLinks = art.querySelectorAll('a[href*="/photo/"], a[href*="/photo.php"], a[href*="/photos/"], a[href*="/commerce/listing/"], a[href*="/marketplace/item/"]');
				const photoImages = art.querySelectorAll('img[src*="fbcdn"]');
				
				// Keep it if it has an explicit photo/commerce link, or if it has images.
				if (photoLinks.length > 0 || photoImages.length > 1) {
					articlesWithPhotos.push({ index: i, el: art, photoLinkCount: photoLinks.length, imgCount: photoImages.length });
				}
			}

			if (articlesWithPhotos.length === 1) {
				// Easy case: only one article has photos
				targetArticle = articlesWithPhotos[0].el;
				targetIndex = articlesWithPhotos[0].index;
			} else if (articlesWithPhotos.length > 1) {
				// Multiple articles with photos — take the LAST one 
				// (target post is typically rendered below feed items on share/permalink pages)
				const last = articlesWithPhotos[articlesWithPhotos.length - 1];
				targetArticle = last.el;
				targetIndex = last.index;
			} else if (articles.length > 0) {
				// No articles with photo links — just take the last article
				targetArticle = articles[articles.length - 1];
				targetIndex = articles.length - 1;
			}

			if (!targetArticle) {
				return { found: false, photoHrefs: [], originalPostURL: '' };
			}

			// Extract photo links from the target article ONLY
			const seen = new Set();
			const hrefs = [];
			const photoLinks = targetArticle.querySelectorAll('a[href*="/photo/"], a[href*="/photo.php"], a[href*="/photos/"], a[href*="/commerce/listing/"], a[href*="/marketplace/item/"]');
			for (const link of photoLinks) {
				let href = link.getAttribute('href');
				if (!href) continue;
				if (href.startsWith('/')) href = 'https://www.facebook.com' + href;
				const m = href.match(/fbid=(\d+)/) || href.match(/\/photo[s]?\/(\d+)/) || href.match(/\/listing\/(\d+)/) || href.match(/\/item\/(\d+)/);
				const key = m ? m[1] : href;
				if (!seen.has(key)) {
					seen.add(key);
					hrefs.push(href);
				}
			}

			// For shared posts: try to find the original post's permalink WITHIN the target article
			let originalPostURL = '';
			const postLinks = targetArticle.querySelectorAll('a[href*="/posts/"], a[href*="/permalink/"]');
			for (const link of postLinks) {
				const href = link.getAttribute('href');
				if (!href) continue;
				if (href.includes('/share/')) continue;
				if (href.includes('/posts/') || href.includes('/permalink/')) {
					originalPostURL = href.startsWith('/') ? 'https://www.facebook.com' + href : href;
					break;
				}
			}

			return {
				found: true,
				articleIndex: targetIndex,
				totalArticles: articles.length,
				articlesWithPhotos: articlesWithPhotos.length,
				photoHrefs: hrefs,
				originalPostURL: originalPostURL
			};
		}
	`)

	var preCollectedURLs []string
	var originalPostURL string
	targetArticleIndex := -1

	if postInfo != nil {
		if info, ok := postInfo.(map[string]interface{}); ok {
			if found, ok := info["found"].(bool); ok && found {
				if idx, ok := info["articleIndex"]; ok {
					switch v := idx.(type) {
					case int: targetArticleIndex = v
					case float64: targetArticleIndex = int(v)
					case float32: targetArticleIndex = int(v)
					}
				}
				totalArticles := 0
				if t, ok := info["totalArticles"]; ok {
					switch v := t.(type) {
					case int: totalArticles = v
					case float64: totalArticles = int(v)
					case float32: totalArticles = int(v)
					}
				}
				awp := 0
				if a, ok := info["articlesWithPhotos"]; ok {
					switch v := a.(type) {
					case int: awp = v
					case float64: awp = int(v)
					case float32: awp = int(v)
					}
				}
				fmt.Printf("   📍 Found target post: article[%d] of %d total (%d with photos)\n", targetArticleIndex, totalArticles, awp)

				if hrefs, ok := info["photoHrefs"].([]interface{}); ok {
					for _, v := range hrefs {
						if s, ok := v.(string); ok {
							preCollectedURLs = append(preCollectedURLs, s)
						}
					}
				}
				fmt.Printf("   📋 Found %d photo links in target post\n", len(preCollectedURLs))

				if orig, ok := info["originalPostURL"].(string); ok && orig != "" {
					originalPostURL = orig
					fmt.Printf("   📋 Found original post URL: %s\n", originalPostURL)
				}
			}
		}
	}

	if isSharedPost && originalPostURL == "" {
		fmt.Println("   🔍 Shared post detected but no original post link found yet.")
	}

	// ══════════════════════════════════════════════════════════════
	// STEP 2: ENTER THEATER MODE via photo link FROM THE TARGET POST
	// We use the targetArticleIndex to scope our search
	// ══════════════════════════════════════════════════════════════
	fmt.Println("📸 Step 2: Opening theater gallery from target post...")

	opened := false

	// Strategy 1: Use the pre-collected photo URLs from the target article
	if len(preCollectedURLs) > 0 {
		firstPhotoURL := preCollectedURLs[0]
		fmt.Printf("   📸 Navigating to first photo from target post: %s\n", firstPhotoURL)
		page.Goto(firstPhotoURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		opened = true
		time.Sleep(3 * time.Second)
	}

	// Strategy 2: Click photo link within the target article element (scoped by index)
	if !opened && targetArticleIndex >= 0 {
		fmt.Println("   🔍 Trying to click photo or commerce link in the target article...")
		clickResult, _ := page.Evaluate(fmt.Sprintf(`
			() => {
				const articles = document.querySelectorAll('div[role="article"]');
				const target = articles[%d];
				if (!target) return false;
				
				// Look for photo or commerce links
				const photoLink = target.querySelector('a[href*="/photo/"], a[href*="/photo.php"], a[href*="/photos/"], a[href*="/commerce/listing/"], a[href*="/marketplace/item/"]');
				if (photoLink) {
					photoLink.click();
					return true;
				}
				
				// Fallback: click a photo image
				const img = target.querySelector('img[src*="fbcdn"]');
				if (img) {
					img.click();
					return true;
				}
				return false;
			}
		`, targetArticleIndex))
		if clickResult != nil {
			if clicked, ok := clickResult.(bool); ok && clicked {
				opened = true
				time.Sleep(3 * time.Second)
				currURL := page.URL()
				if strings.Contains(currURL, "/photo") || strings.Contains(currURL, "fbid=") {
					fmt.Println("   ✅ Theater gallery opened from target post!")
				} else {
					dialog := page.Locator(`div[role="dialog"] img[src*="fbcdn"]`).First()
					if dc, _ := dialog.Count(); dc > 0 {
						fmt.Println("   ✅ Photo dialog opened from target post!")
					}
				}
			}
		}
	}

	// NO MORE GLOBAL FALLBACK HERE
	// We only want to download images from the ACTUAL post.
	// If we can't find photos, we let the normal flow continue.
	// Later, Step 4.5 will try to extract from originalPostURL.
	if !opened {
		fmt.Println("   ⚠️  No photos found in target post to click. Skipping local gallery check.")
	}

	// RE-CHECK: Did opening theater mode redirect us to a Commerce listing?
	if isCommerce() {
		fmt.Println("   🛍️  Redirected to commerce listing. Switching logic...")
		return downloadCommerceImages(page, opts)
	}

	// ══════════════════════════════════════════════════════════════
	// STEP 4: NAVIGATE GALLERY & DOWNLOAD IMAGES IN ONE PASS
	// Combined discovery + download for maximum speed
	// ══════════════════════════════════════════════════════════════
	fmt.Println("\n🚀 Step 4: Navigating gallery and downloading images...")

	downloadedSrcs := make(map[string]bool)
	discoveredFBIDs := make(map[string]bool)
	count := 0
	firstFBID := ""
	maxPhotos := 150 // Safety limit for very large galleries

	// isGalleryURL checks if the current URL is a photo or commerce/marketplace listing we can scrape
	isGalleryURL := func(u string) bool {
		return strings.Contains(u, "/photo") || 
		       strings.Contains(u, "fbid=") || 
		       strings.Contains(u, "/commerce/listing/") || 
		       strings.Contains(u, "/marketplace/item/")
	}

	runGalleryLoop := func() {
		localFirstFBID := ""
		localStale := 0
		localLastCount := count

		for i := 0; i < maxPhotos; i++ {
			currURL := page.URL()

			if isCommerce() && !strings.Contains(opts.URL, "commerce/listing") {
				fmt.Println("   🛍️  Detected commerce layout. Switching to specialized downloader...")
				_ = downloadCommerceImages(page, opts)
				return // Continue or exit depending on overall flow
			}

			// Check if this is a valid gallery page
			if isGalleryURL(currURL) {
				currFBID := extractFBID(currURL)
				
				// Handle case where URL is commerce listing (use the ID from URL as FBID)
				if currFBID == "" {
					if strings.Contains(currURL, "/listing/") {
						parts := strings.Split(currURL, "/listing/")
						if len(parts) > 1 {
							currFBID = strings.Split(parts[1], "/")[0]
							currFBID = strings.Split(currFBID, "?")[0]
						}
					}
				}

				// Wrap-around detection
				if currFBID != "" {
					if localFirstFBID == "" {
						localFirstFBID = currFBID
						// Also set global firstFBID if not set
						if firstFBID == "" {
							firstFBID = currFBID
						}
					} else if currFBID == localFirstFBID && count > localLastCount {
						fmt.Printf("   ✅ Wrap-around detected (back to first photo). Gallery pass complete! (%d total photos)\n", count)
						return
					}

					// Skip if already downloaded this FBID
					if discoveredFBIDs[currFBID] {
						// Don't break — just skip and continue pressing Next
						// This handles the case where the gallery shows an already-seen photo
					} else {
						discoveredFBIDs[currFBID] = true

						// IMMEDIATELY extract and download the current photo
						src := extractHighResImageSrc(page)
						if src != "" && !downloadedSrcs[src] {
							count++
							shortSrc := src
							if len(shortSrc) > 70 {
								shortSrc = shortSrc[:70]
							}
							fmt.Printf("   📥 [%d] Downloading: %s...\n", count, shortSrc)
							fileName := fmt.Sprintf("property_image_%d.jpg", count)
							filePath := filepath.Join(opts.SavePath, fileName)
							if err := downloadFile(src, filePath); err != nil {
								fmt.Printf("   ⚠️ %v\n", err)
								count-- // Don't count failed downloads
							} else {
								fmt.Printf("   ✅ %s\n", fileName)
								downloadedSrcs[src] = true
							}
						}
					}
				}
			}

			// Track stale state (no new downloads)
			if count == localLastCount {
				localStale++
			} else {
				localStale = 0
				localLastCount = count
			}

			// Recovery: login wall check (only when stale)
			if localStale > 5 && (strings.Contains(page.URL(), "login") || strings.Contains(page.URL(), "checkpoint")) {
				fmt.Println("   ⚠️  Detected login wall during discovery. Please log in again...")
				page.WaitForSelector(`div[role="navigation"]`, playwright.PageWaitForSelectorOptions{
					Timeout: playwright.Float(300000),
				})
				fmt.Println("   ✅ Resuming...")
				localStale = 0
			}

			// Focus dialog periodically
			if i%8 == 0 {
				page.Evaluate(`(function(){ const d = document.querySelector('div[role="dialog"]'); if(d) d.focus(); })()`)
			}

			// Click Next button via JS + ArrowRight for maximum reliability
			page.Evaluate(`
				(function() {
					const d = document.querySelector('div[role="dialog"]');
					if (d) d.focus();

					const next = document.querySelector('div[aria-label="Next photo"]') || 
					             document.querySelector('div[aria-label="Next"]') || 
					             document.querySelector('[aria-label*="Next"]');
					if (next) {
						['mousedown', 'mouseup', 'click'].forEach(t => {
							next.dispatchEvent(new MouseEvent(t, {bubbles: true, cancelable: true, view: window}));
						});
					}
				})()
			`)
			page.Keyboard().Press("ArrowRight")

			// Wait for React to update the URL — 1.5s base wait
			time.Sleep(1500 * time.Millisecond)

			// If URL hasn't changed after base wait, give more time
			if page.URL() == currURL && isGalleryURL(currURL) {
				time.Sleep(1500 * time.Millisecond)
			}

			// Stale tolerance: 15 rounds
			if localStale >= 15 {
				fmt.Printf("   🏁 Stale for %d rounds. Stopping gallery pass.\n", localStale)
				return
			}
		}
	}

	// Run the initial gallery loop
	runGalleryLoop()

	// ══════════════════════════════════════════════════════════════
	// STEP 4.5: FOR SHARED POSTS - Find and navigate to the ORIGINAL post
	// Shared posts only show a subset (5-7) of photos in their theater.
	// The FULL gallery is on the original post.
	// ══════════════════════════════════════════════════════════════
	if isSharedPost && count < 15 {
		fmt.Printf("\n🔄 Step 4.5: Shared post gallery had only %d photos. Looking for the FULL gallery...\n", count)

		// Strategy 1: Navigate to the original post URL if we found one
		if originalPostURL != "" {
			fmt.Printf("   🌐 Navigating to original post: %s\n", originalPostURL)
			page.Goto(originalPostURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			time.Sleep(3 * time.Second)
		} else {
			// Strategy 2: Go back to the shared post and look harder for the original
			fmt.Println("   🌐 Going back to shared post to find original post link...")
			page.Goto(opts.URL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			time.Sleep(3 * time.Second)
		}

		// Try to find links to the original post's photos  
		// In the shared post page, look for photo links from the original post
		origPhotos, _ := page.Evaluate(`
			() => {
				// Look for ALL <a> tags that link to photos
				const links = document.querySelectorAll('a[href*="/photo/"], a[href*="/photo.php"], a[href*="/photos/"]');
				const seen = new Set();
				const hrefs = [];
				for (const link of links) {
					let href = link.getAttribute('href');
					if (!href) continue;
					if (href.startsWith('/')) href = 'https://www.facebook.com' + href;
					const m = href.match(/fbid=(\d+)/) || href.match(/\/photo[s]?\/(\d+)/);
					const key = m ? m[1] : href;
					if (!seen.has(key)) {
						seen.add(key);
						hrefs.push(href);
					}
				}

				// Also look for the "See more" or "+X" overlay which might link to more photos
				const seeMoreLinks = document.querySelectorAll('a[href*="/media/set/"]');
				for (const link of seeMoreLinks) {
					let href = link.getAttribute('href');
					if (href && href.startsWith('/')) href = 'https://www.facebook.com' + href;
					if (href) hrefs.push('ALBUM:' + href);
				}

				return hrefs;
			}
		`)

		var origPhotoURLs []string
		var albumURL string
		if origPhotos != nil {
			if arr, ok := origPhotos.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						if strings.HasPrefix(s, "ALBUM:") {
							albumURL = strings.TrimPrefix(s, "ALBUM:")
						} else {
							origPhotoURLs = append(origPhotoURLs, s)
						}
					}
				}
			}
		}

		fmt.Printf("   📋 Found %d photo links on original/shared post page\n", len(origPhotoURLs))
		if albumURL != "" {
			fmt.Printf("   📋 Found media set album URL: %s\n", albumURL)
		}

		// Try to enter theater on the original post now
		foundNewTheater := false
		for _, photoURL := range origPhotoURLs {
			fbid := extractFBID(photoURL)
			if fbid != "" && discoveredFBIDs[fbid] {
				continue // Skip already downloaded
			}
			// Found a new photo - navigate to it to enter the original post's theater
			fmt.Printf("   📸 Entering original post theater via: %s\n", photoURL)
			page.Goto(photoURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			time.Sleep(3 * time.Second)
			foundNewTheater = true
			break
		}

		if !foundNewTheater && len(origPhotoURLs) > 0 {
			// All photo links were already visited, try the first one anyway
			// to re-enter theater which might give access to the full gallery
			fmt.Printf("   📸 Re-entering theater via first available photo...\n")
			page.Goto(origPhotoURLs[0], playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			})
			time.Sleep(3 * time.Second)
			foundNewTheater = true
		}

		if foundNewTheater {
			fmt.Println("   🚀 Running gallery loop on original/full post...")
			runGalleryLoop()
		}
	}

	// ══════════════════════════════════════════════════════════════
	// STEP 5: CHECK for any missed photos from pre-collected URLs
	// ══════════════════════════════════════════════════════════════
	if len(preCollectedURLs) > count && count > 0 {
		fmt.Printf("\n🔍 Step 5: Checking %d pre-collected URLs for missed photos...\n", len(preCollectedURLs))
		for _, photoURL := range preCollectedURLs {
			// Skip URLs we've already downloaded by FBID
			fbid := extractFBID(photoURL)
			if fbid != "" && discoveredFBIDs[fbid] {
				continue
			}

			fmt.Printf("   🌐 Checking extra URL: %s\n", photoURL)
			_, err := page.Goto(photoURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
			if err != nil {
				continue
			}
			time.Sleep(2 * time.Second)

			if strings.Contains(page.URL(), "login") || strings.Contains(page.URL(), "two_factor") {
				break
			}

			src := extractHighResImageSrc(page)
			if src != "" && !downloadedSrcs[src] {
				count++
				fileName := fmt.Sprintf("property_image_%d.jpg", count)
				filePath := filepath.Join(opts.SavePath, fileName)
				if err := downloadFile(src, filePath); err == nil {
					fmt.Printf("   ✅ %s (extra)\n", fileName)
					downloadedSrcs[src] = true
					if fbid != "" {
						discoveredFBIDs[fbid] = true
					}
				}
			}
		}
	}

	if count == 0 {
		fmt.Println("⚠️ No images downloaded.")
	} else {
		fmt.Printf("\n🎉 MISSION COMPLETE: %d images downloaded to %s\n", count, opts.SavePath)
	}
	return nil
}

// extractHighResImageSrc extracts the highest resolution image source from the current photo page.
func extractHighResImageSrc(page playwright.Page) string {
	// Priority 1: Facebook's media-vc-image (theater view high-res)
	for _, sel := range []string{
		`img[data-visualcompletion="media-vc-image"]`,
		`img.x1ey2m1z`,
	} {
		loc := page.Locator(sel).First()
		if c, _ := loc.Count(); c > 0 {
			s, _ := loc.GetAttribute("src")
			if s != "" && strings.Contains(s, "fbcdn") {
				return s
			}
		}
	}

	// Priority 2: Large fbcdn image in dialog (theater mode)
	result, _ := page.Evaluate(`
		() => {
			// First check inside the dialog (theater view)
			const dialog = document.querySelector('div[role="dialog"]');
			if (dialog) {
				const imgs = Array.from(dialog.querySelectorAll('img'));
				for (const img of imgs) {
					const src = img.getAttribute('src') || '';
					if (src.includes('fbcdn') && src.length > 200 && (img.offsetWidth > 200 || img.naturalWidth > 300)) {
						return src;
					}
				}
			}
			// Fallback: any large fbcdn image on the page
			const allImgs = Array.from(document.querySelectorAll('img'));
			for (const img of allImgs) {
				const src = img.getAttribute('src') || '';
				if (src.includes('fbcdn') && src.length > 200) {
					if (img.offsetWidth > 200 || img.naturalWidth > 300) {
						return src;
					}
				}
			}
			for (const img of allImgs) {
				const src = img.getAttribute('src') || '';
				if (src.includes('fbcdn') && src.length > 250) {
					return src;
				}
			}
			return '';
		}
	`)
	if result != nil {
		if s, ok := result.(string); ok && s != "" {
			return s
		}
	}

	return ""
}

func downloadCommerceImages(page playwright.Page, opts DownloadOptions) error {
	// 1. Wait for thumbnails to load
	page.WaitForSelector(`div[aria-label^="Thumbnail"]`, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(10000)})
	thumbnailButtons := page.Locator(`div[aria-label^="Thumbnail"]`)
	thumbCount, _ := thumbnailButtons.Count()

	fmt.Printf("   📸 Found %d listing thumbnails. Starting extraction...\n", thumbCount)
	downloadedSrcs := make(map[string]bool)
	downloadedCount := 0

	// Helper to find the ACTUAL large image via JS
	getHeroSrc := func() string {
		found, _ := page.Evaluate(`
			() => {
				const imgs = Array.from(document.querySelectorAll('img.xz74otr.x15mokao, img[alt^="Listing image"], div[role="dialog"] img'));
				const hero = imgs.find(img => {
					const isVisible = img.offsetWidth > 0 && img.offsetHeight > 0;
					const isLarge = img.naturalWidth >= 300 || img.naturalHeight >= 300; 
					const inButton = img.closest('button') || img.closest('[role="button"]');
					return isVisible && isLarge && (!inButton || img.offsetWidth > 400); 
				});
				return hero ? hero.getAttribute('src') : "";
			}
		`)
		if found != nil {
			return found.(string)
		}
		return ""
	}

	// PHASE 1: Iterate via thumbnails
	for i := 0; i < thumbCount; i++ {
		thumbnailButtons.Nth(i).Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)})
		time.Sleep(2 * time.Second)

		src := getHeroSrc()
		if src != "" && !downloadedSrcs[src] {
			fileName := fmt.Sprintf("listing_image_%d.jpg", downloadedCount+1)
			filePath := filepath.Join(opts.SavePath, fileName)
			if err := downloadFile(src, filePath); err == nil {
				fmt.Printf("   ✅ [%d] %s\n", downloadedCount+1, fileName)
				downloadedSrcs[src] = true
				downloadedCount++
			}
		}
	}

	// PHASE 2: Cycle via 'Next' button
	fmt.Println("   🔍 Checking for additional hidden images...")
	for i := 0; i < 20; i++ {
		nextBtn := page.Locator(`div[aria-label="View next image"], div[aria-label="Next"], [aria-label*="Next photo"]`).First()
		if count, _ := nextBtn.Count(); count > 0 {
			nextBtn.Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)})
			time.Sleep(1500 * time.Millisecond)

			src := getHeroSrc()
			if src != "" && !downloadedSrcs[src] {
				fileName := fmt.Sprintf("listing_image_%d.jpg", downloadedCount+1)
				filePath := filepath.Join(opts.SavePath, fileName)
				if err := downloadFile(src, filePath); err == nil {
					fmt.Printf("   ✅ [%d] %s (extra)\n", downloadedCount+1, fileName)
					downloadedSrcs[src] = true
					downloadedCount++
				}
			}
		} else {
			break
		}
	}

	if downloadedCount > 0 {
		fmt.Printf("\n🎉 MISSION COMPLETE: %d images downloaded to %s\n", downloadedCount, opts.SavePath)
		return nil
	}
	return fmt.Errorf("failed to download any commerce images")
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
