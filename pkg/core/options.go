package core

import (
	"context"
)

type Options interface {
	New(c *core)
	Start(context.Context) error
	Stop(context.Context) error
}

type HttpOptions func(*httpServerOption)

type SocketOptions func(*SocketServerOption)
