package voteclient

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"client-client/internal/authclient"
)

type Service struct {
	serverURL   string
	authService *authclient.Service
}

func NewService(serverURL string, auth *authclient.Service) *Service {
	return &Service{
		serverURL:   serverURL,
		authService: auth,
	}
}

type PublicElectionInfo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	M           string `json:"m"` // RSA modulus (10-ричная строка)
	E           int    `json:"e"`
}

type submitVoteServerRequest struct {
	ElectionID int    `json:"election_id"`
	Login      string `json:"login"`
	Ciphertext string `json:"ciphertext"`
}

type submitVoteServerResponse struct {
	Status string `json:"status"`
}

func (s *Service) GetElectionInfo(electionID int) (*PublicElectionInfo, error) {
	url := fmt.Sprintf("%s/vote/info?election_id=%d", s.serverURL, electionID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s", string(body))
	}

	var info PublicElectionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	return &info, nil
}

func (s *Service) SubmitVote(electionID int, choice string) error {
	login := s.authService.GetCurrentLogin()
	if login == "" {
		return fmt.Errorf("not authenticated (login is empty)")
	}

	info, err := s.GetElectionInfo(electionID)
	if err != nil {
		return fmt.Errorf("get election info: %w", err)
	}

	if info.Status != "open" {
		return fmt.Errorf("election is not open (status = %s)", info.Status)
	}

	var bVal int64
	switch choice {
	case "yes":
		bVal = 2
	case "no":
		bVal = 3
	case "abstain":
		bVal = 1
	default:
		return fmt.Errorf("unknown choice: %s", choice)
	}

	// b_i
	b := big.NewInt(bVal)

	q, err := rand.Prime(rand.Reader, 64) // 64 бит более чем достаточно
	if err != nil {
		return fmt.Errorf("failed to generate prime: %w", err)
	}

	// t_i = b_i * q_i
	t := new(big.Int).Mul(b, q)

	// f_i = t_i^e mod m
	m := new(big.Int)
	if _, ok := m.SetString(info.M, 10); !ok {
		return fmt.Errorf("invalid RSA modulus M")
	}

	eInt := big.NewInt(int64(info.E))
	f := new(big.Int).Exp(t, eInt, m)

	reqBody, _ := json.Marshal(submitVoteServerRequest{
		ElectionID: electionID,
		Login:      login,
		Ciphertext: f.String(),
	})

	url := s.serverURL + "/vote/submit"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("submit vote request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var srvResp submitVoteServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&srvResp); err != nil {
		return fmt.Errorf("decode server response error: %w", err)
	}

	if srvResp.Status != "ok" {
		return fmt.Errorf("vote not accepted: status=%s", srvResp.Status)
	}

	return nil
}
