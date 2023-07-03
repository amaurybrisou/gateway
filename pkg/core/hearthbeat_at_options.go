package core

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type heartBeatAt struct {
	runningInterval, heartBeatIncrement time.Duration
	httpCli                             *http.Client
	requestPath                         string
	done                                chan struct{}

	serviceTickers map[uuid.UUID]*localTicker
	tickerLock     sync.Mutex
	resultsCh      chan healthCheckResult

	fetchservicesFunc       func(context.Context) ([]Service, error)
	updateServiceStatusFunc func(context.Context, uuid.UUID, string) error
}

type Service interface {
	GetHost() string
	GetID() uuid.UUID
	GetRetryCount() int
	SetRetryCount(int)
	SetStatus(string)
	GetStatus() string
}

type ManagementLoopOption func(m *heartBeatAt)

func WithClientTimeout(t time.Duration) ManagementLoopOption {
	return func(m *heartBeatAt) {
		m.httpCli.Timeout = t
	}
}

func WithInterval(t time.Duration) ManagementLoopOption {
	return func(m *heartBeatAt) {
		m.runningInterval = t
	}
}

func WithErrorIncrement(t time.Duration) ManagementLoopOption {
	return func(m *heartBeatAt) {
		m.heartBeatIncrement = t
	}
}

func WithRequestPath(p string) ManagementLoopOption {
	return func(m *heartBeatAt) {
		m.requestPath = p
	}
}

func WithFetchServiceFunction(f func(context.Context) ([]Service, error)) ManagementLoopOption {
	return func(m *heartBeatAt) {
		m.fetchservicesFunc = f
	}
}

func WithUpdateServiceStatusFunction(f func(context.Context, uuid.UUID, string) error) ManagementLoopOption {
	return func(m *heartBeatAt) {
		m.updateServiceStatusFunc = f
	}
}

func (h *heartBeatAt) New(c *Core) {
	c.startFuncs = append(c.startFuncs, h.Start)
	c.stopFuncs = append(c.stopFuncs, h.Stop)
}

func HeartBeat(options ...ManagementLoopOption) Options {
	m := &heartBeatAt{
		httpCli:        &http.Client{},
		done:           make(chan struct{}),
		resultsCh:      make(chan healthCheckResult),
		serviceTickers: make(map[uuid.UUID]*localTicker),
	}

	for _, o := range options {
		o(m)
	}

	return m
}

// Stop the loop.
func (m *heartBeatAt) Stop(ctx context.Context) error {
	m.done <- struct{}{}
	return nil
}

type localTicker struct {
	ticker  *time.Ticker
	service Service
}

type healthCheckResult struct {
	service Service
	err     error
}

func (m *heartBeatAt) Start(ctx context.Context) (<-chan struct{}, <-chan error) {
	log.Ctx(ctx).Info().Msg("start heartbeat")

	go m.updateServiceStatus(ctx)

	errChan := make(chan error)
	startedChan := make(chan struct{})

	t := time.NewTicker(m.runningInterval)
	defer t.Stop()

	loop := func() {
		services, err := m.fetchservicesFunc(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Send()
			return
		}

		m.updateTickers(ctx, services)
	}

	go func() {
		defer close(errChan)
		defer close(startedChan)
		defer close(m.done)
		loop()
		startedChan <- struct{}{}
		for {
			select {
			case <-ctx.Done():
				log.Ctx(ctx).Info().Msg("stop heartbeat")
				errChan <- ctx.Err()
				return
			case <-m.done:
				log.Ctx(ctx).Info().Msg("stop heartbeat")
				return
			case <-t.C:
				loop()
			}
		}
	}()

	return startedChan, errChan
}

func (m *heartBeatAt) updateTickers(ctx context.Context, services []Service) {
	if len(services) == 0 && len(m.serviceTickers) == 0 {
		return
	}
	m.tickerLock.Lock()
	serviceTickers := m.serviceTickers
	m.tickerLock.Unlock()

	updatedTickers := make(map[uuid.UUID]*localTicker)

	servicesMap := make(map[uuid.UUID]Service)
	for _, svc := range services {
		servicesMap[svc.GetID()] = svc
	}

	for _, ticker := range serviceTickers {
		service, ok := servicesMap[ticker.service.GetID()]
		if !ok {
			ticker.ticker.Stop()
			continue
		}
		updatedTickers[service.GetID()] = ticker
	}

	for _, service := range servicesMap {
		if _, ok := updatedTickers[service.GetID()]; !ok {
			ticker := &localTicker{
				ticker:  time.NewTicker(m.runningInterval),
				service: service,
			}
			updatedTickers[service.GetID()] = ticker

			go m.runTicker(ctx, ticker)
		}
	}

	m.tickerLock.Lock()
	m.serviceTickers = updatedTickers
	m.tickerLock.Unlock()
}

func (m *heartBeatAt) runTicker(ctx context.Context, ticker *localTicker) {
	defer func() {
		ticker.ticker.Stop()
		log.Ctx(ctx).Debug().Msgf("service %s ticker stopped", ticker.service.GetID())
	}()

	log.Ctx(ctx).Debug().Msgf("service %s created", ticker.service.GetID())

	wg := sync.WaitGroup{}

	wg.Add(1)
	go m.checkServiceHealth(ctx, &wg, ticker)
	wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.ticker.C:
			wg.Add(1)
			go m.checkServiceHealth(ctx, &wg, ticker)
			wg.Wait()
		}
	}
}

func (m *heartBeatAt) checkServiceHealth(ctx context.Context, wg *sync.WaitGroup, ticker *localTicker) {
	defer wg.Done()

	result := healthCheckResult{
		service: ticker.service,
		err:     nil,
	}

	url := fmt.Sprintf("%s%s", ticker.service.GetHost(), m.requestPath)

	resp, err := m.httpCli.Get(url)
	if err != nil {
		result.service.SetStatus(err.Error())
		m.updateRetryCount(ticker.service)

		result.err = fmt.Errorf("failed to perform health check request: %w", err)
		m.resultsCh <- result
		return
	}
	defer resp.Body.Close()

	result.service.SetStatus(http.StatusText(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		m.updateRetryCount(ticker.service)

		log.Ctx(ctx).Error().Err(fmt.Errorf("%d", resp.StatusCode)).Send()

		m.resultsCh <- result
		return
	}

	m.resetRetryCount(ticker.service)
	m.resultsCh <- result
}

func (m *heartBeatAt) updateRetryCount(service Service) {
	retryCount := service.GetRetryCount() + 5
	service.SetRetryCount(retryCount)
	if s, ok := m.serviceTickers[service.GetID()]; ok {
		s.ticker.Reset(m.runningInterval + time.Second*time.Duration(retryCount))
	}
}

func (m *heartBeatAt) resetRetryCount(service Service) {
	service.SetRetryCount(0)
	if s, ok := m.serviceTickers[service.GetID()]; ok {
		s.ticker.Reset(m.runningInterval)
	}
}

func (m *heartBeatAt) updateServiceStatus(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case result := <-m.resultsCh:
			log.Ctx(ctx).Debug().
				Any("service_id", result.service.GetID()).
				Any("service_status", result.service.GetStatus()).
				Send()
			err := m.updateServiceStatusFunc(ctx, result.service.GetID(), result.service.GetStatus())
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Send()
			}
		}
	}
}
