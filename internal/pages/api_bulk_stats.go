package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strings"
)

type apiBulkStatItem struct {
	Name         string  `json:"name"`
	Views        int     `json:"views"`
	Downloads    int     `json:"downloads"`
	Rating       float64 `json:"rating"`
	RatingCount  int     `json:"ratingCount"`
	CommentCount int     `json:"commentCount"`
}

// APISchematicBulkStatsHandler serves GET /api/schematics/stats?names=a,b,c,
// returning just the volatile counters for those schematics so caches can keep
// content cached for a long time while refreshing view/download/rating counts
// on a short timer. Public (counters are shown publicly on the site anyway).
func APISchematicBulkStatsHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		raw := e.Request.URL.Query().Get("names")
		names := make([]string, 0, 32)
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				names = append(names, p)
			}
		}
		if len(names) == 0 {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "names query param required"})
		}
		if len(names) > 100 {
			names = names[:100]
		}

		stats, err := appStore.Schematics.StatsByNames(context.Background(), names)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to fetch stats"})
		}
		items := make([]apiBulkStatItem, 0, len(stats))
		for _, s := range stats {
			items = append(items, apiBulkStatItem{
				Name:         s.Name,
				Views:        s.Views,
				Downloads:    s.Downloads,
				Rating:       s.AvgRating,
				RatingCount:  s.RatingCount,
				CommentCount: s.CommentCount,
			})
		}
		return writeJSON(e, http.StatusOK, items)
	}
}
