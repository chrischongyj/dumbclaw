package integrations

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"dumbclaw/config"
)

// TelegramBot connects the Agent to a Telegram bot.
type TelegramBot struct {
	cfg     config.TelegramConfig
	handler func(string) string
	reset   func()
	bot     *tgbotapi.BotAPI
}

func NewTelegramBot(cfg config.TelegramConfig, handler func(string) string, reset func()) *TelegramBot {
	return &TelegramBot{cfg: cfg, handler: handler, reset: reset}
}

// Start begins polling for Telegram updates. Blocks until the bot stops.
func (t *TelegramBot) Start() error {
	var err error
	t.bot, err = tgbotapi.NewBotAPI(t.cfg.BotToken)
	if err != nil {
		return fmt.Errorf("failed to connect to Telegram: %w", err)
	}

	log.Printf("Telegram bot started: @%s", t.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	for update := range t.bot.GetUpdatesChan(u) {
		if update.Message == nil {
			continue
		}
		go t.handleUpdate(update)
	}
	return nil
}

func (t *TelegramBot) handleUpdate(update tgbotapi.Update) {
	msg := update.Message

	if len(t.cfg.AllowedUsers) > 0 {
		allowed := false
		for _, id := range t.cfg.AllowedUsers {
			if id == msg.From.ID {
				allowed = true
				break
			}
		}
		if !allowed {
			log.Printf("Telegram: unauthorized user %d (@%s)", msg.From.ID, msg.From.UserName)
			t.reply(msg, "Sorry, you're not authorized to use this bot.")
			return
		}
	}

	switch msg.Command() {
	case "start":
		log.Printf("Telegram [%d/@%s]: /start", msg.From.ID, msg.From.UserName)
		t.reply(msg, "Hello! I'm DumbClaw, your AI assistant.\nSend me a message to get started.\n\n/help — show commands")
	case "help":
		log.Printf("Telegram [%d/@%s]: /help", msg.From.ID, msg.From.UserName)
		t.reply(msg, "/start — start\n/help — this message\n/reset — reset conversation\n\nJust send a message to chat!")
	case "reset":
		log.Printf("Telegram [%d/@%s]: /reset", msg.From.ID, msg.From.UserName)
		t.reset()
		t.reply(msg, "Conversation reset.")
	default:
		if msg.Text != "" {
			log.Printf("Telegram [%d/@%s]: %s", msg.From.ID, msg.From.UserName, msg.Text)
			t.bot.Send(tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatTyping))
			response := t.handler(msg.Text)
			log.Printf("Telegram replying to %d/@%s: %s", msg.From.ID, msg.From.UserName, response)
			t.reply(msg, response)
		}
	}
}

// Push sends a proactive message to the first allowed user.
func (t *TelegramBot) Push(text string) {
	if t.bot == nil {
		log.Println("Telegram push: bot not ready")
		return
	}
	if len(t.cfg.AllowedUsers) == 0 {
		log.Println("Telegram push: no allowed_users configured")
		return
	}
	msg := tgbotapi.NewMessage(t.cfg.AllowedUsers[0], "[Scheduled]: "+text)
	if _, err := t.bot.Send(msg); err != nil {
		log.Printf("Telegram push error: %v", err)
	}
}

func (t *TelegramBot) reply(msg *tgbotapi.Message, text string) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyToMessageID = msg.MessageID
	if _, err := t.bot.Send(reply); err != nil {
		log.Printf("Telegram send error: %v", err)
	}
}
