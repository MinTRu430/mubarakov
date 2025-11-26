package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"server-client/internal/utils"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo        *Repository
	redis       *redis.Client
	sessionKeys sync.Map
	connections sync.Map
	wsMu        sync.Mutex
	wsConns     map[*WSConn]struct{}
}

type Connection struct {
	Out chan *ServerMessage
}

type ServerMessage struct {
	Login     string    `json:"login"`
	From      string    `json:"from"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type WSConn struct {
	Send func([]byte) error
}

func NewService(repo *Repository, rdb *redis.Client) *Service {
	return &Service{
		repo:    repo,
		redis:   rdb,
		wsConns: make(map[*WSConn]struct{}),
	}
}

func (s *Service) SetSessionKey(login, K string) {
	s.sessionKeys.Store(login, K)
}

func (s *Service) GetSessionKey(login string) (string, bool) {
	v, ok := s.sessionKeys.Load(login)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (s *Service) ComputeSharedKey(ctx context.Context, login, B string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	params, err := s.repo.GetDHParams(ctx, login)
	if err != nil {
		return "", err
	}

	Bnum := new(big.Int)
	Bnum.SetString(B, 10)

	P := new(big.Int)
	P.SetString(params.P, 10)

	an := new(big.Int)
	an.SetString(params.APriv, 10)

	K := new(big.Int).Exp(Bnum, an, P)

	return K.String(), nil
}

func redisKey(login string) string {
	return "chat:" + login
}

func (s *Service) SaveMessage(ctx context.Context, login, from, text string, ts int64) error {
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	msg := ServerMessage{
		Login:     login,
		From:      from,
		Text:      text,
		Timestamp: time.UnixMilli(ts),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if s.redis != nil {
		if err := s.redis.RPush(ctx, redisKey(login), b).Err(); err != nil {
			return err
		}
	}

	s.broadcastWS(msg)

	return nil
}

func (s *Service) GetHistory(ctx context.Context, login string) ([]ServerMessage, error) {
	if s.redis == nil {
		return nil, nil
	}

	vals, err := s.redis.LRange(ctx, redisKey(login), 0, -1).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	res := make([]ServerMessage, 0, len(vals))
	for _, v := range vals {
		var m ServerMessage
		if err := json.Unmarshal([]byte(v), &m); err == nil {
			res = append(res, m)
		}
	}
	return res, nil
}

func (s *Service) RegisterConnection(login string, conn *Connection) {
	if old, ok := s.connections.Load(login); ok {
		if oc, ok2 := old.(*Connection); ok2 {
			close(oc.Out)
		}
	}
	s.connections.Store(login, conn)
}

func (s *Service) UnregisterConnection(login string) {
	if old, ok := s.connections.LoadAndDelete(login); ok {
		if oc, ok2 := old.(*Connection); ok2 {
			close(oc.Out)
		}
	}
}

func (s *Service) RegisterWS(c *WSConn) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	s.wsConns[c] = struct{}{}
}

func (s *Service) UnregisterWS(c *WSConn) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	delete(s.wsConns, c)
}

func (s *Service) broadcastWS(msg ServerMessage) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}

	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	for c := range s.wsConns {
		_ = c.Send(b)
	}
}

func (s *Service) SendToClient(login, plaintext string) error {
	K, ok := s.GetSessionKey(login)
	if !ok {
		return errors.New("no session key for login")
	}

	v, ok := s.connections.Load(login)
	if !ok {
		return errors.New("no active connection for login")
	}
	conn, ok := v.(*Connection)
	if !ok {
		return errors.New("invalid connection type")
	}

	ct, err := utils.EncryptAESFromK(K, plaintext)
	if err != nil {
		return err
	}

	msg := &ServerMessage{
		Login:     login,
		From:      "server",
		Text:      plaintext,
		Timestamp: time.Now(),
	}

	_ = s.SaveMessage(context.Background(), login, msg.From, msg.Text, msg.Timestamp.UnixMilli())

	select {
	case conn.Out <- &ServerMessage{
		Login:     login,
		From:      "server",
		Text:      ct,
		Timestamp: msg.Timestamp,
	}:
		return nil
	default:
		return errors.New("client outbound channel is full or closed")
	}
}

func (s *Service) ListActiveLogins() []string {
	res := make([]string, 0)
	s.connections.Range(func(key, value any) bool {
		if login, ok := key.(string); ok {
			res = append(res, login)
		}
		return true
	})
	return res
}

func (s *Service) VerifyBSignature(ctx context.Context, login, B, signature string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pubPEM, err := s.repo.GetUserRSAPublicKey(ctx, login)
	if err != nil {
		return fmt.Errorf("failed to get rsa public key: %w", err)
	}

	ok, err := utils.VerifyRSASignatureFromPEM(pubPEM, B, signature)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid RSA signature for B")
	}

	return nil
}
