package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"server-client/internal/auth"
	"server-client/internal/bot"
	"server-client/internal/chat"
	"server-client/internal/chatpb"
	"server-client/internal/reg"
	"server-client/internal/utils"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	log.Println("mubarakov start!")

	db, err := utils.NewPostgresPool(context.Background())
	if err != nil {
		log.Fatalln("DB connection failed:", err)
	}
	defer db.Close()
	log.Println("database connected")

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalln("Redis connection failed:", err)
	}
	log.Println("redis connected")

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
	chatService := chat.NewService(chatRepo, rdb)
	chatHandler := chat.NewHandler(chatService)

	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen for gRPC: %v", err)
		}

		grpcServer := grpc.NewServer()
		chatGRPC := chat.NewGRPCServer(chatService)
		chatpb.RegisterChatServiceServer(grpcServer, chatGRPC)

		log.Println("gRPC server started on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// static
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	//Api
	http.HandleFunc("/register", handler.Register)
	http.HandleFunc("/reg/rsa/regenerate", handler.RegenerateRSA)
	http.HandleFunc("/auth/start", authHandler.Start)
	http.HandleFunc("/auth/finish", authHandler.Finish)
	http.HandleFunc("/chat/start", chatHandler.Start)
	http.HandleFunc("/chat/server/send", chatHandler.ServerSend)
	http.HandleFunc("/chat/active", chatHandler.ActiveChats)
	http.HandleFunc("/chat/history", chatHandler.History)
	http.HandleFunc("/ws/admin", chatHandler.AdminWS)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
