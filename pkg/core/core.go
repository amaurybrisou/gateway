package core

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type startFuncType func(context.Context) (<-chan struct{}, <-chan error)
type stopFuncType func(context.Context) error

type Core struct {
	startFuncs []startFuncType
	stopFuncs  []stopFuncType
	cleanup    func()
}

func New(opts ...Options) *Core {
	c := &Core{}

	for _, o := range opts {
		o.New(c)
	}

	return c
}

func hasServiceStarted(ctx context.Context, expectedStart int, startedChans <-chan <-chan struct{}) <-chan struct{} {
	startChan := aggChan[struct{}](ctx, startedChans)
	done := make(chan struct{}, 1)

	go func() {
		defer close(done)
		for expectedStart > 0 {
			select {
			case <-ctx.Done():
				return
			case <-startChan:
				expectedStart--
			default:
				time.Sleep(time.Second)
				log.Ctx(ctx).Debug().Msg("waiting to start")
			}
		}
		done <- struct{}{}
	}()

	return done
}

func hasServiceErrors(ctx context.Context, errChans <-chan <-chan error) <-chan error {
	errChan := aggChan[error](ctx, errChans)
	errorEncountered := make(chan error, 1)

	go func() {
		defer close(errorEncountered)
		for {
			select {
			case err, ok := <-errChan:
				if !ok {
					return
				}
				if err != nil {
					// log.Ctx(ctx).Debug().Msg("error starting")
					errorEncountered <- err
					return
				}
			case <-ctx.Done():
				err := ctx.Err()
				if errors.Is(err, context.Canceled) {
					return
				}
				errorEncountered <- err
				return
			}
		}
	}()

	return errorEncountered
}

func (c *Core) Start(ctx context.Context) (<-chan struct{}, <-chan error) {
	log.Ctx(ctx).Info().Msg("starting backend")

	startedChans := make(chan (<-chan struct{}), len(c.startFuncs))
	errChans := make(chan (<-chan error))

	allStarted := hasServiceStarted(ctx, len(c.startFuncs), startedChans)
	hasErrorChan := hasServiceErrors(ctx, errChans)

	for _, sf := range c.startFuncs {
		go func(f startFuncType) {
			started, err := f(ctx)
			errChans <- err
			startedChans <- started
		}(sf)
	}

	c.cleanup = func() {
		close(startedChans)
		close(errChans)
	}

	return allStarted, hasErrorChan
}

func (c *Core) Shutdown(ctx context.Context) {
	log.Ctx(ctx).Debug().Msg("closing services")
	wg := sync.WaitGroup{}
	for _, fn := range c.stopFuncs {
		wg.Add(1)
		go func(f stopFuncType) {
			defer wg.Done()
			err := f(ctx)
			if err != nil {
				log.Ctx(ctx).Fatal().Caller().Err(err).Msg("closing service")
			}
		}(fn)
	}

	wg.Wait()

	if c.cleanup != nil {
		c.cleanup()
	}
}

var ErrSignalReceived = errors.New("signal received")

func aggChan[T any](ctx context.Context, chans <-chan (<-chan T)) <-chan T {
	var wg sync.WaitGroup

	outputChan := make(chan T)
	out := func(cc <-chan T) {
		defer wg.Done()
		select {
		case c, ok := <-cc: // cc must be closed explicitly by its creator.
			if !ok {
				return
			}
			outputChan <- c
		case <-ctx.Done():
			return
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case s, ok := <-chans:
				if !ok {
					return
				}
				wg.Add(1)
				go out(s)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(outputChan)
	}()

	return outputChan
}
