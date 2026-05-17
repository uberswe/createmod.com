package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/json"
	"net/http"
)

type adClickRequest struct {
	AdUnit string `json:"ad_unit"`
	Dest   string `json:"dest"`
}

func AdClickHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var body adClickRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.BadRequestError("invalid request body", nil)
		}
		if body.AdUnit == "" {
			return e.BadRequestError("ad_unit is required", nil)
		}
		if len(body.AdUnit) > 100 || len(body.Dest) > 500 {
			return e.BadRequestError("field too long", nil)
		}

		_ = appStore.AdClicks.RecordClick(context.Background(), body.AdUnit, body.Dest)

		return e.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}
