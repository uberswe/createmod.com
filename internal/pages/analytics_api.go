package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type analyticsEvent struct {
	Type  int `json:"type"`
	Value int `json:"value"`
}

type analyticsRequest struct {
	SchematicID string           `json:"schematic_id"`
	Events      []analyticsEvent `json:"events"`
}

func AnalyticsEventHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var body analyticsRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.BadRequestError("invalid request body", nil)
		}
		if body.SchematicID == "" || len(body.Events) == 0 {
			return e.BadRequestError("schematic_id and events are required", nil)
		}
		if len(body.Events) > 10 {
			return e.BadRequestError("too many events", nil)
		}

		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, body.SchematicID)
		if err != nil || schem == nil {
			return e.BadRequestError("schematic not found", nil)
		}

		ip := e.RealIP()

		for _, ev := range body.Events {
			if ev.Type < 1 || ev.Type > 5 {
				continue
			}
			val := ev.Value
			if val <= 0 {
				val = 1
			}
			if ev.Type == store.EventTimeOnPage && val > 3600 {
				val = 3600
			}

			if ev.Type == store.EventVideoPlay || ev.Type == store.EventYouTubeClick {
				dedupKey := fmt.Sprintf("evt:%s:%s:%d", ip, body.SchematicID, ev.Type)
				if _, found := cacheService.Get(dedupKey); found {
					continue
				}
				cacheService.SetWithTTL(dedupKey, true, 1*time.Hour)
			}

			_ = appStore.Stats.RecordEvent(ctx, body.SchematicID, ev.Type, val)
		}

		return e.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}
