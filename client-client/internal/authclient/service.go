package authclient

import (
	"bytes"
	"client-client/internal/utils"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"sync"
)

type Service struct {
	serverURL   string
	sessionKeys sync.Map
	sessionB    sync.Map
}

func NewService(serverURL string) *Service {
	return &Service{serverURL: serverURL}
}

func (s *Service) SaveSessionKey(login, K string) {
	s.sessionKeys.Store(login, K)
}

func (s *Service) GetSessionKey(login string) (string, bool) {
	v, ok := s.sessionKeys.Load(login)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (s *Service) SaveSessionB(login, B string) {
	s.sessionB.Store(login, B)
}

func (s *Service) GetSessionB(login string) (string, bool) {
	v, ok := s.sessionB.Load(login)
	if !ok {
		return "", false
	}
	return v.(string), true
}

type AuthStartRequest struct {
	Login string `json:"login"`
}
type AuthStartResponse struct {
	CodeWord string `json:"code_word"`
}

type AuthFinishRequest struct {
	Login     string `json:"login"`
	MultiHash string `json:"multihash"`
}

type AuthFinishResponse struct {
	Status     string `json:"status"`
	A          string `json:"A"`
	G          string `json:"g"`
	P          string `json:"p"`
	H          string `json:"h"`
	W          string `json:"W"`
	RsaPublicE int    `json:"rsa_public_e"`
	RsaPublicN string `json:"rsa_public_n"`
	Message    string `json:"message"`
}

func (s *Service) StartAuth(login string) (string, error) {
	reqBody, _ := json.Marshal(AuthStartRequest{Login: login})
	resp, err := http.Post(s.serverURL+"/auth/start", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server error: %s", string(b))
	}

	var res AuthStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.CodeWord, nil
}

func (s *Service) FinishAuthWithRSA(login, password, codeWordHash, tgCode, privateKeyPEM string) (string, string, error) {
	hashedPass := utils.HashMD5(password)
	multiHash := utils.HashMD5(hashedPass + codeWordHash + tgCode)

	reqBody, _ := json.Marshal(AuthFinishRequest{
		Login:     login,
		MultiHash: multiHash,
	})

	resp, err := http.Post(s.serverURL+"/auth/finish", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("server error: %s", string(b))
	}

	var res AuthFinishResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}

	ok := utils.VerifyRSASignature(res.A, res.G, res.P, res.H, res.RsaPublicE, res.RsaPublicN)
	if !ok {
		log.Println("RSA signature verification failed!")
		return "", "", fmt.Errorf("RSA подпись не совпала, возможна подмена данных")
	}
	log.Println("RSA signature verified successfully")

	p := new(big.Int)
	g := new(big.Int)
	A := new(big.Int)
	p.SetString(res.P, 10)
	g.SetString(res.G, 10)
	A.SetString(res.A, 10)

	bRand, _ := rand.Int(rand.Reader, p)
	B := new(big.Int).Exp(g, bRand, p)
	K := new(big.Int).Exp(A, bRand, p)

	log.Println("AAA:", A)
	log.Println("bbb:", bRand)
	log.Println("Login:", login)
	log.Println("BBB:", B)
	log.Println("KKK:", K)

	priv, err := utils.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", "", fmt.Errorf("invalid private key: %w", err)
	}

	signature, err := utils.SignBWithPrivateKey(priv, B.String())
	if err != nil {
		return "", "", fmt.Errorf("failed to sign B: %w", err)
	}

	message, err := s.SendBToServerSigned(login, B, signature)
	if err != nil {
		return "", "", fmt.Errorf("error send B to server: %w", err)
	}

	s.SaveSessionKey(login, K.String())

	return message, K.String(), nil
}

func (s *Service) SendBToServerSigned(login string, B *big.Int, signature string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"login":     login,
		"B":         B.String(),
		"signature": signature,
	})
	resp, err := http.Post(s.serverURL+"/chat/start", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server error: %s", string(b))
	}

	var res map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res["message"], nil
}
