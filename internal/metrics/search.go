// Package metrics provides Prometheus instrumentation for search.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SearchLatency tracks search request duration.
	SearchLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "createmod_search_latency_seconds",
		Help:    "Search request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"engine", "index"})

	// SearchQueries counts total search queries.
	SearchQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_search_queries_total",
		Help: "Total number of search queries.",
	}, []string{"engine", "index", "zero_results"})

	// SearchClicks counts search result clicks.
	SearchClicks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_search_clicks_total",
		Help: "Total number of search result clicks.",
	}, []string{"engine", "index"})
)
