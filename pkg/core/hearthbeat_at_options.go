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

func (h *heartBeatAt) New(c *core) {
	c.startFuncs = append(c.startFuncs, h.Start)
	c.stopFuncs = append(c.stopFuncs, h.Stop)
}

func HeartBeat(options ...ManagementLoopOption) Options {
	m := &heartBeatAt{
		httpCli:        &http.Client{},
		done:           make(chan struct{}, 1),
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
	ticker *time.Ticker
	host   string
}

type healthCheckResult struct {
	service Service
	err     error
}

func (m *heartBeatAt) Start(ctx context.Context) error {
	log.Ctx(ctx).Info().Msg("start loop management")
	go m.updateServiceStatus(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.done:
			log.Ctx(ctx).Info().Msg("closing loop management")
			return nil
		default:
			services, err := m.fetchservicesFunc(ctx)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Send()
				continue
			}

			m.updateTickers(ctx, services)
		}
	}
}

func (m *heartBeatAt) updateTickers(ctx context.Context, services []Service) {
	m.tickerLock.Lock()
	serviceTickers := m.serviceTickers
	m.tickerLock.Unlock()

	updatedTickers := make(map[uuid.UUID]*localTicker)

	for _, service := range services {
		ticker, ok := serviceTickers[service.GetID()]
		if ok && service.GetHost() != ticker.host {
			ticker.ticker.Stop()
			delete(serviceTickers, service.GetID())
			ok = false
		}

		if !ok {
			ticker := time.NewTicker(m.runningInterval)
			updatedTickers[service.GetID()] = &localTicker{
				ticker: ticker,
				host:   service.GetHost(),
			}

			go m.runTicker(ctx, service, ticker)
		} else {
			updatedTickers[service.GetID()] = ticker
		}
	}

	m.tickerLock.Lock()
	m.serviceTickers = updatedTickers
	m.tickerLock.Unlock()
}

func (m *heartBeatAt) runTicker(ctx context.Context, service Service, ticker *time.Ticker) {
	defer func() {
		ticker.Stop()
		log.Ctx(ctx).Debug().Msgf("service %s ticker stopped", service.GetID().String())
	}()

	log.Ctx(ctx).Debug().Msgf("service %s created", service.GetID().String())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go m.checkServiceHealth(ctx, service)
		}
	}
}

func (m *heartBeatAt) checkServiceHealth(ctx context.Context, service Service) {
	result := healthCheckResult{
		service: service,
		err:     nil,
	}

	url := fmt.Sprintf("%s%s", service.GetHost(), m.requestPath)

	resp, err := m.httpCli.Get(url)
	if err != nil {
		result.service.SetStatus(err.Error())
		m.updateRetryCount(service)

		result.err = fmt.Errorf("failed to perform health check request: %w", err)
		m.resultsCh <- result
		return
	}
	defer resp.Body.Close()

	result.service.SetStatus(http.StatusText(resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		m.updateRetryCount(service)

		log.Ctx(ctx).Error().Err(fmt.Errorf("%d", resp.StatusCode)).Send()

		m.resultsCh <- result
		return
	}

	m.resetRetryCount(service)
	m.resultsCh <- result
}
func (m *heartBeatAt) updateRetryCount(service Service) {
	retryCount := service.GetRetryCount() + 5
	service.SetRetryCount(retryCount)
	m.serviceTickers[service.GetID()].ticker.Reset(m.runningInterval + time.Second*time.Duration(retryCount))
}

func (m *heartBeatAt) resetRetryCount(service Service) {
	service.SetRetryCount(0)
	m.serviceTickers[service.GetID()].ticker.Reset(m.runningInterval)
}

func (m *heartBeatAt) updateServiceStatus(ctx context.Context) {
	for result := range m.resultsCh {
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
