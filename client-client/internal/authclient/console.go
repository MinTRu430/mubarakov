package authclient

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func RunConsole(service *Service) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Введите логин: ")
	login, _ := reader.ReadString('\n')
	login = strings.TrimSpace(login)

	codeWord, err := service.StartAuth(login)
	if err != nil {
		return fmt.Errorf("ошибка старта аутентификации: %w", err)
	}
	fmt.Println("Код-слово получено (хэш):", codeWord)
	fmt.Println("📨 Код для Telegram отправлен, проверьте бота.")

	fmt.Print("Введите пароль: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	fmt.Print("Введите код из Telegram: ")
	tgCode, _ := reader.ReadString('\n')
	tgCode = strings.TrimSpace(tgCode)

	message, err := service.FinishAuth(login, password, codeWord, tgCode)
	if err != nil {
		return fmt.Errorf("ошибка завершения аутентификации: %w", err)
	}

	fmt.Println(message)

	return nil
}
