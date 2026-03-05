package integrations

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	_ "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"dumbclaw/config"
)

// WhatsAppBot connects the Agent to WhatsApp via the whatsmeow library.
// On first run it prints a QR code to the terminal — scan it with WhatsApp
// to link the device. The session is then saved in cfg.SessionDir.
type WhatsAppBot struct {
	cfg     config.WhatsAppConfig
	handler func(string) string
	reset   func()
	client  *whatsmeow.Client
}

func NewWhatsAppBot(cfg config.WhatsAppConfig, handler func(string) string, reset func()) *WhatsAppBot {
	return &WhatsAppBot{cfg: cfg, handler: handler, reset: reset}
}

// Start connects to WhatsApp and blocks until the process exits.
func (w *WhatsAppBot) Start() error {
	if err := os.MkdirAll(w.cfg.SessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session dir: %w", err)
	}

	dbPath := w.cfg.SessionDir + "/session.db"
	container, err := sqlstore.New(context.Background(), "sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)", nil)
	if err != nil {
		return fmt.Errorf("failed to open WhatsApp store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get device store: %w", err)
	}

	w.client = whatsmeow.NewClient(deviceStore, nil)
	w.client.AddEventHandler(w.handleEvent)

	if w.client.Store.ID == nil {
		// Not logged in yet — show QR code for scanning
		qrChan, _ := w.client.GetQRChannel(context.Background())
		if err := w.client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		log.Println("Scan the QR code below with WhatsApp (Linked Devices → Link a Device):")
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				log.Printf("WhatsApp login: %s", evt.Event)
			}
		}
	} else {
		if err := w.client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		log.Println("WhatsApp connected!")
	}

	// Block forever (events are handled in handleEvent)
	select {}
}

// Push sends a proactive message to the configured phone number.
func (w *WhatsAppBot) Push(text string) {
	if w.client == nil {
		log.Println("WhatsApp push: client not ready")
		return
	}
	phone := strings.TrimPrefix(w.cfg.PhoneNumber, "+")
	jid := types.NewJID(phone, types.DefaultUserServer)
	_, err := w.client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String("[Scheduled]: " + text),
	})
	if err != nil {
		log.Printf("WhatsApp push error: %v", err)
	}
}

func (w *WhatsAppBot) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.Connected:
		log.Println("WhatsApp: connected and ready")
	case *events.Disconnected:
		log.Println("WhatsApp: disconnected")
	case *events.Message:
		w.handleMessage(v)
	}
}

func (w *WhatsAppBot) handleMessage(msg *events.Message) {
	// Skip messages sent by this bot instance (but allow messages the user sends to themselves from their phone)
	if msg.Info.IsFromMe && w.client.Store.ID != nil && msg.Info.Sender.Device == w.client.Store.ID.Device {
		return
	}
	if msg.Info.IsGroup {
		log.Printf("WhatsApp: ignoring group message from %s", msg.Info.Sender.User)
		return
	}

	// Extract plain text — try all common message types
	text := msg.Message.GetConversation()
	if text == "" {
		if ext := msg.Message.GetExtendedTextMessage(); ext != nil {
			text = ext.GetText()
		}
	}
	if text == "" {
		log.Printf("WhatsApp: received non-text message from %s (ignoring)", msg.Info.Sender.User)
		return
	}

	log.Printf("WhatsApp [%s]: %s", msg.Info.Sender.User, text)

	// Show typing indicator while processing
	w.client.SendChatPresence(context.Background(), msg.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer w.client.SendChatPresence(context.Background(), msg.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	var response string
	if text == "/reset" || text == "!reset" {
		w.reset()
		response = "Conversation reset."
	} else {
		response = w.handler(text)
	}

	log.Printf("WhatsApp replying to %s: %s", msg.Info.Sender.User, response)

	_, err := w.client.SendMessage(context.Background(), msg.Info.Sender, &waE2E.Message{
		Conversation: proto.String("[Response]: " + response),
	})
	if err != nil {
		log.Printf("WhatsApp send error: %v", err)
	}
}
