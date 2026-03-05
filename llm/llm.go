package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"dumbclaw/config"
)

// Message is a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLM handles communication with language model APIs.
type LLM struct {
	cfg config.LLMConfig
}

func New(cfg config.LLMConfig) *LLM {
	return &LLM{cfg: cfg}
}

// Chat sends messages and returns the model's reply.
func (l *LLM) Chat(messages []Message) (string, error) {
	switch l.cfg.Provider {
	case "openai", "ollama":
		return l.chatOpenAI(messages)
	case "anthropic":
		return l.chatAnthropic(messages)
	default:
		return "", fmt.Errorf("unsupported provider %q (use openai, anthropic, or ollama)", l.cfg.Provider)
	}
}

func (l *LLM) chatOpenAI(messages []Message) (string, error) {
	baseURL := l.cfg.APIBase
	if baseURL == "" {
		if l.cfg.Provider == "ollama" {
			baseURL = "http://localhost:11434/v1"
		} else {
			baseURL = "https://api.openai.com/v1"
		}
	}

	body, _ := json.Marshal(map[string]any{
		"model":       l.cfg.Model,
		"messages":    messages,
		"temperature": l.cfg.Temperature,
		"max_tokens":  l.cfg.MaxTokens,
	})

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM API error %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	return result.Choices[0].Message.Content, nil
}

func (l *LLM) chatAnthropic(messages []Message) (string, error) {
	var system string
	var filtered []Message
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		} else {
			filtered = append(filtered, m)
		}
	}

	payload := map[string]any{
		"model":      l.cfg.Model,
		"messages":   filtered,
		"max_tokens": l.cfg.MaxTokens,
	}
	if system != "" {
		payload["system"] = system
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", l.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}

	return result.Content[0].Text, nil
}
