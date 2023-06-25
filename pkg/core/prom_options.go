package core

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func WithPrometheus(addr string, port int) Options {
	r := http.NewServeMux()
	r.Handle("/metrics", promhttp.Handler())
	return WithHTTPServer(addr, port, r)
}
