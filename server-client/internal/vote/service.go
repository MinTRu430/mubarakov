package vote

import (
	"context"
	"errors"
	"math/big"
	"time"

	"server-client/internal/utils"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

var (
	ErrElectionNotOpen   = errors.New("election not open")
	ErrElectionClosed    = errors.New("election closed")
	ErrAlreadyVoted      = errors.New("user has already voted")
	ErrUserNotFound      = errors.New("user not found")
	ErrElectionNotClosed = errors.New("election must be closed to count")
)

type PublicElectionInfo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	M           string `json:"m"`
	E           int    `json:"e"`
}

type PublicElectionResult struct {
	ElectionID   int      `json:"election_id"`
	Title        string   `json:"title"`
	YesVotes     int      `json:"yes_votes"`
	NoVotes      int      `json:"no_votes"`
	AbstainVotes int      `json:"abstain_votes"`
	R            string   `json:"R"`
	TotalVoters  int      `json:"total_voters"`
	F            string   `json:"F"`
	Q            string   `json:"Q"`
	Ciphertexts  []string `json:"ciphertexts"`
	M            string   `json:"m"`
	E            int      `json:"e"`
}

// ===== Админ =====

func (s *Service) CreateElection(ctx context.Context, title, desc string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	mStr, eInt, dStr, err := utils.GenerateRSAForElection(2048)
	if err != nil {
		return 0, err
	}

	return s.repo.CreateElection(ctx, title, desc, mStr, eInt, dStr)
}

func (s *Service) OpenElection(ctx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	e, err := s.repo.GetElection(ctx, id)
	if err != nil {
		return err
	}
	if e.Status == "open" {
		return nil
	}
	return s.repo.SetElectionStatus(ctx, id, "open")
}

func (s *Service) CloseElection(ctx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	e, err := s.repo.GetElection(ctx, id)
	if err != nil {
		return err
	}
	if e.Status == "closed" || e.Status == "counted" {
		return nil
	}
	return s.repo.SetElectionStatus(ctx, id, "closed")
}

func (s *Service) ListElections(ctx context.Context) ([]Election, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.repo.ListElections(ctx)
}

func (s *Service) GetElectionPublicInfo(ctx context.Context, id int) (PublicElectionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	e, err := s.repo.GetElection(ctx, id)
	if err != nil {
		return PublicElectionInfo{}, err
	}

	return PublicElectionInfo{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		Status:      e.Status,
		M:           e.RSAM,
		E:           e.RSAE,
	}, nil
}

// ===== Голосование пользователем =====

func (s *Service) SubmitVote(ctx context.Context, electionID int, login, ciphertext string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	e, err := s.repo.GetElection(ctx, electionID)
	if err != nil {
		return err
	}

	if e.Status != "open" {
		return ErrElectionNotOpen
	}

	userID, err := s.repo.GetUserIDByLogin(ctx, login)
	if err != nil {
		return ErrUserNotFound
	}

	voted, err := s.repo.HasVoted(ctx, electionID, userID)
	if err != nil {
		return err
	}
	if voted {
		return ErrAlreadyVoted
	}

	return s.repo.InsertVote(ctx, electionID, userID, ciphertext)
}

// ===== Подсчёт =====

func (s *Service) CountVotes(ctx context.Context, electionID int) (ElectionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	e, err := s.repo.GetElection(ctx, electionID)
	if err != nil {
		return ElectionResult{}, err
	}
	if e.Status != "closed" && e.Status != "counted" {
		return ElectionResult{}, ErrElectionNotClosed
	}

	cts, err := s.repo.ListCiphertexts(ctx, electionID)
	if err != nil {
		return ElectionResult{}, err
	}
	if len(cts) == 0 {
		// никого, всё по нулям
		res := ElectionResult{
			ElectionID:   electionID,
			YesVotes:     0,
			NoVotes:      0,
			AbstainVotes: 0,
			R:            "1",
			TotalVoters:  0,
			F:            "1",
			Q:            "1",
		}
		if err := s.repo.SaveResult(ctx, res); err != nil {
			return ElectionResult{}, err
		}
		_ = s.repo.SetElectionStatus(ctx, electionID, "counted")
		return res, nil
	}

	// F = prod(f_i) mod m
	FStr, err := utils.BigIntProdMod(cts, e.RSAM)
	if err != nil {
		return ElectionResult{}, err
	}

	// Q = F^d mod m
	QStr, err := utils.RSADecodeCipher(FStr, e.RSAM, e.RSAD)
	if err != nil {
		return ElectionResult{}, err
	}

	Qbig := new(big.Int)
	if _, ok := Qbig.SetString(QStr, 10); !ok {
		return ElectionResult{}, utils.ErrParseBigInt("Q")
	}

	two := big.NewInt(2)
	three := big.NewInt(3)
	zero := big.NewInt(0)

	r := 0 // yes
	p := 0 // no

	tmp := new(big.Int)

	// выносим степени 2
	for {
		tmp.Mod(Qbig, two)
		if tmp.Cmp(zero) != 0 {
			break
		}
		Qbig.Div(Qbig, two)
		r++
	}

	// выносим степени 3
	for {
		tmp.Mod(Qbig, three)
		if tmp.Cmp(zero) != 0 {
			break
		}
		Qbig.Div(Qbig, three)
		p++
	}

	RStr := Qbig.String()
	total := len(cts)
	abstain := total - (r + p)
	if abstain < 0 {
		abstain = 0 // на всякий пожарный
	}

	res := ElectionResult{
		ElectionID:   electionID,
		YesVotes:     r,
		NoVotes:      p,
		AbstainVotes: abstain,
		R:            RStr,
		TotalVoters:  total,
		F:            FStr,
		Q:            QStr,
	}

	if err := s.repo.SaveResult(ctx, res); err != nil {
		return ElectionResult{}, err
	}

	_ = s.repo.SetElectionStatus(ctx, electionID, "counted")

	return res, nil
}

func (s *Service) GetPublicResult(ctx context.Context, electionID int) (PublicElectionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	e, err := s.repo.GetElection(ctx, electionID)
	if err != nil {
		return PublicElectionResult{}, err
	}
	res, err := s.repo.GetResult(ctx, electionID)
	if err != nil {
		return PublicElectionResult{}, err
	}
	cts, err := s.repo.ListCiphertexts(ctx, electionID)
	if err != nil {
		return PublicElectionResult{}, err
	}

	return PublicElectionResult{
		ElectionID:   res.ElectionID,
		Title:        e.Title,
		YesVotes:     res.YesVotes,
		NoVotes:      res.NoVotes,
		AbstainVotes: res.AbstainVotes,
		R:            res.R,
		TotalVoters:  res.TotalVoters,
		F:            res.F,
		Q:            res.Q,
		Ciphertexts:  cts,
		M:            e.RSAM,
		E:            e.RSAE,
	}, nil
}
