package core_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type CustomTestService struct {
	startErr bool
	stopErr  bool
	done     chan struct{}
}

var customStartError = errors.New("custom start error")
var customStopError = errors.New("custom stop error")

func (s CustomTestService) New(c *core.Core) {
	c.AddStartFunc(s.Start)
	c.AddStopFunc(s.Stop)
}

func (s CustomTestService) Start(_ context.Context) (<-chan struct{}, <-chan error) {
	errChan := make(chan error)
	startedChan := make(chan struct{})

	go func() {
		time.Sleep(time.Millisecond * 10)
		startedChan <- struct{}{}
	}()

	if s.startErr {
		go func() {
			time.Sleep(time.Millisecond * 10)
			errChan <- customStartError
		}()
	}

	return startedChan, errChan
}

func (s CustomTestService) Stop(_ context.Context) error {
	close(s.done)
	if s.stopErr {
		return customStopError
	}
	return nil
}

func TestCoreNoServiceError(t *testing.T) {
	lcore := core.New()
	started, errChan := lcore.Start(context.Background())
	require.Nil(t, started)
	require.ErrorIs(t, <-errChan, core.ErrNoService)
}

func TestCoreError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	lcore := core.New(CustomTestService{startErr: true, done: make(chan struct{})})
	started, errChan := lcore.Start(ctx)
	require.NotNil(t, started)
	<-started
	require.ErrorIs(t, <-errChan, customStartError)
}

func TestCoreContextDeadlineError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	lcore := core.New(CustomTestService{done: make(chan struct{})})
	started, errChan := lcore.Start(ctx)
	require.NotNil(t, started)
	<-started
	require.ErrorIs(t, <-errChan, context.DeadlineExceeded)
}

func TestCoreShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	lcore := core.New(CustomTestService{startErr: false, done: make(chan struct{})})
	started, _ := lcore.Start(ctx)
	require.NotNil(t, started)
	<-started
	err := lcore.Shutdown(ctx)
	require.NoError(t, err)
}

func TestCoreShutdownError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	lcore := core.New(CustomTestService{startErr: false, stopErr: true, done: make(chan struct{})})
	started, _ := lcore.Start(ctx)
	require.NotNil(t, started)
	<-started
	err := lcore.Shutdown(ctx)
	require.ErrorIs(t, err, customStopError)
}

func TestCoreShutdownContextDeadlineError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	lcore := core.New(CustomTestService{startErr: false, stopErr: true, done: make(chan struct{})})
	started, _ := lcore.Start(ctx)
	require.NotNil(t, started)
	<-started
	ctx, cancel = context.WithTimeout(context.Background(), 0)
	defer cancel()
	err := lcore.Shutdown(ctx)
	require.ErrorIs(t, err, customStopError)
}

func TestCoreServices(t *testing.T) {
	services := []func() core.Options{
		func() core.Options { return core.WithLogLevel(core.LookupEnv("LOG_LEVEL", "debug")) },
		func() core.Options {
			return core.WithHTTPServer(
				core.LookupEnv("HTTP_SERVER_ADDR", "0.0.0.0"),
				core.LookupEnvInt("HTTP_SERVER_PORT", 8089),
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }),
			)
		},
		core.WithSignals,
		func() core.Options {
			return core.WithPrometheus(
				core.LookupEnv("HTTP_PROM_ADDR", "0.0.0.0"),
				core.LookupEnvInt("HTTP_PROM_PORT", 2112),
			)
		},
		func() core.Options {
			return core.HeartBeat(
				core.WithRequestPath("/hc"),
				core.WithClientTimeout(5*time.Second),
				core.WithInterval(core.LookupEnvDuration("HEARTBEAT_INTERVAL", "10s")),
				core.WithErrorIncrement(core.LookupEnvDuration("HEARTBEAT_ERROR_INCREMENT", "5s")),
				core.WithFetchServiceFunction(func(ctx context.Context) ([]core.Service, error) {
					return nil, nil
				}),
				core.WithUpdateServiceStatusFunction(func(ctx context.Context, u uuid.UUID, s string) error {
					return nil
				}),
			)
		}}

	for i := range services {
		options := []core.Options{}
		for j := 0; j <= i; j++ {
			options = append(options, services[j]())
		}

		lcore := core.New(options...)
		started, _ := lcore.Start(context.Background())
		require.NotNil(t, started)
		<-started
		err := lcore.Shutdown(context.Background())
		require.NoError(t, err)

		options = []core.Options{}
		for j := 0; j <= i; j++ {
			options = append(options, services[j]())
		}
	}
}
