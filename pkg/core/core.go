package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

type startFuncType func(context.Context) (<-chan struct{}, <-chan error)
type stopFuncType func(context.Context) error

type Core struct {
	startFuncs []startFuncType
	stopFuncs  []stopFuncType
}

func New(opts ...Options) *Core {
	c := &Core{}

	for _, o := range opts {
		o.New(c)
	}

	return c
}

func hasServiceStarted(ctx context.Context, expectedStart int, startedChans <-chan <-chan struct{}) <-chan struct{} {
	startChan := aggChan[struct{}](startedChans)
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
	errChan := aggChan[error](errChans)
	errorEncountered := make(chan error, 1)

	go func() {
		defer close(errorEncountered)
		for {
			select {
			case err := <-errChan:
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

	c.startFuncs = append(c.startFuncs, WithSignals)

	startedChans := make(chan (<-chan struct{}), len(c.startFuncs))
	errChans := make(chan (<-chan error))

	ctx = log.Logger.WithContext(ctx)

	allStarted := hasServiceStarted(ctx, len(c.startFuncs), startedChans)
	hasErrorChan := hasServiceErrors(ctx, errChans)

	for _, sf := range c.startFuncs {
		// defer close(startedChans)
		// defer close(errChans)
		go func(f startFuncType) {
			started, err := f(ctx)
			errChans <- err
			startedChans <- started
		}(sf)
	}

	// return func() (<-chan struct{}, <-chan error) {
	// 	started := make(chan struct{})

	// 	go func() {
	// 		<-allStarted
	// 		log.Ctx(ctx).Debug().Msg("all backend services started")
	// 		started <- struct{}{}
	// 	}()

	return allStarted, hasErrorChan
	// }
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
}

var ErrSignalReceived = errors.New("signal received")

func WithSignals(ctx context.Context) (<-chan struct{}, <-chan error) {
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
		startedChan <- struct{}{}
		select {
		case <-ctx.Done():
			errChan <- nil
		case s := <-sigc:
			errChan <- fmt.Errorf("%s %w", s, ErrSignalReceived)
			close(errChan)
		}
	}()

	return startedChan, errChan
}

func aggChan[T any](chans <-chan (<-chan T)) <-chan T {
	var wg sync.WaitGroup

	outputChan := make(chan T)
	out := func(cc <-chan T) {
		for c := range cc {
			outputChan <- c
		}
		wg.Done()
	}

	wg.Add(1)
	go func() {
		for s := range chans {
			wg.Add(1)
			go out(s)
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(outputChan)
	}()

	return outputChan
}
