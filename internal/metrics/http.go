package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "createmod_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "path", "status"})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "status"})

	HTTPResponseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "createmod_http_response_size_bytes",
		Help:    "HTTP response size in bytes.",
		Buckets: []float64{100, 1000, 10000, 100000, 500000, 1000000, 5000000},
	}, []string{"method"})
)
