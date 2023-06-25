package core

import (
	"context"
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type logLevelOption struct {
	Level  string
	Output io.Writer
}

func WithLogLevel(level string) Options {
	log.Warn().Str("level", level).Msg("logging level")
	return &logLevelOption{Level: level}
}

func (i *logLevelOption) New(c *core) {
	l, err := zerolog.ParseLevel(i.Level)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse log level")
	}

	zerolog.SetGlobalLevel(l)
}

func (i *logLevelOption) Start(ctx context.Context) error {
	if i.Output != nil {
		log.Logger = log.Logger.Output(i.Output)
	}
	return nil
}

func (i *logLevelOption) Stop(ctx context.Context) error {
	return nil
}
