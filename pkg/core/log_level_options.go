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
	log.Info().Str("level", level).Msg("logging level")
	return &logLevelOption{Level: level}
}

func (i *logLevelOption) New(c *Core) {
	l, err := zerolog.ParseLevel(i.Level)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse log level")
	}

	zerolog.SetGlobalLevel(l)
}

func (i *logLevelOption) Start(ctx context.Context) (<-chan struct{}, <-chan error) {
	startedChan := make(chan struct{})
	defer close(startedChan)

	if i.Output != nil {
		log.Logger = log.Logger.Output(i.Output)
	}
	startedChan <- struct{}{}

	return startedChan, nil
}

func (i *logLevelOption) Stop(ctx context.Context) error {
	return nil
}
