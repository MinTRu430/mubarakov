package reg

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

var ErrUserAlreadyExists = errors.New("user with this login already exists")

func (r *Repository) InsertUser(ctx context.Context, login, password, tgLogin, rsaPublic string) (int, error) {
	var existingID int
	err := r.db.QueryRow(ctx, "SELECT id FROM users WHERE login = $1", login).Scan(&existingID)
	if err == nil {
		return 0, ErrUserAlreadyExists
	}
	if err != pgx.ErrNoRows {
		return 0, err
	}

	query := `
		INSERT INTO users (login, password, rsa_signature, tg_login)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id int
	err = r.db.QueryRow(ctx, query, login, password, rsaPublic, tgLogin).Scan(&id)
	if err != nil {
		return 0, err
	}
	log.Println("registered user:", login)

	return id, nil
}

func (r *Repository) UpdateRSAPublicByLogin(ctx context.Context, login, rsaPublic string) error {
	cmd, err := r.db.Exec(ctx,
		`UPDATE users SET rsa_signature = $1 WHERE login = $2`,
		rsaPublic, login,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}
