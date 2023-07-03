package core

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type heartbeatOption struct {
	i    time.Duration
	done chan struct{}
}

func WithHeartBeat(i time.Duration) Options {
	tmp := &heartbeatOption{i: i, done: make(chan struct{})}
	return tmp
}

func (h *heartbeatOption) New(c *Core) {
	c.startFuncs = append(c.startFuncs, h.Start)
}

func (h *heartbeatOption) Start(ctx context.Context) (<-chan struct{}, <-chan error) {
	ticker := time.NewTicker(h.i)
	defer ticker.Stop()

	errChan := make(chan error)
	startedChan := make(chan struct{})
	go func() {
		defer close(startedChan)
		defer close(errChan)
		startedChan <- struct{}{}
		for {
			select {
			case <-h.done:
				return
			case <-ctx.Done():
				log.Ctx(ctx).Debug().Caller().Msg("closing heartbeat")
				errChan <- ctx.Err()
			case t := <-ticker.C:
				log.Ctx(ctx).
					Debug().
					Time("time", t).
					Msg("heartbeat")
			}
		}
	}()

	return startedChan, errChan
}

func (h *heartbeatOption) Stop(ctx context.Context) error {
	close(h.done)
	return nil
}
