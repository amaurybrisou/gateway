package core

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type httpServerOption struct {
	srv http.Server
}

func (h *httpServerOption) Start(ctx context.Context) error {
	return func() error {
		log.Ctx(ctx).Debug().
			Str("address", h.srv.Addr).
			Bool("tls_enabled", h.srv.TLSConfig != nil).
			Msg("start http server")
		err := h.srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Ctx(ctx).Error().Err(err).Send()
			return err
		}
		return nil
	}()
}

func (h *httpServerOption) Stop(ctx context.Context) error {
	log.Ctx(ctx).Debug().Msg("closing http server")
	return h.srv.Close()
}

func (h *httpServerOption) New(c *core) {
	c.startFuncs = append(c.startFuncs, h.Start)
	c.stopFuncs = append(c.stopFuncs, h.Stop)
}

func WithServerTLSConfig(ctx context.Context, t *tls.Config) HttpOptions {
	return func(hso *httpServerOption) {
		hso.srv.TLSConfig = t
	}
}

func WithHTTPServer(addr string, port int, r http.Handler, opts ...HttpOptions) Options {
	srv := &httpServerOption{
		srv: http.Server{
			Addr:              fmt.Sprintf("%s:%d", addr, port),
			Handler:           r,
			ReadHeaderTimeout: time.Second * 30,
		},
	}

	for _, o := range opts {
		o(srv)
	}

	return srv
}
