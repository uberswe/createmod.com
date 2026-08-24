package pages

import (
	"strconv"

	"createmod/internal/store"
)

// OwnerBanner is one stacked status banner shown to a schematic's owner. (#1646)
type OwnerBanner struct {
	Level string // ok | warn | bad | info
	Title string
	Body  string
}

// OwnerChecklistRow is one "To unlock full visibility" item for the owner.
type OwnerChecklistRow struct {
	Note        string
	SourceLabel string // "Auto-check" | "Moderator"
	SourceLevel string // auto | moderator
	CTALabel    string
	CTAURL      string
}

// OwnerModeration is the owner-only moderation view model for the schematic page.
type OwnerModeration struct {
	Banners        []OwnerBanner
	Checklist      []OwnerChecklistRow
	HasChecklist   bool
	IsFinalReject  bool // drives the chat "appeal" empty-state copy
	ChatEnabled    bool
	AppealOnly     bool
	FullyPublished bool
}

// computeOwnerModeration builds the owner's banners + checklist from the
// schematic's state, its open checklist items, and any held/removed images.
// Banners stack: one per active condition (state + per-image holds/removals). It
// is only meaningful for the owner/admin; callers gate on that. (#1646)
func computeOwnerModeration(schem *store.Schematic, openItems []store.ModerationChecklistItem) OwnerModeration {
	var om OwnerModeration
	if schem == nil {
		return om
	}
	sla := strconv.Itoa(moderationSLAHours())
	heldCount := len(store.HeldGallery(schem))
	removedCount := len(schem.RemovedImages)
	page := "/schematics/" + schem.Name

	// --- State banner(s) ---
	switch schem.ModerationState {
	case store.ModerationPublished, store.ModerationApproved:
		if heldCount == 0 && removedCount == 0 {
			om.Banners = append(om.Banners, OwnerBanner{
				Level: "ok",
				Title: "Published and fully visible",
				Body:  "Your schematic is live everywhere: direct link, Latest, and search.",
			})
			om.FullyPublished = true
		}
	case store.ModerationPublishedLimited:
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "warn",
			Title: "Published with notes: visible via direct link only",
			Body:  "Anyone with the link can view and download it, but it stays out of Latest and search until the checklist below is resolved. Fixing it promotes it automatically — no re-submission needed.",
		})
	case store.ModerationChangesRequested:
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "warn",
			Title: "Changes requested: temporarily unlisted",
			Body:  "A moderator asked for changes. Your schematic is hidden from everyone but you until the checklist below is resolved.",
		})
	case store.ModerationRejectedFixable:
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "bad",
			Title: "Not published — but you can fix it",
			Body:  "This upload broke a content rule. Resolve the checklist below and resubmit once. Questions? Use the moderation thread below.",
		})
	case store.ModerationRejectedFinal, store.ModerationRejected:
		body := "This schematic was removed for violating the rules and won't be republished. "
		if schem.ModerationReason != "" {
			body = schem.ModerationReason + " "
		}
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "bad",
			Title: "Removed: rule violation",
			Body:  body + "A human confirmed this decision. You can appeal in the moderation thread below.",
		})
		om.IsFinalReject = true
		om.AppealOnly = true
	case store.ModerationAutoReview:
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "info",
			Title: "Running final checks",
			Body:  "This usually takes a few seconds. Your schematic is already reachable at its link.",
		})
	case store.ModerationFlagged:
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "warn",
			Title: "Submitted for review",
			Body:  "A moderator will take a look within " + sla + " hours. You'll get an email either way.",
		})
	}

	// --- Per-image banners (can co-occur with published/limited) ---
	if heldCount > 0 {
		word, verb := "image", "is"
		if heldCount > 1 {
			word, verb = "images", "are"
		}
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "info",
			Title: strconv.Itoa(heldCount) + " " + word + " hidden pending review",
			Body: "A moderator will check " + verb + " within " + sla + " hours. Shaders sometimes confuse the scanner; this is normal. " +
				"Visitors simply don't see the hidden tile meanwhile — it returns automatically if approved.",
		})
	}
	if removedCount > 0 {
		word := "image"
		if removedCount > 1 {
			word = "images"
		}
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "bad",
			Title: strconv.Itoa(removedCount) + " " + word + " removed after review",
			Body:  "A moderator removed " + word + " that didn't meet the rules. You're welcome to upload a replacement from the edit page.",
		})
	}

	// --- Checklist ---
	for _, it := range openItems {
		row := OwnerChecklistRow{Note: it.Note, CTAURL: page + "/edit"}
		if it.Source == store.ChecklistSourceModerator {
			row.SourceLabel = "Moderator"
			row.SourceLevel = "moderator"
		} else {
			row.SourceLabel = "Auto-check"
			row.SourceLevel = "auto"
		}
		switch it.Kind {
		case store.ChecklistKindDescription:
			row.CTALabel = "Fix description"
		case store.ChecklistKindTitle:
			row.CTALabel = "Edit title"
		case store.ChecklistKindTags:
			row.CTALabel = "Edit tags"
		case store.ChecklistKindCategory:
			row.CTALabel = "Change category"
		case store.ChecklistKindImages:
			row.CTALabel = "Review images"
		default:
			row.CTALabel = "Fix now"
		}
		om.Checklist = append(om.Checklist, row)
	}
	om.HasChecklist = len(om.Checklist) > 0
	return om
}
