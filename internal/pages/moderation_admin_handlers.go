package pages

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/nbtparser"
	"createmod/internal/server"
	"createmod/internal/store"

	"github.com/mergestat/timediff"
	"github.com/sym01/htmlsanitizer"
)

var moderatorReviewTemplates = append([]string{
	"./template/moderator_review.html",
}, commonTemplates...)

// ModFlag is a color-coded flag chip on a queue card. Level ∈ content|image|severe.
type ModFlag struct {
	Label string
	Level string
}

// ModQueueCard is one item in the moderator queue column.
type ModQueueCard struct {
	ID        string
	Title     string
	Name      string
	Author    string
	Approvals int64
	Removals  int64
	Age       string
	Category  string
	Tags      []string
	Flags     []ModFlag
	Selected  bool
}

// ModImage is one image in the detail media viewer.
type ModImage struct {
	Filename string
	URL      string
	Featured bool
	Held     bool
	Removed  bool
}

// ModDetail is the selected schematic's review detail.
type ModDetail struct {
	ID               string
	Title            string
	Name             string
	Author           string
	Approvals        int64
	Removals         int64
	Age              string
	State            string
	DetectedLanguage string
	Description      template.HTML
	AutoReason       string
	HasAutoReason    bool
	Images           []ModImage
	Materials        []nbtparser.Material
	Category         string
	Tags             []string
}

type ModeratorReviewData struct {
	DefaultData
	Queue     []ModQueueCard
	Total     int64
	Detail    *ModDetail
	HasDetail bool
	SLAHours  int
}

const moderatorQueuePageSize = 50

