package main

import (
	"context"
	"log"
	"net/http"
	"server-client/internal/auth"
	"server-client/internal/bot"
	"server-client/internal/chat"
	"server-client/internal/reg"
	"server-client/internal/utils"
)

func main() {
	log.Println("mubarakov start!")

	db, err := utils.NewPostgresPool(context.Background())
	if err != nil {
		log.Fatalln("DB connection failed:", err)
	}
	defer db.Close()
	log.Println("database connected")

	tgToken := "" // убрать в env

	botRepo := bot.NewRepository(db)
	tgBot, err := bot.NewBot(tgToken, botRepo)
	if err != nil {
		log.Fatalf("failed to init telegram bot: %v", err)
	}
	log.Println("telegram bot started")

	go tgBot.Start()

	//Reg
	repo := reg.NewRepository(db)
	service := reg.NewService(repo)
	handler := reg.NewHandler(service)

	//Auth
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, tgBot)
	authHandler := auth.NewHandler(authService)

	//Auth
	chatRepo := chat.NewRepository(db)
	chatService := chat.NewService(chatRepo)
	chatHandler := chat.NewHandler(chatService)

	// static
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	//Api
	http.HandleFunc("/register", handler.Register)
	http.HandleFunc("/auth/start", authHandler.Start)
	http.HandleFunc("/auth/finish", authHandler.Finish)
	http.HandleFunc("/chat/start", chatHandler.Start)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
