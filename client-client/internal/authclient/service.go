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
)

type Service struct {
	serverURL string
}

func NewService(serverURL string) *Service {
	return &Service{serverURL: serverURL}
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

func (s *Service) FinishAuth(login, password, codeWordHash, tgCode string) (string, error) {
	hashedPass := utils.HashMD5(password)
	multiHash := utils.HashMD5(hashedPass + codeWordHash + tgCode)

	reqBody, _ := json.Marshal(AuthFinishRequest{
		Login:     login,
		MultiHash: multiHash,
	})

	resp, err := http.Post(s.serverURL+"/auth/finish", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server error: %s", string(b))
	}

	var res AuthFinishResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	// log.Printf("RSA: E=%d, N=%s\n", res.RsaPublicE, res.RsaPublicN)
	// log.Printf("Diffie-Hellman A=%s, G=%s, P=%s\n", res.A, res.G, res.P)
	// log.Printf("Hash h=%s, Sign w=%s\n", res.H, res.W)

	ok := utils.VerifyRSASignature(res.A, res.G, res.P, res.H, res.RsaPublicE, res.RsaPublicN)
	if !ok {
		log.Println("RSA signature verification failed!")
		return "", fmt.Errorf("RSA подпись не совпала, возможна подмена данных")
	}
	log.Println("RSA signature verified successfully")

	p := new(big.Int)
	g := new(big.Int)
	A := new(big.Int)
	p.SetString(res.P, 10)
	g.SetString(res.G, 10)
	A.SetString(res.A, 10)

	b, _ := rand.Int(rand.Reader, p)
	B := new(big.Int).Exp(g, b, p)
	K := new(big.Int).Exp(A, b, p)

	log.Println("Login:", login)
	log.Println("BBB:", B)
	log.Println("KKK:", K)

	message, err := s.SendBToServer(login, B)
	if err != nil {
		return "", fmt.Errorf("error send B to server")
	}

	return message, nil
}

func (s *Service) SendBToServer(login string, B *big.Int) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"login": login,
		"B":     B.String(),
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
