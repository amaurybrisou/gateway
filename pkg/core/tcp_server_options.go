package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type SocketHandler func(context.Context, net.Conn)

type SocketServerOption struct {
	fd                          net.Listener
	addr, proto                 string
	port                        int
	h                           SocketHandler
	ReadDeadline, WriteDeadline time.Duration
}

func (o *SocketServerOption) New(c *core) {
	c.startFuncs = append(c.startFuncs, o.Start)
	c.stopFuncs = append(c.stopFuncs, o.Stop)
}

func (o *SocketServerOption) Start(ctx context.Context) error {
	var lc net.ListenConfig
	fd, err := lc.Listen(ctx, o.proto, fmt.Sprintf("%s:%d", o.addr, o.port))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		return err
	}

	log.Ctx(ctx).Info().
		Str("address", fd.Addr().String()).
		Msg("start socket server")

	o.fd = fd

	return func() error {
		defer fd.Close()
		for {
			conn, err := fd.Accept()
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			if err != nil {
				log.Ctx(ctx).Error().Err(err).Send()
				return err
			}

			ctx = log.With().
				Str("address", conn.RemoteAddr().String()).
				Str("x-correlation-id", uuid.NewString()).
				Logger().WithContext(ctx)

			log.Ctx(ctx).Debug().
				Str("proto", conn.RemoteAddr().Network()).
				Msg("new connection")

			if err := conn.SetReadDeadline(time.Now().Add(o.ReadDeadline)); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("conn.SetReadDeadline()")
				return err
			}

			go o.h(ctx, conn)
		}
	}()
}

func (o *SocketServerOption) Stop(ctx context.Context) error {
	log.Ctx(ctx).Info().Msg("closing socket server")
	if o.fd != nil {
		return o.fd.Close()
	}
	return nil
}

func WithSocketServer(proto, addr string, port int, h SocketHandler, opts ...SocketOptions) Options {
	s := &SocketServerOption{
		proto: proto,
		port:  port,
		addr:  addr,
		h:     h,
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func WithReadDeadline(t time.Duration) SocketOptions {
	return func(sso *SocketServerOption) {
		sso.ReadDeadline = t
	}
}

func WithWriteDeadline(t time.Duration) SocketOptions {
	return func(sso *SocketServerOption) {
		sso.WriteDeadline = t
	}
}
