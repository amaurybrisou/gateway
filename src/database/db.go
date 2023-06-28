package database

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Database {
	return &Database{db: db}
}
