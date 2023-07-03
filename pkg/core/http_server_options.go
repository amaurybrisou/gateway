package core

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type httpServerOption struct {
	srv http.Server
}

func (h *httpServerOption) Start(ctx context.Context) (<-chan struct{}, <-chan error) {
	errChan := make(chan error)

	startedChan := make(chan struct{})

	go func() {
		defer close(errChan)
		defer close(startedChan)

		log.Ctx(ctx).Info().
			Str("address", h.srv.Addr).
			Bool("tls_enabled", h.srv.TLSConfig != nil).
			Msg("start http server")

		l, err := net.Listen("tcp", h.srv.Addr)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Send()
			errChan <- err
			return
		}
		startedChan <- struct{}{}

		if err := h.srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Ctx(ctx).Error().Err(err).Send()
			errChan <- err
		}
	}()

	return startedChan, errChan
}

func (h *httpServerOption) Stop(ctx context.Context) error {
	log.Ctx(ctx).Info().Msg("closing http server")
	return h.srv.Close()
}

func (h *httpServerOption) New(c *Core) {
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
