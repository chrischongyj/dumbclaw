package skills

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"dumbclaw/config"
)

func init() {
	Register("rss", func(cfg *config.Config) Skill {
		return &RSSSkill{}
	})
}

// RSSSkill fetches and parses an RSS or Atom feed.
type RSSSkill struct{}

func (s *RSSSkill) Name() string { return "rss" }
func (s *RSSSkill) Description() string {
	return `Fetch and read an RSS or Atom feed. Params: {"url": "https://example.com/feed.xml", "limit": 10}`
}

func (s *RSSSkill) Execute(params map[string]any) (string, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("missing required param: url")
	}
	limit := 10
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	log.Printf("RSS: %s (limit %d)", url, limit)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; dumbclaw/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read feed: %w", err)
	}

	items, title, err := parseFeed(body)
	if err != nil {
		return "", fmt.Errorf("failed to parse feed: %w", err)
	}
	if len(items) == 0 {
		return "Feed is empty or could not be parsed.", nil
	}
	if len(items) > limit {
		items = items[:limit]
	}

	var sb strings.Builder
	if title != "" {
		fmt.Fprintf(&sb, "# %s\n\n", title)
	}
	for i, item := range items {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, item.Title)
		if item.Link != "" {
			fmt.Fprintf(&sb, "   %s\n", item.Link)
		}
		if item.Summary != "" {
			fmt.Fprintf(&sb, "   %s\n", truncate(item.Summary, 200))
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String()), nil
}

type feedItem struct {
	Title   string
	Link    string
	Summary string
}

// parseFeed handles both RSS 2.0 and Atom feeds.
func parseFeed(data []byte) ([]feedItem, string, error) {
	// Try RSS first
	var rss struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(data, &rss); err == nil && len(rss.Channel.Items) > 0 {
		var items []feedItem
		for _, i := range rss.Channel.Items {
			items = append(items, feedItem{
				Title:   strings.TrimSpace(i.Title),
				Link:    strings.TrimSpace(i.Link),
				Summary: stripTags(i.Description),
			})
		}
		return items, rss.Channel.Title, nil
	}

	// Try Atom
	var atom struct {
		XMLName xml.Name `xml:"feed"`
		Title   string   `xml:"title"`
		Entries []struct {
			Title   string `xml:"title"`
			Summary string `xml:"summary"`
			Content string `xml:"content"`
			Links   []struct {
				Href string `xml:"href,attr"`
				Rel  string `xml:"rel,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(data, &atom); err == nil && len(atom.Entries) > 0 {
		var items []feedItem
		for _, e := range atom.Entries {
			link := ""
			for _, l := range e.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					link = l.Href
					break
				}
			}
			summary := e.Summary
			if summary == "" {
				summary = e.Content
			}
			items = append(items, feedItem{
				Title:   strings.TrimSpace(e.Title),
				Link:    link,
				Summary: stripTags(summary),
			})
		}
		return items, atom.Title, nil
	}

	return nil, "", fmt.Errorf("not a valid RSS or Atom feed")
}

// stripTags removes HTML tags from a string.
func stripTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, c := range s {
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
			out.WriteRune(' ')
		case !inTag:
			out.WriteRune(c)
		}
	}
	return strings.TrimSpace(out.String())
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
