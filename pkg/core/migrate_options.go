package core

import (
	"context"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

type Migrate struct{ path, url string }

func WithMigrate(path, url string) Options {
	m := Migrate{path: path, url: url}

	return m
}

func (m Migrate) New(c *core) {
	c.startFuncs = append(c.startFuncs, m.Start)
}

func (m Migrate) Start(ctx context.Context) error {
	log.Ctx(ctx).Info().Msg("migrating database")
	mig, err := migrate.New(m.path, m.url)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("migrating database")
		return err
	}

	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Ctx(ctx).Error().Err(err).Msg("migrating database")
		return err
	}

	return nil
}

func (m Migrate) Stop(_ context.Context) error {
	return nil
}
