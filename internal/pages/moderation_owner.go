package pages

import (
	"strconv"
	"strings"

	"createmod/internal/store"

	strip "github.com/grokify/html-strip-tags-go"
)

// OwnerBanner is one stacked status banner shown to a schematic's owner. (#1646)
type OwnerBanner struct {
	Level string // ok | warn | bad | info
	Title string
	Body  string
}

// OwnerChecklistKind is one affected section within a grouped checklist note:
// its display label ("Title") and the action-button label ("Edit title").
type OwnerChecklistKind struct {
	Kind     string // description | title | images | tags | category
	Label    string // "Description" | "Title" | "Images" | "Tags" | "Category"
	CTALabel string // "Fix now" | "Edit title" | "Review images" ...
}

// OwnerChecklistRow is one "To unlock full visibility" note for the owner. A
// moderator who selects several sections with one note produces a single row
// tagged with every affected section instead of the note repeated per section.
type OwnerChecklistRow struct {
	Note        string
	SourceLabel string // "Auto-check" | "Moderator"
	SourceLevel string // auto | moderator
	Kinds       []OwnerChecklistKind
}

// checklistKindDisplay maps a checklist kind to its section label and action
// button label.
func checklistKindDisplay(kind string) (label, cta string) {
	switch kind {
	case store.ChecklistKindDescription:
		return "Description", "Fix now"
	case store.ChecklistKindTitle:
		return "Title", "Edit title"
	case store.ChecklistKindTags:
		return "Tags", "Edit tags"
	case store.ChecklistKindCategory:
		return "Category", "Change category"
	case store.ChecklistKindImages:
		return "Images", "Review images"
	default:
		return kind, "Fix now"
	}
}

// OwnerModeration is the owner-only moderation view model for the schematic page.
type OwnerModeration struct {
	Banners      []OwnerBanner
	Checklist    []OwnerChecklistRow
	HasChecklist bool
	// DescriptionText is the current description as plain text, prefilled into
	// the inline "Fix now" editor. (#1646)
	DescriptionText string
	IsFinalReject   bool // drives the chat "appeal" empty-state copy
	ChatEnabled     bool
	AppealOnly      bool
	FullyPublished  bool
	// CanRequestHumanReview is true when the current outcome was automated and
	// the author hasn't already asked for a human, so the schematic page offers a
	// "request human review without changes" button. (#1646)
	CanRequestHumanReview bool
	// HumanReviewRequested is true once the author has asked for a human to look. (#1646)
	HumanReviewRequested bool
	// EditURL is the schematic edit page, used by the non-description checklist
	// action buttons (title/images/tags/category).
	EditURL string
	// HasOpenDescription is true when any open checklist item targets the
	// description, so the single inline "Fix now" editor is rendered once.
	HasOpenDescription bool
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
			Body:  "Your schematic is published and people can view it via the link, but it isn't shown anywhere else on the site until you fix the issues below. Once you do, it goes fully live on its own. You don't need to resubmit.",
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
			Title: "Not published, but you can fix it",
			Body:  "This upload broke a content rule. Clear the checklist below and resubmit once. If you have questions, use the moderation thread below.",
		})
	case store.ModerationRejectedFinal, store.ModerationRejected:
		body := "This schematic was removed for violating the rules and won't be republished. "
		if schem.ModerationReason != "" {
			body = schem.ModerationReason + " "
		}
		om.Banners = append(om.Banners, OwnerBanner{
			Level: "bad",
			Title: "Removed: rule violation",
			Body:  body + "A moderator confirmed this decision. You can appeal in the moderation thread below.",
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
				"Visitors simply don't see the hidden tile meanwhile. It returns automatically if approved.",
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

	om.DescriptionText = strings.TrimSpace(strip.StripTags(schem.Content))

	// --- Checklist ---
	// Collapse items that share a note + source into one row tagged with every
	// affected section, so a moderator selecting Title + Description + Images with
	// one message shows once (tags + a button each) rather than three identical
	// bullets. Keeps first-seen order. (#1646)
	om.EditURL = page + "/edit"
	groups := map[string]int{} // "source\x00note" -> index into om.Checklist
	for _, it := range openItems {
		sourceLevel, sourceLabel := "auto", "Auto-check"
		if it.Source == store.ChecklistSourceModerator {
			sourceLevel, sourceLabel = "moderator", "Moderator"
		}
		if it.Kind == store.ChecklistKindDescription {
			om.HasOpenDescription = true
		}
		label, cta := checklistKindDisplay(it.Kind)
		kind := OwnerChecklistKind{Kind: it.Kind, Label: label, CTALabel: cta}

		key := sourceLevel + "\x00" + it.Note
		if idx, ok := groups[key]; ok {
			dup := false
			for _, k := range om.Checklist[idx].Kinds {
				if k.Kind == it.Kind {
					dup = true
					break
				}
			}
			if !dup {
				om.Checklist[idx].Kinds = append(om.Checklist[idx].Kinds, kind)
			}
			continue
		}
		groups[key] = len(om.Checklist)
		om.Checklist = append(om.Checklist, OwnerChecklistRow{
			Note: it.Note, SourceLabel: sourceLabel, SourceLevel: sourceLevel,
			Kinds: []OwnerChecklistKind{kind},
		})
	}
	om.HasChecklist = len(om.Checklist) > 0

	// Offer "request human review" only when the current outcome was automated
	// (a human hasn't decided) and the author hasn't already asked. A human
	// decision is final until a human revisits it. (#1646)
	om.HumanReviewRequested = schem.HumanReviewRequested
	switch schem.ModerationState {
	case store.ModerationPublishedLimited, store.ModerationChangesRequested:
		om.CanRequestHumanReview = schem.ModerationReviewedBy == store.ReviewedBySystem && !schem.HumanReviewRequested
	}
	return om
}
