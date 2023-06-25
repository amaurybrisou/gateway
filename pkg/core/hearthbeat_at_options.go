package core

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

type heartbeatAtOption struct {
	i time.Duration
	p *url.URL
}

func WithHeartBeatAt(p string, i time.Duration) Options {
	u, err := url.Parse(p)
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("parsing url")
	}
	tmp := &heartbeatAtOption{p: u, i: i}
	return tmp
}

func (h *heartbeatAtOption) New(c *core) {
	c.startFuncs = append(c.startFuncs, h.Start)
}

func (h *heartbeatAtOption) Start(ctx context.Context) error {
	ticker := time.NewTicker(h.i)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Debug().Caller().Msg("closing heartbeat")
			return nil
		case t := <-ticker.C:
			_, err := http.Get(h.p.String())
			if err != nil {
				log.Ctx(ctx).Debug().Caller().Err(err).Msg("http client error")
				return err
			}
			log.Ctx(ctx).
				Debug().
				Time("time", t).
				Msg("heartbeat")
		}
	}
}

func (h *heartbeatAtOption) Stop(ctx context.Context) error {
	return nil
}
