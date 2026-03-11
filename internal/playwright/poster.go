package playwright

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/zinwaishine/super-agent/internal/config"
)

type PostOptions struct {
	Target    string   // "pages" or "marketplace"
	PageID    string   // The username or ID of the Facebook Page (e.g. "myrealestatepage")
	TargetURL string   // Explicit URL to bypass auto-generation for Pages
	Content   string   // The AI-generated text to post
	Images    []string // Local file paths of images to upload
	Headless  bool
	DryRun    bool
	AutoPost  bool
}

// PostListing automates a browser to post to Facebook.
func PostListing(opts PostOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sessionPath := expandTilde(cfg.Facebook.SessionPath)
	if _, err := os.Stat(sessionPath); err != nil {
		return fmt.Errorf("no facebook session found. Please run 'super-agent auth facebook' first")
	}

	fmt.Println("🤖 Booting Playwright Sniper Bot...")

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(opts.Headless),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		StorageStatePath: playwright.String(sessionPath),
		Viewport:         &playwright.Size{Width: 1280, Height: 720},
		UserAgent:        playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return fmt.Errorf("could not create context: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}

	if opts.Target == "pages" {
		if opts.PageID == "" && opts.TargetURL == "" {
			return fmt.Errorf("you must provide a --page-id or --url when targeting 'pages'")
		}

		targetURL := opts.TargetURL
		if targetURL == "" {
			targetURL = fmt.Sprintf("https://www.facebook.com/%s", opts.PageID)
		}

		fmt.Printf("🌐 Navigating to Facebook Page / Business Suite: %s\n", targetURL)

		if _, err := page.Goto(targetURL); err != nil {
			return err
		}

		fmt.Println("⏳ Waiting for page load...")
		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateNetworkidle})

		// This is a "Best Effort" automation since Facebook's DOM changes often.
		// We try to find the "Create post" or textbox area based on common aria-labels or roles.
		var clicked bool

		if strings.Contains(targetURL, "composer") {
			fmt.Println("🎯 Business Suite Composer explicitly loaded. Skipping 'Create Post' button hunt.")
			clicked = true
		} else {
			fmt.Println("🎯 Searching for 'Create post' input...")

			// Facebook often uses spans with text like "Write something..." or "Create post"
			createPostSelectors := []string{
				`div[aria-label="Create a post"]`,
				`div[aria-label="Create post"]`,
				`text="Write something..."`,
				`text="Create Post"`,
			}

			for _, sel := range createPostSelectors {
				if count, _ := page.Locator(sel).Count(); count > 0 {
					fmt.Printf("   Found create post button via %s. Clicking...\n", sel)
					page.Locator(sel).First().Click()
					clicked = true
					break
				}
			}

			if !clicked {
				if strings.Contains(targetURL, "business.facebook.com") {
					fmt.Println("⚠️  Business Suite UI Detected. Cannot auto-click 'Create Post'.")
				} else {
					fmt.Println("⚠️  Could not firmly locate the 'Create post' button automagically.")
				}

				if !opts.Headless {
					fmt.Println("\n👇 MANUAL INTERVENTION REQUIRED 👇")
					fmt.Println("1. Navigate the browser to the exact page where you want to post.")
					fmt.Println("2. Manually click the 'Create Post' button in the browser to open the popup.")
					fmt.Println("3. Wait until you see the 'Add Photo/Video' option and Text box on screen.")
					fmt.Print("👉 Press ENTER in this terminal once the popup is completely ready...")

					// Wait for user to press ENTER
					os.Stdin.Read(make([]byte, 1))
					clicked = true
				} else {
					fmt.Println("   Screenshotting to debug...")
					page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String("debug_fb_post.png")})
				}
			}
		}

		if clicked {
			// Usually a modal pops up with a new input field
			time.Sleep(3 * time.Second)

			if len(opts.Images) > 0 {
				fmt.Printf("🖼️  Uploading %d images...\n", len(opts.Images))
				// Meta Business Suite and Pages use explicit buttons to trigger the file upload.
				// We tell Playwright to intercept the "Choose File" dialog box that opens.
				fileChooser, err := page.ExpectFileChooser(func() error {
					// Try clicking the Meta Business Suite "Add photo/video" button or regular page "Photo/video" button
					photoBtn, _ := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "photo/video"}).All()
					if len(photoBtn) == 0 {
						photoBtn, _ = page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Add Photo/Video"}).All()
					}
					if len(photoBtn) == 0 {
						// Fallback to text match
						btn := page.Locator(`text="Add photo/video"`).First()
						if c, _ := btn.Count(); c > 0 {
							return btn.Click()
						}
						// Fallback to older text
						btn2 := page.Locator(`text="Photo/video"`).First()
						if c, _ := btn2.Count(); c > 0 {
							return btn2.Click()
						}
						return fmt.Errorf("could not find photo button")
					}
					// Click the first visible one
					for _, b := range photoBtn {
						if v, _ := b.IsVisible(); v {
							return b.Click()
						}
					}
					return fmt.Errorf("no visible photo button")
				})

				if err == nil {
					err = fileChooser.SetFiles(opts.Images)
					if err == nil {
						fmt.Println("   ✅ Images attached via File Chooser!")
						time.Sleep(3 * time.Second) // wait for upload preview
					} else {
						fmt.Printf("   ⚠️ Failed to set files: %v\n", err)
					}
				} else {
					fmt.Println("   ⚠️ Could not intercept the File Chooser dialog. Button not found?")
				}
			}

			fmt.Println("📝 Injecting AI-generated text...")

			// FULL-PROOF METHOD: We bypass arbitrary CSS selectors and use Playwright's A11y tree
			// to find ALL valid textboxes on the screen. We then click the first highly-visible one.
			var foundBox bool

			// Try getting all things marked as "textbox" by the browser's accessibility tree
			textboxes, err := page.GetByRole("textbox").All()
			if err == nil {
				for _, box := range textboxes {
					isVisible, _ := box.IsVisible()
					isEditable, _ := box.IsEditable()
					if isVisible && isEditable {
						// Hard physical click on the box
						box.Click()
						// Small wait for the editor to focus
						time.Sleep(500 * time.Millisecond)

						// Type using the main keyboard emulator instead of the element to avoid permission blocks
						err := page.Keyboard().Type(opts.Content, playwright.KeyboardTypeOptions{Delay: playwright.Float(10)})
						if err == nil {
							foundBox = true
							break
						}
					}
				}
			}

			// Fallback if browser role doesn't match
			if !foundBox {
				fallbackSelectors := []string{
					`div[data-lexical-editor="true"]`,
					`div.notranslate[data-contents="true"]`,
					`div[contenteditable="true"]`,
				}
				for _, sel := range fallbackSelectors {
					locs, _ := page.Locator(sel).All()
					for _, loc := range locs {
						v, _ := loc.IsVisible()
						if v {
							loc.Click()
							time.Sleep(500 * time.Millisecond)
							page.Keyboard().Type(opts.Content, playwright.KeyboardTypeOptions{Delay: playwright.Float(10)})
							foundBox = true
							break
						}
					}
					if foundBox {
						break
					}
				}
			}

			if foundBox {
				fmt.Println("✅ Text injected!")

				if opts.DryRun {
					fmt.Println("🛑 DRY RUN: Skipping the final 'Publish' click.")
					page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String("dry_run_success.png")})

					if !opts.Headless {
						fmt.Println("\n👀 ALMOST DONE: Look at the browser window!")
						fmt.Println("   The post is fully typed out and the images are attached.")
						fmt.Println("   👉 You can review it, tweak the text, or manually push 'Publish' yourself.")
						fmt.Print("\nPress ENTER here in the terminal when you are ready to gracefully close the robot...")
						os.Stdin.Read(make([]byte, 1))
					} else {
						time.Sleep(5 * time.Second) // pause in headless
					}
				} else {
					if opts.AutoPost {
						fmt.Println("\n👀 ALMOST DONE: The post is typed and images are attached.")
						fmt.Println("   ⏳ AUTO-POST: Waiting 10 seconds for UI to settle before clicking Publish...")
						time.Sleep(10 * time.Second)
					} else {
						fmt.Println("\n👀 ALMOST DONE: The post is typed and images are attached.")
						fmt.Print("👉 Press ENTER to click 'Publish' (Wait until you see it is ready)...")
						os.Stdin.Read(make([]byte, 1))
					}

					fmt.Println("🚀 Clicking Publish...")
					publishLocators := []string{
						`div[aria-label="Post"]`,
						`div[aria-label="Publish"]`,
						`button:has-text("Publish")`,
						`text="Publish"`,
					}

					var clickedPublish bool
					for _, sel := range publishLocators {
						postBtn := page.Locator(sel).First()
						if count, _ := postBtn.Count(); count > 0 {
							if isVis, _ := postBtn.IsVisible(); isVis {
								fmt.Printf("   Found publish button via %s.\n", sel)
								postBtn.Click()
								fmt.Println("🎉 Successfully Published to Facebook Page!")
								time.Sleep(5 * time.Second) // Wait for network
								clickedPublish = true
								break
							}
						}
					}

					if !clickedPublish {
						fmt.Println("⚠️ Could not find the final 'Post' / 'Publish' button.")
					}
				}
			} else {
				fmt.Println("⚠️ Could not find the text input box inside the modal.")
				fmt.Println("   Screenshotting to debug: textbox_failed.png")
				page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String("textbox_failed.png")})
			}
		}

	} else {
		// Marketplace stub
		fmt.Println("🌐 Navigating to Facebook Marketplace...")
		page.Goto("https://www.facebook.com/marketplace/create/item")
		time.Sleep(5 * time.Second)
		fmt.Println("⚠️  Marketplace posting not yet implemented in demo mode.")
	}

	return nil
}
