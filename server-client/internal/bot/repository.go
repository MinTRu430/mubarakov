package bot

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveChatID(ctx context.Context, username string, chatID int64) error {
	fmt.Println(username, chatID)
	un := "@" + username
	fmt.Println(un, username, chatID)
	_, err := r.db.Exec(ctx,
		"UPDATE users SET tg_chat_id = $1 WHERE tg_login = $2",
		chatID, un)
	return err
}
