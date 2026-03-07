package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/json"
	"log/slog"
	"net/http"
)

// RatingUpsertHandler handles POST /api/ratings to create or update a rating.
// Replaces PB's POST /api/collections/schematic_ratings/records endpoint.
func RatingUpsertHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		if userID == "" {
			return e.UnauthorizedError("", nil)
		}

		var body struct {
			SchematicID string  `json:"schematic"`
			Rating      float64 `json:"rating"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return e.BadRequestError("invalid request body", nil)
		}
		if body.SchematicID == "" || body.Rating < 1 || body.Rating > 5 {
			return e.BadRequestError("schematic and rating (1-5) are required", nil)
		}

		ctx := context.Background()

		// Validate schematic exists
		schem, err := appStore.Schematics.GetByID(ctx, body.SchematicID)
		if err != nil || schem == nil {
			return e.BadRequestError("invalid schematic", nil)
		}

		if err := appStore.ViewRatings.UpsertRating(ctx, userID, body.SchematicID, body.Rating); err != nil {
			slog.Error("failed to upsert rating", "error", err)
			return e.InternalServerError("could not save rating", nil)
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
