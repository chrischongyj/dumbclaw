package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM       LLMConfig       `yaml:"llm"`
	Messaging MessagingConfig `yaml:"messaging"`
	Skills    SkillsConfig    `yaml:"skills"`
	General   GeneralConfig   `yaml:"general"`
}

type LLMConfig struct {
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	APIKey      string  `yaml:"api_key"`
	APIBase     string  `yaml:"api_base"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

type MessagingConfig struct {
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`
	Telegram TelegramConfig `yaml:"telegram"`
}

type WhatsAppConfig struct {
	Enabled     bool   `yaml:"enabled"`
	PhoneNumber string `yaml:"phone_number"`
	SessionDir  string `yaml:"session_dir"`
}

type TelegramConfig struct {
	Enabled      bool    `yaml:"enabled"`
	BotToken     string  `yaml:"bot_token"`
	AllowedUsers []int64 `yaml:"allowed_users"`
}

type SkillsConfig struct {
	Enabled   []string            `yaml:"enabled"`
	WebSearch WebSearchSkillConfig `yaml:"web_search"`
	FileOps   FileOpsSkillConfig   `yaml:"file_operations"`
}

type WebSearchSkillConfig struct {
	Engine     string `yaml:"engine"`
	MaxResults int    `yaml:"max_results"`
}

type FileOpsSkillConfig struct {
	AllowedDirs   []string `yaml:"allowed_dirs"`
	MaxFileSizeMB int      `yaml:"max_file_size"`
}

type GeneralConfig struct {
	Debug       bool   `yaml:"debug"`
	LogFile     string `yaml:"log_file"`
	HistoryFile string `yaml:"history_file"`
	MaxHistory  int    `yaml:"max_history"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file not found: %s\nCopy config.example.yaml to config.yaml and update it", path)
	}

	cfg := &Config{}
	cfg.LLM.Provider = "openai"
	cfg.LLM.Model = "gpt-4"
	cfg.LLM.Temperature = 0.7
	cfg.LLM.MaxTokens = 2000
	cfg.Skills.WebSearch.Engine = "duckduckgo"
	cfg.Skills.WebSearch.MaxResults = 5
	cfg.General.HistoryFile = "history.json"
	cfg.General.MaxHistory = 20

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid config YAML: %w", err)
	}

	return cfg, nil
}
