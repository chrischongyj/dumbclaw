package skills

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"dumbclaw/config"
)

func init() {
	Register("web_search", func(cfg *config.Config) Skill {
		return &WebSearchSkill{MaxResults: cfg.Skills.WebSearch.MaxResults}
	})
}

// WebSearchSkill scrapes DuckDuckGo HTML search results.
type WebSearchSkill struct {
	MaxResults int
}

func (s *WebSearchSkill) Name() string { return "web_search" }
func (s *WebSearchSkill) Description() string {
	return `Search the web using DuckDuckGo. Params: {"query": "your search terms"}`
}

func (s *WebSearchSkill) Execute(params map[string]any) (string, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing required param: query")
	}

	max := s.MaxResults
	if max <= 0 {
		max = 5
	}

	log.Printf("WebSearch: %q (max %d)", query, max)

	results, err := ddgSearch(query, max)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No results found for: " + query, nil
	}

	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "%d. %s\n   %s\n   %s\n", i+1, r.title, r.snippet, r.url)
	}
	return strings.TrimSpace(sb.String()), nil
}

type searchResult struct {
	title   string
	snippet string
	url     string
}

func ddgSearch(query string, max int) ([]searchResult, error) {
	reqURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; dumbclaw/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return parseDDGResults(string(body), max), nil
}

// parseDDGResults extracts results from DDG's HTML page.
// Each result is a <div class="result"> containing an <a class="result__a"> (title+URL)
// and a <a class="result__snippet"> (snippet).
func parseDDGResults(body string, max int) []searchResult {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil
	}

	var results []searchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= max {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "result") && !hasClass(n, "result--ad") {
			r := extractResult(n)
			if r.title != "" && r.url != "" {
				results = append(results, r)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return results
}

func extractResult(n *html.Node) searchResult {
	var r searchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			cls := attrVal(n, "class")
			if strings.Contains(cls, "result__a") {
				r.title = textContent(n)
				// DDG wraps the real URL in a redirect; grab the data-href or href
				if href := attrVal(n, "href"); href != "" {
					r.url = cleanDDGURL(href)
				}
			}
			if strings.Contains(cls, "result__snippet") {
				r.snippet = textContent(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return r
}

// cleanDDGURL strips DDG's redirect wrapper and returns the real URL.
func cleanDDGURL(href string) string {
	// DDG HTML hrefs look like //duckduckgo.com/l/?uddg=https%3A%2F%2F...
	if idx := strings.Index(href, "uddg="); idx != -1 {
		encoded := href[idx+5:]
		if amp := strings.Index(encoded, "&"); amp != -1 {
			encoded = encoded[:amp]
		}
		decoded, err := url.QueryUnescape(encoded)
		if err == nil {
			return decoded
		}
	}
	return href
}

func hasClass(n *html.Node, cls string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == cls {
					return true
				}
			}
		}
	}
	return false
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
