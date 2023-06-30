package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func NewPostgres(ctx context.Context, u, p, h string, port int, d, s string) *pgxpool.Pool {
	db, err := pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", u, p, h, port, d, s))
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}
	return db
}
