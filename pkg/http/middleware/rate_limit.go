package coremiddleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

type RateLimitMiddleware struct {
	limiter *rate.Limiter
}

type RateLimitMiddlewareOption func(*RateLimitMiddleware)

func WithRateLimit(rateLimit rate.Limit, burst int) RateLimitMiddlewareOption {
	return func(m *RateLimitMiddleware) {
		m.limiter = rate.NewLimiter(rateLimit, burst)
	}
}

func NewRateLimitMiddleware(options ...RateLimitMiddlewareOption) *RateLimitMiddleware {
	middleware := &RateLimitMiddleware{
		limiter: rate.NewLimiter(rate.Inf, 0),
	}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
