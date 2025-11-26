package reg

import (
	"context"
	"errors"
	"regexp"
	"server-client/internal/utils"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterUser(ctx context.Context, login, password, tgLogin string) (int, string, error) {
	if err := validateLogin(login); err != nil {
		return 0, "", err
	}

	if err := validatePassword(password); err != nil {
		return 0, "", err
	}

	hashedPass := utils.HashMD5(password)

	if err := validateTelegram(tgLogin); err != nil {
		return 0, "", err
	}

	// генерируем RSA-ключи
	pubPEM, privPEM, err := utils.GenerateRSAKeyPairPEM(2048)
	if err != nil {
		return 0, "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id, err := s.repo.InsertUser(ctx, login, hashedPass, tgLogin, pubPEM)
	if err != nil {
		return 0, "", err
	}

	return id, privPEM, nil
}

func (s *Service) RegenerateRSAForUser(ctx context.Context, login string) (string, error) {
	if err := validateLogin(login); err != nil {
		return "", err
	}

	pubPEM, privPEM, err := utils.GenerateRSAKeyPairPEM(2048)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.repo.UpdateRSAPublicByLogin(ctx, login, pubPEM); err != nil {
		return "", err
	}

	return privPEM, nil
}

func validateLogin(login string) error {
	if len(login) < 3 || len(login) > 50 {
		return errors.New("login must be between 3 and 50 characters")
	}

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.]+$`, login)
	if !matched {
		return errors.New("login contains invalid characters")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}

func validateTelegram(tg string) error {
	if !strings.HasPrefix(tg, "@") {
		return errors.New("telegram login must start with '@'")
	}

	matched, _ := regexp.MatchString(`^@[a-zA-Z0-9_]{5,32}$`, tg)
	if !matched {
		return errors.New("invalid telegram login")
	}
	return nil
}
