package test

import (
	"context"
	"fmt"
	"io"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/stripe/stripe-go/v72"
)

type Container struct {
	*dockertest.Pool
	*dockertest.Resource
}

type ContainerConfig struct {
	Repository string
	Tag        string
	Env        []string
	Cmd        []string
}

type PoolConfig struct {
	Addr *string
	Pool *dockertest.Pool
}

type NewContainerOption func(o *dockertest.RunOptions)

func NewContainer(poolconfig *PoolConfig, config ContainerConfig, opts ...NewContainerOption) (*Container, error) {
	if poolconfig == nil || poolconfig.Addr == nil {
		poolconfig = &PoolConfig{
			Addr: stripe.String(""),
		}
	}

	var pool *dockertest.Pool
	if poolconfig.Pool != nil {
		pool = poolconfig.Pool
	} else if poolconfig.Addr != nil {
		var err error
		pool, err = dockertest.NewPool(*poolconfig.Addr)
		if err != nil {
			return nil, fmt.Errorf("failed creating pool | %w", err)
		}
	}

	runOptions := &dockertest.RunOptions{
		Repository: config.Repository,
		Tag:        config.Tag,
		Env:        config.Env,
		Cmd:        config.Cmd,
	}

	for _, o := range opts {
		o(runOptions)
	}

	r, err := pool.RunWithOptions(runOptions)
	if err != nil {
		return nil, err
	}

	return &Container{
		Pool:     pool,
		Resource: r,
	}, nil
}

func (c *Container) TailLogs(ctx context.Context, wr io.Writer, follow bool) error {
	opts := docker.LogsOptions{
		Context: ctx,

		Stderr:      true,
		Stdout:      true,
		Follow:      follow,
		Timestamps:  true,
		RawTerminal: true,

		Container: c.Resource.Container.ID,

		OutputStream: wr,
	}

	return c.Pool.Client.Logs(opts)
}
