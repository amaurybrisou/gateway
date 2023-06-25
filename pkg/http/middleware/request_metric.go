package coremiddleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func RequestMetric(namespace string) mux.MiddlewareFunc {
	handlerCallCnt := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "handler_call_total",
		Help:      "Total number of handler calls",
	}, []string{"path", "method", "status"})

	if err := prometheus.Register(handlerCallCnt); err != nil {
		log.Debug().Err(err).Msg("registering handler_call_total metric")
	}

	latencyHistrogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_latency_histogram",
		Help:      "Handler execution time",
		Buckets:   []float64{0, 0.2, 0.63, 0.8, 1, 30, 60},
	}, []string{"path", "method", "status"})

	if err := prometheus.Register(latencyHistrogram); err != nil {
		log.Debug().Err(err).Msg("registering http_latency_histogram metric")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			o := &responseObserver{ResponseWriter: w}

			next.ServeHTTP(o, r)

			handlerCallCnt.WithLabelValues(r.URL.Path, r.Method, fmt.Sprint(o.status)).Inc()
			latencyHistrogram.WithLabelValues(r.URL.Path, r.Method, fmt.Sprint(o.status)).Observe(time.Since(start).Seconds())
		})
	}
}
