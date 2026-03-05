package skills

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"dumbclaw/config"
)

const (
	browseMaxChars = 8000
	browseTimeout  = 30 * time.Second
)

var (
	rodBrowser *rod.Browser
	rodOnce    sync.Once
	rodErr     error
)

func getBrowser() (*rod.Browser, error) {
	rodOnce.Do(func() {
		u, err := launcher.New().Headless(true).Leakless(false).Launch()
		if err != nil {
			rodErr = fmt.Errorf("failed to launch browser: %w", err)
			return
		}
		rodBrowser = rod.New().ControlURL(u)
		rodErr = rodBrowser.Connect()
		if rodErr == nil {
			log.Println("Browse: headless browser ready")
		}
	})
	return rodBrowser, rodErr
}

func init() {
	Register("browse", func(cfg *config.Config) Skill {
		return &BrowseSkill{}
	})
}

// BrowseSkill fetches a URL using a headless browser and returns visible text.
type BrowseSkill struct{}

func (s *BrowseSkill) Name() string { return "browse" }
func (s *BrowseSkill) Description() string {
	return `Fetch and read the text content of any web page, including JavaScript-rendered SPAs. Params: {"url": "https://example.com"}`
}

func (s *BrowseSkill) Execute(params map[string]any) (string, error) {
	log.Printf("Browse: received params: %+v", params)
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("missing required param: url")
	}

	log.Printf("Browse: %s", url)

	b, err := getBrowser()
	if err != nil {
		log.Printf("Browse: error initializing browser: %v", err)
		return "", fmt.Errorf("browser unavailable: %w", err)
	}

	page, err := b.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return "", fmt.Errorf("failed to open page: %w", err)
	}
	defer page.Close()

	// Wait for page to load; ignore timeout errors and use whatever loaded
	if err := page.Timeout(browseTimeout).WaitLoad(); err != nil {
		log.Printf("Browse: load timeout for %s (using partial content)", url)
	}

	res, err := page.Eval(`() => document.body.innerText`)
	if err != nil {
		return "", fmt.Errorf("failed to get page text: %w", err)
	}
	text := res.Value.String()

	text = compactText(text)
	if len(text) > browseMaxChars {
		text = text[:browseMaxChars] + "\n\n[truncated]"
	}
	if strings.TrimSpace(text) == "" {
		return "Page loaded but no readable text found.", nil
	}
	return text, nil
}

// compactText filters short noisy lines (UI elements, buttons) and collapses blank lines.
func compactText(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blank := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blank++
			if blank <= 1 {
				out = append(out, "")
			}
			continue
		}
		blank = 0
		if len(trimmed) < 20 {
			continue // skip short fragments (nav items, buttons, labels)
		}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n")
}
