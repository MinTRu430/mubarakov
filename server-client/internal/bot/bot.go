package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	handler *Handler
}

func NewBot(token string, repo *Repository) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	handler := NewHandler(api, repo)
	return &Bot{api: api, handler: handler}, nil
}

func (b *Bot) Start() {
	log.Printf("Telegram bot authorized as @%s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	for update := range updates {
		b.handler.HandleUpdate(update)
	}
}

func (b *Bot) SendCode(chatID int64, code string) error {
	msg := tgbotapi.NewMessage(chatID, "Ваш код для входа: "+code)
	_, err := b.api.Send(msg)
	return err
}
