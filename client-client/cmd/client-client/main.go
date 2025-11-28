package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"client-client/internal/authclient"
	"client-client/internal/chatpb"
	"client-client/internal/utils"
	"client-client/internal/voteclient"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	serverURL := "http://localhost:8080"

	authSvc := authclient.NewService(serverURL)
	authHandler := authclient.NewHandler(authSvc)

	voteSvc := voteclient.NewService(serverURL, authSvc)
	voteHandler := voteclient.NewHandler(voteSvc)

	http.HandleFunc("/start-auth", authHandler.StartAuth)
	http.HandleFunc("/finish-auth-web", authHandler.FinishAuthWeb)

	http.HandleFunc("/ws/chat", func(w http.ResponseWriter, r *http.Request) {
		chatWSHandler(w, r, authSvc)
	})

	http.HandleFunc("/vote/info", voteHandler.Info)
	http.HandleFunc("/vote/submit", voteHandler.Submit)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("Client started on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func chatWSHandler(w http.ResponseWriter, r *http.Request, service *authclient.Service) {
	login := r.URL.Query().Get("login")
	if login == "" {
		http.Error(w, "login required", http.StatusBadRequest)
		return
	}

	K, ok := service.GetSessionKey(login)
	if !ok {
		http.Error(w, "no session for this login (not authenticated?)", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}
	defer conn.Close()

	grpcConn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Println("grpc dial error:", err)
		return
	}
	defer grpcConn.Close()

	client := chatpb.NewChatServiceClient(grpcConn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Chat(ctx)
	if err != nil {
		log.Println("grpc Chat error:", err)
		return
	}

	encLogin, err := utils.EncryptAESFromK(K, login)
	if err != nil {
		log.Println("encrypt login error:", err)
		return
	}

	if err := stream.Send(&chatpb.ClientMessage{
		Login:      login,
		Ciphertext: encLogin,
	}); err != nil {
		log.Println("send first login message error:", err)
		return
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Println("grpc recv error:", err)
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				cancel()
				return
			}

			pt, err := utils.DecryptAESFromK(K, msg.GetCiphertext())
			if err != nil {
				log.Println("decrypt chat msg error:", err)
				continue
			}

			out := struct {
				From      string `json:"from"`
				Text      string `json:"text"`
				Timestamp int64  `json:"timestamp"`
				IsHistory bool   `json:"is_history"`
			}{
				From:      msg.GetFrom(),
				Text:      pt,
				Timestamp: msg.GetTimestamp(),
				IsHistory: msg.GetIsHistory(),
			}

			if err := conn.WriteJSON(out); err != nil {
				log.Println("ws write error:", err)
				cancel()
				return
			}
		}
	}()

	for {
		var in struct {
			Text string `json:"text"`
		}
		if err := conn.ReadJSON(&in); err != nil {
			log.Println("ws read error:", err)
			cancel()
			return
		}

		if in.Text == "" {
			continue
		}

		ct, err := utils.EncryptAESFromK(K, in.Text)
		if err != nil {
			log.Println("encrypt client msg error:", err)
			continue
		}

		if err := stream.Send(&chatpb.ClientMessage{
			Login:      login,
			Ciphertext: ct,
			Timestamp:  time.Now().UnixMilli(),
		}); err != nil {
			log.Println("grpc send error:", err)
			cancel()
			return
		}
	}
}
