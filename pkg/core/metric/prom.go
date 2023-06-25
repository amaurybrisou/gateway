// Package metric allows to create Prometheus metrics easily.
package metric

import "github.com/prometheus/client_golang/prometheus"

// Namespace is a Prometheus namespace.
type Namespace string

// RegisterCounter registers a new Prometheus vectorial counter.
func (ns Namespace) RegisterCounter(name, help string, labels ...string) *prometheus.CounterVec {
	opts := prometheus.CounterOpts{
		Namespace: string(ns),
		Name:      name,
		Help:      help,
	}
	met := prometheus.NewCounterVec(opts, labels)
	prometheus.MustRegister(met)
	return met
}

// RegisterSummary registers a new Prometheus summary.
func (ns Namespace) RegisterHistogram(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Namespace: string(ns),
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	met := prometheus.NewHistogramVec(opts, labels)
	prometheus.MustRegister(met)
	return met
}
