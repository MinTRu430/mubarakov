package chat

import (
	"io"
	"log"
	"time"

	"server-client/internal/chatpb"
	"server-client/internal/utils"
)

type GRPCServer struct {
	chatpb.UnimplementedChatServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) Chat(stream chatpb.ChatService_ChatServer) error {
	first, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		log.Println("chat first recv error:", err)
		return err
	}

	login := first.GetLogin()
	encLogin := first.GetCiphertext()
	if login == "" || encLogin == "" {
		log.Println("empty login or ciphertext in first message")
		return nil
	}

	K, ok := s.service.GetSessionKey(login)
	if !ok {
		log.Println("no session key for login:", login)
		return nil
	}

	decLogin, err := utils.DecryptAESFromK(K, encLogin)
	if err != nil || decLogin != login {
		log.Println("login decrypt mismatch")
		return nil
	}

	log.Println("gRPC chat started for login:", login)

	conn := &Connection{
		Out: make(chan *ServerMessage, 16),
	}
	s.service.RegisterConnection(login, conn)
	defer s.service.UnregisterConnection(login)

	// отправитель в gRPC
	go func() {
		for msg := range conn.Out {
			// msg.Text у нас ciphertext
			_ = stream.Send(&chatpb.ServerMessage{
				From:       msg.From,
				Ciphertext: msg.Text,
				Timestamp:  msg.Timestamp.UnixMilli(),
				IsHistory:  false,
			})
		}
	}()

	ctx := stream.Context()

	history, err := s.service.GetHistory(ctx, login)
	if err == nil {
		for _, h := range history {
			ct, err := utils.EncryptAESFromK(K, h.Text)
			if err != nil {
				continue
			}
			_ = stream.Send(&chatpb.ServerMessage{
				From:       h.From,
				Ciphertext: ct,
				Timestamp:  h.Timestamp.UnixMilli(),
				IsHistory:  true,
			})
		}
	}

	for {
		in, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Println("gRPC chat closed by client:", login)
				return nil
			}
			log.Println("chat recv error:", err)
			return err
		}

		ct := in.GetCiphertext()
		ts := in.GetTimestamp()
		if ts == 0 {
			ts = time.Now().UnixMilli()
		}

		plain, err := utils.DecryptAESFromK(K, ct)
		if err != nil {
			log.Println("decrypt message error:", err)
			continue
		}

		log.Printf("message from %s: %s\n", login, plain)

		_ = s.service.SaveMessage(ctx, login, login, plain, ts)

		// если хочешь echo — можно:
		// _ = s.service.SendToClient(login, "[echo] "+plain)
	}
}
