package pages

import (
	"os"
	"strconv"

	"createmod/internal/store"
)

const uploadResultTemplate = "./template/upload_result.html"

// defaultModerationSLAHours is the promise shown wherever a human review is
// pending. Configurable via MODERATION_SLA_HOURS. (#1646)
const defaultModerationSLAHours = 48

func moderationSLAHours() int {
	if v := os.Getenv("MODERATION_SLA_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultModerationSLAHours
}

// PublishCheckRow is one row of the "Automated checks" card / pre-publish rail.
type PublishCheckRow struct {
	Name  string
	State string // ok | warn | bad | neutral
	Note  string
}

// AppearRow is one row of the "Where it appears" rail.
type AppearRow struct {
	Name  string
	State string // ok | warn | bad
	Note  string
}

// ChecklistItemView is one actionable "What you can do now" row.
type ChecklistItemView struct {
	Kind     string
	Note     string
	CTALabel string
	CTAURL   string
	NoAction bool // informational only (e.g. an image being reviewed)
	Source   string
}

// computePublishOutcome fills the outcome fields on d from the schematic's
// moderation state, its open checklist items and any held images. It is the
// single place that turns backend state into the user-facing result page, so the
// page and the HTMX poll fragment always agree. (#1646)
func computePublishOutcome(d *UploadPendingData, schem *store.Schematic, openItems []store.ModerationChecklistItem) {
	d.SLAHours = moderationSLAHours()
	if schem == nil {
		d.Outcome = "pending"
		d.PollActive = true
		d.HeroLevel = "pending"
		d.HeroTitle = "Running final checks…"
		d.HeroBody = "This takes a few seconds. Your schematic is already saved."
		return
	}

	d.HeldImageCount = len(store.HeldGallery(schem))
	held := d.HeldImageCount > 0
	schematicPage := d.SchematicURL // "/schematics/{name}"

	// Actionable items from the open checklist, plus an informational held-image
	// row (holds are not checklist items and need no user action).
	for _, it := range openItems {
		v := ChecklistItemView{Kind: it.Kind, Note: it.Note, Source: it.Source, CTAURL: schematicPage}
		switch it.Kind {
		case store.ChecklistKindDescription:
			v.CTALabel = "Fix description"
		case store.ChecklistKindTitle:
			v.CTALabel = "Edit title"
		case store.ChecklistKindTags:
			v.CTALabel = "Edit tags"
		case store.ChecklistKindCategory:
			v.CTALabel = "Change category"
		case store.ChecklistKindImages:
			v.CTALabel = "Review images"
		default:
			v.CTALabel = "Open schematic"
		}
		d.OpenItems = append(d.OpenItems, v)
	}
	if held {
		word := "image"
		if d.HeldImageCount > 1 {
			word = "images"
		}
		d.OpenItems = append(d.OpenItems, ChecklistItemView{
			Kind:     store.ChecklistKindImages,
			NoAction: true,
			Note:     "Nothing to do. A moderator is double-checking your " + word + ". Shaders sometimes confuse the scanner; this is normal. Visitors don't see the hidden tile meanwhile.",
		})
	}

	descLimited := false
	for _, it := range openItems {
		if it.Kind == store.ChecklistKindDescription {
			descLimited = true
		}
	}

	switch schem.ModerationState {
	case store.ModerationAutoReview:
		d.Outcome = "pending"
		d.PollActive = true
		d.HeroLevel = "pending"
		d.HeroTitle = "Running final checks…"
		d.HeroBody = "This takes a few seconds. Your schematic is already saved and reachable at its link."

	case store.ModerationPublished, store.ModerationApproved:
		if held {
			d.Outcome = "held"
			d.HeroLevel = "ok"
			d.HeroTitle = "Published! One image is being double-checked"
			if d.HeldImageCount > 1 {
				d.HeroTitle = "Published! Some images are being double-checked"
			}
			d.HeroBody = "Your schematic is live everywhere. A moderator will review the flagged image within " + strconv.Itoa(d.SLAHours) + " hours. Visitors do not see it meanwhile, and it returns automatically if approved."
		} else {
			d.Outcome = "full"
			d.HeroLevel = "ok"
			d.HeroTitle = "Your schematic has been published!"
			d.HeroBody = "It's live everywhere: direct link, Latest, and search. Thanks for sharing your build."
		}

	case store.ModerationPublishedLimited, store.ModerationChangesRequested:
		if held {
			d.Outcome = "limited_held"
			d.HeroTitle = "Published, with two things to sort out"
		} else {
			d.Outcome = "limited"
			d.HeroTitle = "Published, with one note"
		}
		d.HeroLevel = "warn"
		d.HeroBody = "Your schematic is published and people can view and download it via the link, but it isn't shown anywhere else on the site until you fix the note below. Once you do, it goes fully live on its own. You don't need to resubmit."

	case store.ModerationRejectedFixable:
		d.Outcome = "rejected_fixable"
		d.HeroLevel = "bad"
		d.HeroTitle = "Not published, but you can fix it"
		d.HeroBody = "This upload broke a content rule. Address the note below and resubmit once. If you think this is a mistake, reply in the moderation thread on the schematic page."

	case store.ModerationRejectedFinal, store.ModerationRejected:
		d.Outcome = "rejected_final"
		d.HeroLevel = "bad"
		d.HeroTitle = "Not published: content violates the rules"
		if schem.ModerationReason != "" {
			d.HeroBody = schem.ModerationReason + " "
		}
		d.HeroBody += "A moderator reviews every removal. You can appeal in the moderation thread on the schematic page."

	default: // flagged and any legacy state: awaiting human review
		d.Outcome = "flagged"
		d.HeroLevel = "pending"
		d.HeroTitle = "Submitted for review"
		d.HeroBody = "A moderator will take a look within " + strconv.Itoa(d.SLAHours) + " hours. You'll get an email either way."
	}

	rejected := d.Outcome == "rejected_fixable" || d.Outcome == "rejected_final"
	flagged := d.Outcome == "flagged"
	viewable := store.IsViewableState(schem.ModerationState)
	listed := store.IsPublicState(schem.ModerationState)

	// Automated checks card.
	d.Checks = []PublishCheckRow{
		{Name: "Schematic file", State: "ok", Note: "Valid Create .nbt."},
		{Name: "Title", State: "ok", Note: "Looks good."},
	}
	if descLimited {
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Description", State: "warn", Note: "Add more detail to appear in Latest and search."})
	} else {
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Description", State: "ok", Note: "Detailed enough for Latest and search."})
	}
	switch {
	case held:
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Images", State: "warn", Note: "An image is being reviewed and is hidden meanwhile."})
	default:
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Images", State: "ok", Note: "All images passed."})
	}
	switch {
	case rejected:
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Content scan", State: "bad", Note: "Violated the content rules."})
	case flagged:
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Content scan", State: "neutral", Note: "Awaiting human review."})
	default:
		d.Checks = append(d.Checks, PublishCheckRow{Name: "Content scan", State: "ok", Note: "No issues found."})
	}

	// Where it appears rail.
	linkState, linkNote := "ok", "Anyone with the link can view it."
	if !viewable {
		linkState, linkNote = "bad", "Not reachable while under review."
	}
	listState, listNote := "ok", "Shown in Latest and search."
	switch {
	case rejected || flagged || !viewable:
		listState, listNote = "bad", "Not listed."
	case !listed:
		listState, listNote = "warn", "Hidden from Latest and search until the note is resolved."
	}
	dlState, dlNote := "ok", "Available to download."
	if !viewable {
		dlState, dlNote = "bad", "Unavailable while under review."
	}
	d.Appears = []AppearRow{
		{Name: "Direct link", State: linkState, Note: linkNote},
		{Name: "Latest & search", State: listState, Note: listNote},
		{Name: "Downloads", State: dlState, Note: dlNote},
	}
}
