package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"dumbclaw/agent"
	"dumbclaw/config"
	"dumbclaw/integrations"
	"dumbclaw/llm"
	"dumbclaw/skills"
)


func runCLI(a *agent.Agent) {
	sep := strings.Repeat("─", 50)
	fmt.Println(sep)
	fmt.Println("DumbClaw — CLI Mode")
	fmt.Println(sep)
	fmt.Println("Commands: exit | reset | skills")
	fmt.Println(sep)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "reset":
			a.Reset()
			fmt.Println("Conversation reset.")
		case "skills":
			names := a.SkillNames()
			if len(names) == 0 {
				fmt.Println("No skills loaded.")
			} else {
				fmt.Println("Skills:", strings.Join(names, ", "))
			}
		default:
			fmt.Printf("\nDumbClaw: %s\n", a.ProcessMessage(input))
		}
	}
}

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if cfg.General.Debug {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetFlags(log.Ltime)
	}

	log.Println("Starting DumbClaw...")

	skillList := skills.Load(cfg)
	a := agent.New(llm.New(cfg.LLM), skillList)

	log.Printf("LLM: %s / %s | Skills: %d", cfg.LLM.Provider, cfg.LLM.Model, len(skillList))

	wa := cfg.Messaging.WhatsApp
	tg := cfg.Messaging.Telegram

	if wa.Enabled {
		bot := integrations.NewWhatsAppBot(wa, a.ProcessMessage, a.Reset)
		skills.SetSchedulerCallbacks(bot.Push, a.ProcessMessage)
		if tg.Enabled {
			// Run WhatsApp in background if Telegram is also starting
			go func() {
				if err := bot.Start(); err != nil {
					log.Fatalf("WhatsApp error: %v", err)
				}
			}()
		} else {
			// WhatsApp only — block on it directly
			if err := bot.Start(); err != nil {
				log.Fatalf("WhatsApp error: %v", err)
			}
			return
		}
	}

	if tg.Enabled {
		bot := integrations.NewTelegramBot(tg, a.ProcessMessage, a.Reset)
		if !wa.Enabled {
			skills.SetSchedulerCallbacks(bot.Push, a.ProcessMessage)
		}
		log.Println("Starting Telegram bot...")
		if err := bot.Start(); err != nil {
			log.Fatalf("Telegram error: %v", err)
		}
		return
	}

	// CLI mode: print scheduled messages to stdout
	skills.SetSchedulerCallbacks(func(text string) {
		fmt.Println("\n[Scheduled]:", text)
	}, a.ProcessMessage)
	runCLI(a)
}
