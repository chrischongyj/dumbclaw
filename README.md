# DumbClaw

<img src="https://i.imgur.com/9fGIKLk.jpeg" alt="A red flower" width="400" height="400">

DumbClaw is a deliberately simple AI assistant bot — the "dumb" version of [OpenClaw](https://github.com/OpenClaw). No framework magic, no abstractions for the sake of it. Every feature lives in one readable file. If you want to add something new, you should be able to vibe-code it in minutes.

## Features

- **Skills system** — one file per skill, self-registers via `init()`, no switch statements
- **WhatsApp** — full multi-device support via whatsmeow
- **Telegram** — polling bot with user allowlist
- **Scheduler** — recurring tasks via the `schedule` skill (e.g. hourly weather updates)
- **OpenAI-compatible** — works with OpenAI, Anthropic, Ollama, or any custom API base
- **CLI mode** — for quick local testing without a messaging platform

## Quickstart

```bash
go mod tidy
cp config.example.yaml config.yaml
# edit config.yaml — add your API key, enable a messaging platform
go run .
```

Or build a binary:

```bash
go build -o dumbclaw .
./dumbclaw
```

## Configuration

```yaml
llm:
  provider: "openai"      # openai | anthropic | ollama
  model: "gpt-4"
  api_key: "your-api-key"
  api_base: ""            # optional: custom endpoint (e.g. https://api.poe.com/v1)

messaging:
  whatsapp:
    enabled: false
    phone_number: "+1234567890"
  telegram:
    enabled: false
    bot_token: "your-token"
    allowed_users: []     # empty = allow everyone

skills:
  enabled:
    - web_search
    - file_operations
    - weather
    - schedule
```

## Project Structure

```
dumbclaw/
├── main.go                    # Entry point, ~100 lines
│
├── config/
│   └── config.go              # Config structs and YAML loader
│
├── llm/
│   └── llm.go                 # LLM HTTP client (OpenAI / Anthropic / Ollama)
│
├── agent/
│   └── agent.go               # Conversation loop and skill dispatch
│
├── skills/
│   ├── skill.go               # Skill interface (3 methods)
│   ├── registry.go            # Auto-registration and Load()
│   ├── websearch.go           # DuckDuckGo HTML scraper
│   ├── fileops.go             # Read / write / list files
│   ├── weather.go             # Current weather via Open-Meteo (free, no key)
│   └── schedule.go            # Recurring tasks with push notifications
│
├── integrations/
│   ├── telegram.go            # Telegram bot
│   └── whatsapp.go            # WhatsApp via whatsmeow
│
└── workspace/                 # Default directory for file_operations skill
```

## Adding a Skill

1. Create `skills/yourskill.go`
2. Implement the `Skill` interface
3. Self-register in `init()` — nothing else changes:

```go
func init() {
    Register("your_skill", func(cfg *config.Config) Skill {
        return &YourSkill{}
    })
}
```

4. Enable it in `config.yaml` under `skills.enabled`

That's it. The LLM automatically learns what the skill does from its `Description()`.

## License

MIT
