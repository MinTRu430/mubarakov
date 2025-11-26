package auth

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpdateCodeWord(ctx context.Context, login, codeWord string) error {
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	_, err := r.db.Exec(ctx, "UPDATE users SET code_word=$1, code_word_live_time=$2 WHERE login=$3", codeWord, expiresAt, login)
	return err
}

type PassCodeWord struct {
	HashedPass       string
	CodeWord         string
	TgCode           string
	CodeWordLifeTime time.Time
}

func (r *Repository) GetUser(ctx context.Context, login string) (PassCodeWord, error) {
	var pcw PassCodeWord
	err := r.db.QueryRow(ctx, "SELECT password, code_word, code_word_live_time, tg_code FROM users WHERE login = $1",
		login).Scan(&pcw.HashedPass, &pcw.CodeWord, &pcw.CodeWordLifeTime, &pcw.TgCode)

	return pcw, err
}

func (r *Repository) UpdateTelegramCode(ctx context.Context, login, tgCode string) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET tg_code = $1 WHERE login = $2", tgCode, login)
	return err
}

func (r *Repository) GetChatID(ctx context.Context, login string) (int64, error) {
	var chatID int64
	err := r.db.QueryRow(ctx, "SELECT tg_chat_id FROM users WHERE login = $1", login).Scan(&chatID)
	return chatID, err
}

func (r *Repository) SaveDiffe(ctx context.Context, login, aValue, aPrivate, pValue string) error {
	query := `
		INSERT INTO dh_temp (login, a_value, a_private, p_value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (login)
		DO UPDATE SET a_value = EXCLUDED.a_value, a_private = EXCLUDED.a_private, p_value = EXCLUDED.p_value
	`
	_, err := r.db.Exec(ctx, query, login, aValue, aPrivate, pValue)
	if err != nil {
		log.Printf("failed to save Diffie-Hellman params for %s: %v", login, err)
		return err
	}

	log.Printf("Diffie-Hellman params saved for user: %s", login)
	return nil
}