// ModeratorReviewHandler renders the moderator review screen: a queue of items
// needing attention plus the selected item's detail + actions. (#1646)
func ModeratorReviewHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ctx := context.Background()

		rows, err := appStore.Schematics.ListModerationQueue(ctx, moderatorQueuePageSize, 0)
		if err != nil {
			return err
		}
		total, _ := appStore.Schematics.CountModerationQueue(ctx)

		selID := e.Request.URL.Query().Get("id")
		if selID == "" && len(rows) > 0 {
			selID = rows[0].ID
		}

		d := ModeratorReviewData{Total: total, SLAHours: moderationSLAHours()}
		d.Queue = make([]ModQueueCard, 0, len(rows))
		for i := range rows {
			d.Queue = append(d.Queue, buildModQueueCard(ctx, appStore, &rows[i], selID))
		}
		if selID != "" {
			d.Detail = buildModDetail(ctx, appStore, selID)
			d.HasDetail = d.Detail != nil
		}

		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Moderation"))
		d.Title = i18n.T(d.Language, "Moderation Review")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Slug = "/admin/moderation"
		d.NoIndex = true

		html, err := registry.LoadFiles(moderatorReviewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func authorTrust(ctx context.Context, appStore *store.Store, authorID string) (approvals, removals int64) {
	if authorID == "" {
		return 0, 0
	}
	approvals, _ = appStore.Schematics.CountByAuthor(ctx, authorID)
	removals, _ = appStore.Schematics.CountSoftDeletedByAuthor(ctx, authorID)
	return approvals, removals
}

func schemFlags(s *store.Schematic) []ModFlag {
	var flags []ModFlag
	if len(store.HeldGallery(s)) > 0 {
		flags = append(flags, ModFlag{Label: "Image", Level: "image"})
	}
	if s.ModerationState == store.ModerationFlagged {
		flags = append(flags, ModFlag{Label: "Content", Level: "content"})
	}
	return flags
}

func buildModQueueCard(ctx context.Context, appStore *store.Store, s *store.Schematic, selID string) ModQueueCard {
	approvals, removals := authorTrust(ctx, appStore, s.AuthorID)
	author := ""
	if u := findUserFromStore(appStore, s.AuthorID); u != nil {
		author = u.Username
	}
	return ModQueueCard{
		ID:        s.ID,
		Title:     s.Title,
		Name:      s.Name,
		Author:    author,
		Approvals: approvals,
		Removals:  removals,
		Age:       timediff.TimeDiff(s.Created),
		Flags:     schemFlags(s),
		Selected:  s.ID == selID,
	}
}

func buildModDetail(ctx context.Context, appStore *store.Store, id string) *ModDetail {
	s, err := appStore.Schematics.GetByID(ctx, id)
	if err != nil || s == nil {
		return nil
	}
	approvals, removals := authorTrust(ctx, appStore, s.AuthorID)
	author := ""
	if u := findUserFromStore(appStore, s.AuthorID); u != nil {
		author = u.Username
	}
	sanitized := s.Content
	if san, err := htmlsanitizer.NewHTMLSanitizer().SanitizeString(s.Content); err == nil {
		sanitized = san
	}
	d := &ModDetail{
		ID:               s.ID,
		Title:            s.Title,
		Name:             s.Name,
		Author:           author,
		Approvals:        approvals,
		Removals:         removals,
		Age:              timediff.TimeDiff(s.Created),
		State:            s.ModerationState,
		DetectedLanguage: s.DetectedLanguage,
		Description:      template.HTML(sanitized),
		AutoReason:       s.ModerationReason,
		HasAutoReason:    s.ModerationReason != "",
	}
	// Images: featured + gallery, marked held/removed/featured.
	held := map[string]bool{}
	for _, h := range store.HeldGallery(s) {
		held[h] = true
	}
	removed := map[string]bool{}
	for _, r := range s.RemovedImages {
		removed[r] = true
	}
	addImg := func(fn string, featured bool) {
		if fn == "" {
			return
		}
		d.Images = append(d.Images, ModImage{
			Filename: fn,
			URL:      "/api/files/schematics/" + s.ID + "/" + url.PathEscape(fn),
			Featured: featured,
			Held:     held[fn],
			Removed:  removed[fn],
		})
	}
	addImg(s.FeaturedImage, true)
	for _, g := range s.Gallery {
		if g != s.FeaturedImage {
			addImg(g, false)
		}
	}
	// Also surface held images not in gallery (e.g. a held former-featured).
	for _, h := range store.HeldGallery(s) {
		found := false
		for _, im := range d.Images {
			if im.Filename == h {
				found = true
				break
			}
		}
		if !found {
			addImg(h, false)
		}
	}
	var mats []nbtparser.Material
	if len(s.Materials) > 0 {
		_ = json.Unmarshal(s.Materials, &mats)
	}
	d.Materials = mats
	return d
}

// ModeratorDecisionHandler applies a moderator's decision (approve fully /
// publish with notes / request changes / reject fixable|final). (#1646)
func ModeratorDecisionHandler(appStore *store.Store, enqueueSearchUpsert SearchIndexEnqueuer) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		_ = e.Request.ParseForm()
		action := e.Request.FormValue("action")
		note := trimModNote(e.Request.FormValue("note"))
		kinds := e.Request.Form["kinds"]
		moderatorID := authenticatedUserID(e)

		ctx := context.Background()
		newState, _, err := applyModeratorDecision(ctx, appStore, id, moderatorID, action, note, kinds)
		if err != nil {
			return e.String(http.StatusBadRequest, err.Error())
		}
		// Reindex when the schematic became listed; drop from index otherwise.
		if enqueueSearchUpsert != nil {
			_ = enqueueSearchUpsert(ctx, id)
		}
		_ = newState

		dest := "/admin/moderation"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// ModeratorImageHandler approves (un-holds) or removes a single held image.
func ModeratorImageHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		_ = e.Request.ParseForm()
		filename := e.Request.FormValue("filename")
		action := e.Request.FormValue("action")
		if filename == "" {
			return e.String(http.StatusBadRequest, "missing filename")
		}
		ctx := context.Background()
		var err error
		switch action {
		case "approve":
			err = appStore.Schematics.ApproveHeldImage(ctx, id, filename)
		case "remove":
			err = appStore.Schematics.RemoveHeldImage(ctx, id, filename)
		default:
			return e.String(http.StatusBadRequest, "invalid action")
		}
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to update image")
		}
		if appStore.ModerationLog != nil {
			_ = appStore.ModerationLog.Create(ctx, &store.ModerationLogEntry{
				SchematicID: id, ActorID: authenticatedUserID(e), ActorType: "admin",
				Action: "image_" + action, Reason: filename,
			})
		}
		dest := "/admin/moderation?id=" + url.QueryEscape(id)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// trimModNote caps a moderator note to a sane length (rune-safe) and trims it.
func trimModNote(s string) string {
	return truncateRunes(strings.TrimSpace(s), 1000)
}
