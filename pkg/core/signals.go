package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

type CoreSignals struct {
	done chan struct{}
}

func WithSignals() Options {
	return CoreSignals{done: make(chan struct{})}
}

func (s CoreSignals) New(c *Core) {
	c.startFuncs = append(c.startFuncs, s.Start)
	c.stopFuncs = append(c.stopFuncs, s.Stop)
}

func (s CoreSignals) Stop(ctx context.Context) error {
	close(s.done)
	log.Ctx(ctx).Debug().Msg("signal handler stopped")
	return nil
}

func (s CoreSignals) Start(ctx context.Context) (<-chan struct{}, <-chan error) {
	sigc := make(chan os.Signal, 1)

	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	errChan := make(chan error)
	startedChan := make(chan struct{})
	go func() {
		defer close(startedChan)
		defer close(errChan)
		startedChan <- struct{}{}
		log.Ctx(ctx).Debug().Msg("signal handler ready")
		select {
		case <-s.done:
			return
		case <-ctx.Done():
			errChan <- nil
		case s := <-sigc:
			errChan <- fmt.Errorf("%s %w", s, ErrSignalReceived)
		}
	}()

	return startedChan, errChan
}
