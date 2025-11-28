package vote

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Election struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	RSAM        string    `json:"rsa_m"`
	RSAE        int       `json:"rsa_e"`
	RSAD        string    `json:"rsa_d"`
}

type ElectionResult struct {
	ElectionID   int
	YesVotes     int
	NoVotes      int
	AbstainVotes int
	R            string
	TotalVoters  int
	F            string
	Q            string
	CreatedAt    time.Time
}

// ===== elections =====

func (r *Repository) CreateElection(ctx context.Context, title, desc, m string, e int, d string) (int, error) {
	const q = `
		INSERT INTO elections (title, description, status, rsa_m, rsa_e, rsa_d)
		VALUES ($1, $2, 'created', $3, $4, $5)
		RETURNING id
	`
	var id int
	err := r.db.QueryRow(ctx, q, title, desc, m, e, d).Scan(&id)
	return id, err
}

func (r *Repository) GetElection(ctx context.Context, id int) (Election, error) {
	const q = `
		SELECT id, title, description, status, created_at, rsa_m, rsa_e, rsa_d
		FROM elections
		WHERE id = $1
	`
	var e Election
	err := r.db.QueryRow(ctx, q, id).Scan(
		&e.ID, &e.Title, &e.Description, &e.Status, &e.CreatedAt,
		&e.RSAM, &e.RSAE, &e.RSAD,
	)
	return e, err
}

func (r *Repository) SetElectionStatus(ctx context.Context, id int, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE elections SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *Repository) ListElections(ctx context.Context) ([]Election, error) {
	const q = `
		SELECT id, title, description, status, created_at, rsa_m, rsa_e, rsa_d
		FROM elections
		ORDER BY id DESC
	`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Election
	for rows.Next() {
		var e Election
		if err := rows.Scan(
			&e.ID, &e.Title, &e.Description, &e.Status, &e.CreatedAt,
			&e.RSAM, &e.RSAE, &e.RSAD,
		); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, rows.Err()
}

// ===== users / votes =====

func (r *Repository) GetUserIDByLogin(ctx context.Context, login string) (int, error) {
	const q = `SELECT id FROM users WHERE login = $1`
	var id int
	err := r.db.QueryRow(ctx, q, login).Scan(&id)
	return id, err
}

func (r *Repository) HasVoted(ctx context.Context, electionID, userID int) (bool, error) {
	const q = `SELECT 1 FROM election_votes WHERE election_id = $1 AND user_id = $2`
	var dummy int
	err := r.db.QueryRow(ctx, q, electionID, userID).Scan(&dummy)
	if err != nil {
		// pgx.ErrNoRows -> false, nil
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) InsertVote(ctx context.Context, electionID, userID int, ciphertext string) error {
	const q = `
		INSERT INTO election_votes (election_id, user_id, ciphertext)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, q, electionID, userID, ciphertext)
	return err
}

func (r *Repository) ListCiphertexts(ctx context.Context, electionID int) ([]string, error) {
	const q = `
		SELECT ciphertext
		FROM election_votes
		WHERE election_id = $1
		ORDER BY id
	`
	rows, err := r.db.Query(ctx, q, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, rows.Err()
}

// ===== results =====

func (r *Repository) SaveResult(ctx context.Context, res ElectionResult) error {
	const q = `
		INSERT INTO election_results
		(election_id, yes_votes, no_votes, abstain_votes, R, total_voters, F, Q)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (election_id) DO UPDATE SET
		  yes_votes = EXCLUDED.yes_votes,
		  no_votes = EXCLUDED.no_votes,
		  abstain_votes = EXCLUDED.abstain_votes,
		  R = EXCLUDED.R,
		  total_voters = EXCLUDED.total_voters,
		  F = EXCLUDED.F,
		  Q = EXCLUDED.Q,
		  created_at = now()
	`
	_, err := r.db.Exec(ctx, q,
		res.ElectionID,
		res.YesVotes,
		res.NoVotes,
		res.AbstainVotes,
		res.R,
		res.TotalVoters,
		res.F,
		res.Q,
	)
	return err
}

func (r *Repository) GetResult(ctx context.Context, electionID int) (ElectionResult, error) {
	const q = `
		SELECT election_id, yes_votes, no_votes, abstain_votes, R, total_voters, F, Q, created_at
		FROM election_results
		WHERE election_id = $1
	`
	var res ElectionResult
	err := r.db.QueryRow(ctx, q, electionID).Scan(
		&res.ElectionID,
		&res.YesVotes,
		&res.NoVotes,
		&res.AbstainVotes,
		&res.R,
		&res.TotalVoters,
		&res.F,
		&res.Q,
		&res.CreatedAt,
	)
	return res, err
}
