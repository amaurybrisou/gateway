package core

import (
	"context"
)

type Options interface {
	New(c *Core)
	Start(context.Context) (<-chan struct{}, <-chan error)
	Stop(context.Context) error
}

type HttpOptions func(*httpServerOption)

type SocketOptions func(*SocketServerOption)
