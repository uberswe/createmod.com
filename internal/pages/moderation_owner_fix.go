package pages

import (
	"context"
	"net/http"
	"strings"

	"createmod/internal/mailer"
	"createmod/internal/models"
	"createmod/internal/moderation"
	"createmod/internal/ratelimit"
	"createmod/internal/server"
	"createmod/internal/store"

	strip "github.com/grokify/html-strip-tags-go"
	"github.com/sym01/htmlsanitizer"
)

const schematicModerationFragment = "./template/include/schematic_moderation.html"

// ownerModFragmentData renders the owner moderation region (banners + checklist)
// standalone for the inline fix-and-recheck HTMX swap. Its accessed fields
// (.OwnerModeration, .Schematic.ID/.Name) match SchematicData, so the same
// fragment works both inside the page and as a swap target. (#1646)
type ownerModFragmentData struct {
	OwnerModeration OwnerModeration
	Schematic       models.Schematic
}

// SchematicDescriptionRecheckHandler backs the inline "Save & re-check" editor on
// a schematic's owner view. It saves the new description, re-runs the quality
// check, resolves the description checklist item on pass, auto-promotes when
// nothing is left open (reindex + "live" email), and returns the refreshed
// moderation region so HTMX swaps it in place — no page reload. (#1646)
func SchematicDescriptionRecheckHandler(registry *server.Registry, appStore *store.Store, moderationSvc *moderation.Service, mailService *mailer.Service, rl ratelimit.Limiter, enqueueChecklistRecheck ChecklistRecheckEnqueuer, enqueueSearchUpsert SearchIndexEnqueuer) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		userID := authenticatedUserID(e)
		id := e.Request.PathValue("id")
		if id == "" {
			return e.BadRequestError("schematic id is required", nil)
		}
		ctx := context.Background()
		schem, err := appStore.Schematics.GetByID(ctx, id)
		if err != nil || schem == nil {
			return e.NotFoundError("schematic not found", nil)
		}
		if schem.AuthorID != userID {
			return e.ForbiddenError("you are not the author of this schematic", nil)
		}

		desc := strings.TrimSpace(e.Request.FormValue("description"))
		if err := validateDescription(desc); err != nil {
			// Client disables Save under 5 words; this is a backstop. HTMX won't
			// swap on a 4xx, so the editor stays open with the user's text intact.
			return e.String(http.StatusBadRequest, err.Error())
		}
		sanitized := desc
		if san, sErr := htmlsanitizer.NewHTMLSanitizer().SanitizeString(desc); sErr == nil {
			sanitized = san
		}
		schem.Content = sanitized
		schem.Excerpt = truncateRunesEllipsis(strings.TrimSpace(strip.StripTags(sanitized)), 180)
		if err := appStore.Schematics.Update(ctx, schem); err != nil {
			return e.InternalServerError("failed to save description", nil)
		}

		// Re-check + promote only for the limited/changes-requested states.
		// The quality re-check is a paid OpenAI call, so it draws on the user's
		// hourly paid-AI budget. Under budget we run it now for instant feedback;
		// over budget we DEFER to the deduped recheck job (which snoozes until the
		// budget frees), so rapid editing can't run up cost. (#1646)
		if schem.ModerationState == store.ModerationPublishedLimited ||
			schem.ModerationState == store.ModerationChangesRequested {
			if moderationSvc != nil && appStore.ModerationChecklist != nil && ratelimit.AllowPaidAI(ctx, rl, userID) {
				if q, qErr := moderationSvc.CheckSchematicQuality(schem.Title, schem.Content); qErr == nil && q.Approved {
					_, _ = appStore.ModerationChecklist.ResolveOpenByKind(ctx, id, store.ChecklistKindDescription)
				}
				if promoted, _ := PromoteIfChecklistResolved(ctx, appStore, id); promoted {
					if enqueueSearchUpsert != nil {
						_ = enqueueSearchUpsert(ctx, id)
					}
					SendSchematicLiveEmail(ctx, mailService, appStore, id)
				}
			} else if enqueueChecklistRecheck != nil {
				// Over budget (or no moderation svc): the description is saved;
				// defer the re-check. The job dedupes and snoozes until budget frees.
				_ = enqueueChecklistRecheck(ctx, id)
			}
		}

		// Render the refreshed region from the fresh state.
		fresh, err := appStore.Schematics.GetByID(ctx, id)
		if err != nil || fresh == nil {
			fresh = schem
		}
		var open []store.ModerationChecklistItem
		if appStore.ModerationChecklist != nil {
			open, _ = appStore.ModerationChecklist.ListOpenBySchematic(ctx, id)
		}
		data := ownerModFragmentData{
			OwnerModeration: computeOwnerModeration(fresh, open),
			Schematic:       models.Schematic{ID: fresh.ID, Name: fresh.Name},
		}
		html, err := registry.LoadFiles(schematicModerationFragment).Render(data)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
