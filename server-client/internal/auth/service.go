package auth

import (
	"context"
	"fmt"
	"server-client/internal/bot"
	"server-client/internal/utils"
	"time"
)

type Service struct {
	repo  *Repository
	tgBot *bot.Bot
}

func NewService(repo *Repository, tgBot *bot.Bot) *Service {
	return &Service{repo: repo, tgBot: tgBot}
}

func (s *Service) SaveCodeWord(ctx context.Context, login, codeWord string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.repo.UpdateCodeWord(ctx, login, codeWord); err != nil {
		return err
	}

	tgCode := utils.RandomStringSimple(6)
	if err := s.repo.UpdateTelegramCode(ctx, login, tgCode); err != nil {
		return err
	}

	chatID, err := s.repo.GetChatID(ctx, login)
	if err != nil {
		return err
	}

	return s.tgBot.SendCode(chatID, tgCode)
}

func (s *Service) CheckAuth(ctx context.Context, login, multiHashFromClient string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pcw, err := s.repo.GetUser(ctx, login)
	if err != nil {
		return false, err
	}

	if time.Now().UTC().After(pcw.CodeWordLifeTime) {
		return false, fmt.Errorf("time expired")
	}

	multiHashFromServer := utils.HashMD5(pcw.HashedPass + utils.HashMD5(pcw.CodeWord) + pcw.TgCode)
	if multiHashFromClient != multiHashFromServer {
		return false, fmt.Errorf("wrong password")
	}

	return true, nil
}
