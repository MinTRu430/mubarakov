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

func (r *Repository) InsertUser(ctx context.Context, login, password, tgLogin string) (int, error) {
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
		VALUES ($1, $2, 'stub_rsa', $3)
		RETURNING id
	`
	var id int
	err = r.db.QueryRow(ctx, query, login, password, tgLogin).Scan(&id)
	if err != nil {
		return 0, err
	}
	log.Println("registred user:", login)

	return id, nil
}
