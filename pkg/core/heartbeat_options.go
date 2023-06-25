package core

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type heartbeatOption struct {
	i time.Duration
}

func WithHeartBeat(i time.Duration) Options {
	tmp := &heartbeatOption{i: i}
	return tmp
}

func (h *heartbeatOption) New(c *core) {
	c.startFuncs = append(c.startFuncs, h.Start)
}

func (h *heartbeatOption) Start(ctx context.Context) error {
	ticker := time.NewTicker(h.i)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Debug().Caller().Msg("closing heartbeat")
			return nil
		case t := <-ticker.C:
			log.Ctx(ctx).
				Debug().
				Time("time", t).
				Msg("heartbeat")
		}
	}
}

func (h *heartbeatOption) Stop(ctx context.Context) error {
	return nil
}
