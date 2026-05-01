package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CacheItems = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "createmod_cache_items",
		Help: "Current number of items in the in-memory cache.",
	})

	CacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "createmod_cache_hits_total",
		Help: "Total cache hits.",
	})

	CacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "createmod_cache_misses_total",
		Help: "Total cache misses.",
	})
)
