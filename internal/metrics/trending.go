package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// IndexPageViews counts index page views per trending variant.
	IndexPageViews = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_index_page_views_total",
		Help: "Total index page views per trending variant.",
	}, []string{"variant", "window_days"})

	// IndexClicks counts index page schematic clicks per trending variant.
	IndexClicks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "createmod_index_clicks_total",
		Help: "Total index page schematic clicks per trending variant.",
	}, []string{"variant", "window_days", "section"})
)
