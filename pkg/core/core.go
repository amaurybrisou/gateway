package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type startFuncType func(context.Context) error
type stopFuncType func(context.Context) error

type core struct {
	startFuncs []startFuncType
	stopFuncs  []stopFuncType
}

func Run(ctx context.Context, opts ...Options) (err error) {
	c := &core{}

	for _, o := range opts {
		o.New(c)
	}

	err = c.Start(ctx)
	if err != nil && !errors.Is(err, errSignalReceived) {
		log.Ctx(ctx).Fatal().Caller().Err(err).Msg("shutdown")
	}

	log.Ctx(ctx).Debug().Caller().Err(err).Msg("services closed...")

	return
}

func (c *core) Start(ctx context.Context) (err error) {
	errChan := make(chan error, len(c.startFuncs))

	c.startFuncs = append(c.startFuncs, signals)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.Logger.WithContext(ctx)

	wg := sync.WaitGroup{}
	for _, sf := range c.startFuncs {
		wg.Add(1)
		go func(f startFuncType) {
			defer wg.Done()
			err := f(ctx)
			if err != nil {
				errChan <- err
			}
		}(sf)
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errChan:
	}

	if errors.Is(err, errSignalReceived) {
		log.Ctx(ctx).Debug().Caller().Err(err).Msg("closing services...")
	}

	cancel()

	for _, fn := range c.stopFuncs {
		wg.Add(1)
		go func(f stopFuncType) {
			defer wg.Done()
			err = f(ctx)
			if err != nil {
				log.Ctx(ctx).Fatal().Caller().Err(err).Msg("closing service")
			}
		}(fn)
	}

	wg.Wait()

	return
}

var errSignalReceived = errors.New("signal received")

func signals(ctx context.Context) error {
	sigc := make(chan os.Signal, 1)

	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	select {
	case <-ctx.Done():
		return nil
	case s := <-sigc:
		return fmt.Errorf("%s %w", s, errSignalReceived)
	}
}

func Logger() {
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
	zerolog.TimestampFieldName = "t"
	zerolog.LevelFieldName = "l"
	zerolog.MessageFieldName = "m"
	zerolog.CallerFieldName = "c"

	log.Logger = zerolog.New(os.Stderr).With().Caller().Timestamp().Logger()
}
