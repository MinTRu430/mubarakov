package chat

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type DHParams struct {
	A     string
	P     string
	APriv string
}

func (r *Repository) GetDHParams(ctx context.Context, login string) (DHParams, error) {
	var params DHParams
	err := r.db.QueryRow(ctx, `
		SELECT a_value, p_value, a_private
		FROM dh_temp
		WHERE login = $1
	`, login).Scan(&params.A, &params.P, &params.APriv)
	return params, err
}

func (r *Repository) GetUserRSAPublicKey(ctx context.Context, login string) (string, error) {
	var pub string
	err := r.db.QueryRow(ctx, `
		SELECT rsa_signature
		FROM users
		WHERE login = $1
	`, login).Scan(&pub)
	return pub, err
}
