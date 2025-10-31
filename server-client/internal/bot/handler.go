package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	repo *Repository
	api  *tgbotapi.BotAPI
}

func NewHandler(api *tgbotapi.BotAPI, repo *Repository) *Handler {
	return &Handler{api: api, repo: repo}
}

func (h *Handler) HandleUpdate(update tgbotapi.Update) {
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	switch update.Message.Command() {
	case "start":
		username := update.Message.From.UserName
		chatID := update.Message.Chat.ID

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := h.repo.SaveChatID(ctx, username, chatID)
		if err != nil {
			log.Printf("failed to save chat id: %v", err)
		}

		msg := tgbotapi.NewMessage(chatID,
			fmt.Sprintf("Привет, %s! Твой chat_id: %d сохранён.", username, chatID))
		h.api.Send(msg)
	}
}
