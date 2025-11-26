package chatclient

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"client-client/internal/chatpb"
	"client-client/internal/utils"

	"google.golang.org/grpc"
)

type Client struct {
	addr string
}

func NewClient(addr string) *Client {
	return &Client{addr: addr}
}

func (c *Client) Run(login, K string) error {
	conn, err := grpc.Dial(c.addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := chatpb.NewChatServiceClient(conn)
	ctx := context.Background()
	stream, err := client.Chat(ctx)
	if err != nil {
		return err
	}

	encLogin, err := utils.EncryptAESFromK(K, login)
	if err != nil {
		return err
	}

	if err := stream.Send(&chatpb.ClientMessage{
		Login:      login,
		Ciphertext: encLogin,
	}); err != nil {
		return err
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Println("recv error:", err)
				return
			}

			pt, err := utils.DecryptAESFromK(K, msg.GetCiphertext())
			if err != nil {
				log.Println("decrypt err:", err)
				continue
			}

			prefix := ""
			if msg.GetIsHistory() {
				prefix = "[history] "
			}

			fmt.Printf("%s%s: %s\n", prefix, msg.GetFrom(), pt)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		ct, err := utils.EncryptAESFromK(K, text)
		if err != nil {
			log.Println("encrypt err:", err)
			continue
		}

		if err := stream.Send(&chatpb.ClientMessage{
			Login:      login,
			Ciphertext: ct,
			Timestamp:  time.Now().UnixMilli(),
		}); err != nil {
			return err
		}
	}
}
