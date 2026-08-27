package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/server"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"net/http"
	"strconv"
)

var mySchematicsTemplates = append([]string{
	"./template/my_schematics.html",
	"./template/include/schematic_card.html",
}, commonTemplates...)

type MySchematicsData struct {
	DefaultData
	Schematics []models.Schematic
	Cards      []MySchematicCard
	// AttentionCount is how many cards ON THIS PAGE need the owner's action
	// (fixable states). Page-scoped to avoid an extra full-table count.
	AttentionCount int
	Page           int
	HasPrev        bool
	HasNext        bool
	PrevURL        string
	NextURL        string
}

// MySchematicCard wraps a schematic with its moderation badge and one-line
// status for the My Schematics grid. (#1646)
type MySchematicCard struct {
	Schematic   models.Schematic
	Badge       string // uppercase label
	BadgeLevel  string // green | yellow | blue | orange | red
	NeedsAction bool   // owner can act to improve visibility
	StatusText  string // one-line status under the card (non-published only)
	StatusLabel string // gold arrow link label, e.g. "Fix description"
	StatusLink  string // URL for the arrow link
}

// buildMySchematicCard derives the badge + status from the schematic's
// moderation state and any held/removed images.
func buildMySchematicCard(s *store.Schematic, m models.Schematic) MySchematicCard {
	c := MySchematicCard{Schematic: m}
	page := "/schematics/" + s.Name
	held := len(store.HeldGallery(s)) > 0

	switch s.ModerationState {
	case store.ModerationPublished, store.ModerationApproved:
		if held {
			c.Badge, c.BadgeLevel = "Image in review", "blue"
			c.StatusText, c.StatusLabel, c.StatusLink = "An image is being reviewed and is hidden meanwhile.", "View", page
		} else {
			c.Badge, c.BadgeLevel = "Published", "green"
		}
	case store.ModerationPublishedLimited:
		c.Badge, c.BadgeLevel = "Link only", "yellow"
		c.NeedsAction = true
		c.StatusText, c.StatusLabel, c.StatusLink = "Add more detail to appear in Latest and search.", "Fix description", page+"/edit"
	case store.ModerationChangesRequested:
		c.Badge, c.BadgeLevel = "Changes requested", "orange"
		c.NeedsAction = true
		c.StatusText, c.StatusLabel, c.StatusLink = "A moderator asked for changes.", "View request", page
	case store.ModerationRejectedFixable:
		c.Badge, c.BadgeLevel = "Rejected", "orange"
		c.NeedsAction = true
		c.StatusText, c.StatusLabel, c.StatusLink = "Fixable. Resolve the issue and resubmit once.", "See why", page
	case store.ModerationRejectedFinal, store.ModerationRejected:
		c.Badge, c.BadgeLevel = "Removed", "red"
		c.StatusText, c.StatusLabel, c.StatusLink = "Removed for a rule violation.", "See why", page
	case store.ModerationAutoReview:
		c.Badge, c.BadgeLevel = "Under review", "blue"
		c.StatusText = "Running final checks…"
	case store.ModerationFlagged:
		c.Badge, c.BadgeLevel = "Under review", "blue"
		c.StatusText, c.StatusLabel, c.StatusLink = "A moderator is taking a look.", "View", page
	default:
		c.Badge, c.BadgeLevel = "Published", "green"
	}
	return c
}

func MySchematicsHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		userID := authenticatedUserID(e)

		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		page = clampPage(page, 1000)
		pageSize := 24
		limit := pageSize + 1
		offset := (page - 1) * pageSize

		results, err := appStore.Schematics.ListByAuthorAll(context.Background(), userID, limit, offset)
		if err != nil {
			return err
		}

		hasNext := len(results) > pageSize
		if hasNext {
			results = results[:pageSize]
		}

		d := MySchematicsData{
			Schematics: MapStoreSchematics(appStore, results, cacheService),
			Page:       page,
			HasPrev:    page > 1,
			HasNext:    hasNext,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/my-schematics?p=%d", page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/my-schematics?p=%d", page+1)
		}

		d.Populate(e)
		translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)

		// Build per-card moderation badges/status from the store rows (which carry
		// held/removed image state) zipped with the translated card models. (#1646)
		d.Cards = make([]MySchematicCard, len(d.Schematics))
		for i := range d.Schematics {
			card := buildMySchematicCard(&results[i], d.Schematics[i])
			if card.NeedsAction {
				d.AttentionCount++
			}
			d.Cards[i] = card
		}
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "My Schematics"))
		d.Title = i18n.T(d.Language, "My Schematics")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Slug = "/my-schematics"
		d.NoIndex = true

		html, err := registry.LoadFiles(mySchematicsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
