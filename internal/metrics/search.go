// Package metrics provides Prometheus instrumentation for search A/B testing.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SearchLatency tracks search request duration per variant.
	SearchLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "createmod_search_latency_seconds",
		Help:    "Search request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"variant", "engine", "index_level"})

	// SearchQueries counts total search queries per variant.
	SearchQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_search_queries_total",
		Help: "Total number of search queries.",
	}, []string{"variant", "engine", "index_level", "zero_results"})

	// SearchClicks counts search result clicks per variant.
	SearchClicks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_search_clicks_total",
		Help: "Total number of search result clicks.",
	}, []string{"variant", "engine", "index_level"})

	// SearchFallbacks counts fallback events (Meilisearch → Bleve).
	SearchFallbacks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_search_fallbacks_total",
		Help: "Total number of search engine fallback events.",
	}, []string{"variant"})
)
