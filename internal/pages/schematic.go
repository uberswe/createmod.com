package pages

import (
	stdctx "context"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/nbtparser"
	"createmod/internal/promotion"
	"createmod/internal/search"
	"createmod/internal/translation"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"log/slog"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/gosimple/slug"
	"github.com/mergestat/timediff"
	"createmod/internal/server"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var schematicTemplates = append([]string{
	"./template/schematic.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_full.html",
}, commonTemplates...)

type CollectionOption struct {
	ID    string
	Slug  string
	Title string
}

type SchematicData struct {
	DefaultData
	Schematic     models.Schematic
	Comments      []models.Comment
	AuthorHasMore bool
	// IsAuthor of the current schematic, for edit and delete actions
	IsAuthor        bool
	FromAuthor      []models.Schematic
	Similar         []models.Schematic
	Promotion       template.HTML
	Versions        []models.SchematicVersion
	HasVersions     bool
	UserCollections []CollectionOption
	Materials       []nbtparser.Material
	BloxelizerURL   string
	Mods            []string
	ModInfoList     []ModInfo
	// Additional files (variations)
	AdditionalFiles []store.SchematicFile
	// PendingModeration is true when the schematic is not yet moderated and the viewer is the author.
	PendingModeration bool
	// ScheduledFor is set when the schematic is scheduled for future publication and the viewer is the author.
	ScheduledFor *time.Time
	// Translation fields
	IsTranslated     bool
	OriginalLanguage string
}

// ModInfo holds display info for a mod in the Required Mods section.
type ModInfo struct {
	Namespace string
	Name      string
	IconURL   string
}

func SchematicHandler(searchService *search.Service, cacheService *cache.Service, registry *server.Registry, promotionService *promotion.Service, discordService *discord.Service, translationService *translation.Service, appStore *store.Store, webhookSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ctx := stdctx.Background()
		name := e.Request.PathValue("name")
		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil || s.Deleted != nil || s.Blacklisted {
			// Try to find and fix a schematic with percent-encoded characters in its name
			if newName, found := tryFixEncodedSchematicNameStore(appStore, name); found {
				return e.Redirect(http.StatusMovedPermanently, LangRedirectURL(e, "/schematics/"+newName))
			}
			nd := DefaultData{}
			nd.Populate(e)
			nd.Title = i18n.T(nd.Language, "Page Not Found")
			html, err := registry.LoadFiles(fourOhFourTemplates...).Render(nd)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}
		// Check scheduled_at — allow the author to view their own scheduled schematic
		isScheduled := s.ScheduledAt != nil && s.ScheduledAt.After(time.Now())
		if isScheduled {
			nd := DefaultData{}
			nd.Populate(e)
			if nd.UserID != s.AuthorID {
				nd.Title = i18n.T(nd.Language, "Page Not Found")
				html, err := registry.LoadFiles(fourOhFourTemplates...).Render(nd)
				if err != nil {
					return err
				}
				return e.HTML(http.StatusNotFound, html)
			}
		}

		d := SchematicData{
			Schematic: MapStoreSchematicToModel(appStore, *s, cacheService),
		}
		d.Populate(e)
		d.Title = d.Schematic.Title
		d.Slug = fmt.Sprintf("/schematics/%s", d.Schematic.Name)
		d.Description = strip.StripTags(d.Schematic.Content)
		d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematic.ID, d.Schematic.FeaturedImage)
		d.SubCategory = "Schematic"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Comments = findSchematicCommentsFromStore(appStore, d.Schematic.ID)
		authorID := ""
		if d.Schematic.Author != nil {
			authorID = d.Schematic.Author.ID
		}
		d.FromAuthor = findAuthorSchematicsFromStore(appStore, cacheService, d.Schematic.ID, authorID, 5)
		d.Similar = findSimilarSchematicsFromStore(appStore, cacheService, d.Schematic, d.FromAuthor, searchService)
		d.AuthorHasMore = len(d.FromAuthor) > 0
		d.IsAuthor = authorID == d.UserID
		d.PendingModeration = !s.Moderated && d.IsAuthor
		if isScheduled && d.IsAuthor {
			d.ScheduledFor = s.ScheduledAt
		}
		d.Promotion = promotionService.RandomPromotion()

		// Parse materials from stored JSON
		if s.Materials != nil {
			var materials []nbtparser.Material
			if err := json.Unmarshal(s.Materials, &materials); err == nil {
				d.Materials = materials
			}
		}

		// Load mods from the schematic record
		d.Mods = d.Schematic.Mods

		// Build enriched mod info list for display
		d.ModInfoList = buildModInfoListFromStore(appStore, d.Mods)

		// Construct Bloxelizer URL (only for free schematics with a file)
		if s.SchematicFile != "" && !d.Schematic.Paid {
			scheme := "http"
			if e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https") {
				scheme = "https"
			}
			host := e.Request.Host
			fileURL := fmt.Sprintf("%s://%s/api/files/schematics/%s/%s", scheme, host, d.Schematic.ID, s.SchematicFile)
			d.BloxelizerURL = "https://bloxelizer.com/viewer?url=" + url.QueryEscape(fileURL)
		}

		// Load collections for the current user (for Add to collection dropdown)
		if isAuthenticated(e) {
			userColls, err := appStore.Collections.ListByAuthor(ctx, authenticatedUserID(e))
			if err == nil {
				opts := make([]CollectionOption, 0, len(userColls))
				for _, c := range userColls {
					t := c.Title
					if t == "" {
						t = c.Name
					}
					opts = append(opts, CollectionOption{ID: c.ID, Slug: c.Slug, Title: t})
				}
				d.UserCollections = opts
			}
		}

		// Load recent version history (up to 10)
		storeVersions, err := appStore.Versions.ListBySchematic(ctx, d.Schematic.ID)
		if err == nil && len(storeVersions) > 0 {
			maxVersions := 10
			if len(storeVersions) < maxVersions {
				maxVersions = len(storeVersions)
			}
			versions := make([]models.SchematicVersion, 0, maxVersions)
			for i := 0; i < maxVersions; i++ {
				versions = append(versions, models.SchematicVersion{
					Version: storeVersions[i].Version,
					Created: storeVersions[i].Created,
					Note:    storeVersions[i].Note,
				})
			}
			d.Versions = versions
			d.HasVersions = true
		}

		// Load additional files (variations)
		if additionalFiles, afErr := appStore.SchematicFiles.ListBySchematicID(ctx, s.ID); afErr == nil && len(additionalFiles) > 0 {
			d.AdditionalFiles = additionalFiles
		}

		// Translation: show translated title/description if user's language differs from detected language
		detectedLang := s.DetectedLanguage
		if detectedLang == "" {
			detectedLang = "en"
		}
		d.OriginalLanguage = detectedLang
		showOriginal := e.Request.URL.Query().Get("lang") == "original"
		transSanitizer := htmlsanitizer.NewHTMLSanitizer()
		if !showOriginal && translationService != nil && d.Language != "" && d.Language != "en" {
			// User's UI language is not English - try to show a translation
			t := translationService.GetTranslationCached(cacheService, d.Schematic.ID, d.Language)
			if t != nil && t.Title != "" {
				d.Schematic.Title = t.Title
				d.Title = t.Title
				if t.Content != "" {
					d.Schematic.Content = t.Content
					sanitizedTransContent, sanitizeErr := transSanitizer.SanitizeString(strings.ReplaceAll(t.Content, "\n", "<br/>"))
					if sanitizeErr != nil {
						sanitizedTransContent = template.HTMLEscapeString(strings.ReplaceAll(t.Content, "\n", "<br/>"))
					}
					d.Schematic.HTMLContent = template.HTML(sanitizedTransContent)
				}
				d.IsTranslated = true
			}
		} else if showOriginal && translationService != nil && detectedLang != "en" {
			// User clicked "show original" - display the original language text
			t := translationService.GetTranslationCached(cacheService, d.Schematic.ID, detectedLang)
			if t != nil && t.Title != "" {
				d.Schematic.Title = t.Title
				d.Title = t.Title
				if t.Content != "" {
					d.Schematic.Content = t.Content
					sanitizedOrigContent, sanitizeErr := transSanitizer.SanitizeString(strings.ReplaceAll(t.Content, "\n", "<br/>"))
					if sanitizeErr != nil {
						sanitizedOrigContent = template.HTMLEscapeString(strings.ReplaceAll(t.Content, "\n", "<br/>"))
					}
					d.Schematic.HTMLContent = template.HTML(sanitizedOrigContent)
				}
			}
		}

		countSchematicViewStore(appStore, d.Schematic.ID, discordService, e.RealIP(), cacheService, webhookSecret, slog.Default())
		html, err := registry.LoadFiles(schematicTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}


// pctEncodedRe matches percent-encoded sequences like %cc%b6
var pctEncodedRe = regexp.MustCompile(`%[0-9a-fA-F]{2}`)

// cleanSlugName decodes percent-encoded sequences in a schematic name,
// strips non-slug characters, and produces a clean slug.
func cleanSlugName(name string) string {
	decoded, err := url.PathUnescape(name)
	if err != nil {
		decoded = name
	}
	clean := slug.Make(decoded)
	// Remove any leftover empty segments from stripping
	for strings.Contains(clean, "--") {
		clean = strings.ReplaceAll(clean, "--", "-")
	}
	clean = strings.Trim(clean, "-")
	return clean
}


// ---------------------------------------------------------------------------
// Store-based mapping and helper functions (PostgreSQL migration - Group 1)
// ---------------------------------------------------------------------------


// MapStoreSchematicToModel converts a store.Schematic to a models.Schematic,
// using the store for all lookups (user, categories, tags, versions, views,
// ratings, downloads).
func MapStoreSchematicToModel(appStore *store.Store, s store.Schematic, cacheService *cache.Service) models.Schematic {
	ctx := stdctx.Background()

	// --- Views ---
	vk := cache.ViewKey(s.ID)
	views, found := cacheService.GetInt(vk)
	if !found {
		v, err := appStore.ViewRatings.GetViewCount(ctx, s.ID)
		if err == nil && v > 0 {
			views = v
			cacheService.SetInt(vk, views)
		}
	}

	// --- Downloads ---
	dk := cache.DownloadKey(s.ID)
	downloads, found := cacheService.GetInt(dk)
	if !found {
		dl, err := appStore.ViewRatings.GetDownloadCount(ctx, s.ID)
		if err == nil && dl > 0 {
			downloads = dl
			cacheService.SetInt(dk, downloads)
		}
	}

	// --- Rating ---
	rk := cache.RatingKey(s.ID)
	rck := cache.RatingCountKey(s.ID)
	rating, found := cacheService.GetFloat(rk)
	ratingCount, found2 := cacheService.GetInt(rck)
	if !found || !found2 {
		sr, err := appStore.ViewRatings.GetRating(ctx, s.ID)
		if err == nil && sr != nil && sr.RatingCount > 0 {
			rating = sr.AvgRating
			ratingCount = sr.RatingCount
			cacheService.SetFloat(rk, rating)
			cacheService.SetInt(rck, ratingCount)
		}
	}

	// --- Author ---
	author := findUserFromStore(appStore, s.AuthorID)

	// --- Categories ---
	var categories []models.SchematicCategory
	catIDs, err := appStore.Schematics.GetCategoryIDs(ctx, s.ID)
	if err == nil && len(catIDs) > 0 {
		cats, err := appStore.Categories.GetByIDs(ctx, catIDs)
		if err == nil {
			for _, c := range cats {
				categories = append(categories, models.SchematicCategory{
					ID:   c.ID,
					Key:  c.Key,
					Name: c.Name,
				})
			}
		}
	}

	// --- Tags ---
	var tags []models.SchematicTag
	tagIDs, err := appStore.Schematics.GetTagIDs(ctx, s.ID)
	if err == nil && len(tagIDs) > 0 {
		storeTags, err := appStore.Tags.GetByIDs(ctx, tagIDs)
		if err == nil {
			for _, t := range storeTags {
				tags = append(tags, models.SchematicTag{
					ID:   t.ID,
					Key:  t.Key,
					Name: t.Name,
				})
			}
		}
	}

	// --- Minecraft version ---
	minecraftVersion := ""
	if s.MinecraftVersionID != nil && *s.MinecraftVersionID != "" {
		if gv, err := appStore.VersionLookup.GetMinecraftVersionByID(ctx, *s.MinecraftVersionID); err == nil && gv != nil {
			minecraftVersion = gv.Version
		}
	}

	// --- Create mod version ---
	createmodVersion := ""
	if s.CreatemodVersionID != nil && *s.CreatemodVersionID != "" {
		if gv, err := appStore.VersionLookup.GetCreatemodVersionByID(ctx, *s.CreatemodVersionID); err == nil && gv != nil {
			createmodVersion = gv.Version
		}
	}

	// --- Content sanitization ---
	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	sanitizedHTML, err := sanitizer.SanitizeString(strings.ReplaceAll(s.Content, "\n", "<br/>"))
	if err != nil {
		// Fallback legacy sanitizer
		sanitizedHTML = template.HTMLEscapeString(strings.ReplaceAll(s.Content, "\n", "<br/>"))
	}

	// --- Postdate formatting ---
	postdate := s.Created
	if s.Postdate != nil {
		postdate = *s.Postdate
	}

	// --- Parse mods ---
	var mods []string
	if s.Mods != nil {
		_ = json.Unmarshal(s.Mods, &mods)
	}

	// --- Schematic file URL ---
	schematicFile := ""
	if s.SchematicFile != "" {
		schematicFile = fmt.Sprintf("/api/files/schematics/%s/%s", s.ID, s.SchematicFile)
	}

	// --- Category ID (first) ---
	categoryID := ""
	if len(categories) > 0 {
		categoryID = categories[0].ID
	}

	result := models.Schematic{
		ID:                   s.ID,
		Created:              s.Created,
		CreatedFormatted:     postdate.Format(time.DateTime),
		CreatedHumanReadable: timediff.TimeDiff(postdate),
		Author:               author,
		Content:              s.Content,
		HTMLContent:          template.HTML(sanitizedHTML),
		Excerpt:              s.Excerpt,
		FeaturedImage:        s.FeaturedImage,
		Gallery:              s.Gallery,
		HasGallery:           len(s.Gallery) > 0,
		Title:                s.Title,
		Name:                 s.Name,
		Video:                s.Video,
		HasDependencies:      s.HasDependencies,
		Dependencies:         s.Dependencies,
		HTMLDependencies:     template.HTML(strings.ReplaceAll(template.HTMLEscapeString(s.Dependencies), "\n", "<br/>")),
		Categories:           categories,
		CategoryId:           categoryID,
		Tags:                 tags,
		HasTags:              len(tags) > 0,
		CreatemodVersion:     createmodVersion,
		MinecraftVersion:     minecraftVersion,
		Views:                views,
		Downloads:            downloads,
		Rating:               fmt.Sprintf("%.1f", rating),
		RatingCount:          ratingCount,
		HasRating:            rating > 0,
		SchematicFile:        schematicFile,
		AIDescription:        s.AIDescription,
		Paid:                 s.Paid,
		Featured:             s.Featured,
		Materials:            string(s.Materials),
		ExternalURL:          s.ExternalURL,
		BlockCount:           s.BlockCount,
		DimX:                 s.DimX,
		DimY:                 s.DimY,
		DimZ:                 s.DimZ,
		Mods:                 mods,
	}

	return result
}

// MapStoreSchematics converts a slice of store.Schematic to []models.Schematic,
// using the cache where possible.
func MapStoreSchematics(appStore *store.Store, schematics []store.Schematic, cacheService *cache.Service) []models.Schematic {
	var result []models.Schematic
	for i := range schematics {
		sk := cache.SchematicKey(schematics[i].ID)
		schematic, found := cacheService.GetSchematic(sk)
		if !found {
			schematic = MapStoreSchematicToModel(appStore, schematics[i], cacheService)
			cacheService.SetSchematic(sk, schematic)
		}
		result = append(result, schematic)
	}
	return result
}

// findAuthorSchematicsFromStore returns schematics by the same author,
// excluding the given schematic ID.
func findAuthorSchematicsFromStore(appStore *store.Store, cacheService *cache.Service, excludeID, authorID string, limit int) []models.Schematic {
	ctx := stdctx.Background()
	schematics, err := appStore.Schematics.ListByAuthorExcluding(ctx, authorID, excludeID, limit)
	if err != nil {
		return nil
	}
	return MapStoreSchematics(appStore, schematics, cacheService)
}

// findSchematicCommentsFromStore returns approved comments for a schematic,
// using the store which already joins user info.
func findSchematicCommentsFromStore(appStore *store.Store, schematicID string) []models.Comment {
	ctx := stdctx.Background()
	storeComments, err := appStore.Comments.ListBySchematic(ctx, schematicID)
	if err != nil {
		return nil
	}

	// Convert to DatabaseComment so we can reuse the sorting/nesting logic
	var dbComments []models.DatabaseComment
	for _, c := range storeComments {
		published := ""
		if c.Published != nil {
			published = c.Published.Format("2006-01-02 15:04:05.999Z07:00")
		}
		authorID := ""
		if c.AuthorID != nil {
			authorID = *c.AuthorID
		}
		schematicID := ""
		if c.SchematicID != nil {
			schematicID = *c.SchematicID
		}
		parentID := ""
		if c.ParentID != nil {
			parentID = *c.ParentID
		}
		dbComments = append(dbComments, models.DatabaseComment{
			ID:        c.ID,
			Created:   c.Created,
			Published: published,
			Author:    authorID,
			Schematic: schematicID,
			Karma:     c.Karma,
			Approved:  c.Approved,
			Type:      c.Type,
			ParentID:  parentID,
			Content:   c.Content,
		})
	}

	// Sort: top-level first, then by published time
	sort.Slice(dbComments, func(i, j int) bool {
		if dbComments[j].ParentID != "" && dbComments[i].ParentID == "" {
			return true
		} else if dbComments[i].ParentID != "" && dbComments[j].ParentID == "" {
			return false
		}
		t1, err := time.Parse("2006-01-02 15:04:05.999Z07:00", dbComments[i].Published)
		if err != nil {
			t1 = dbComments[i].Created
		}
		t2, err := time.Parse("2006-01-02 15:04:05.999Z07:00", dbComments[j].Published)
		if err != nil {
			t2 = dbComments[j].Created
		}
		return t1.Before(t2)
	})

	// Build comments with nesting (same logic as MapResultsToComment)
	var comments []models.Comment
	for _, c := range dbComments {
		if c.ParentID != "" {
			for i := range comments {
				if c.ParentID == comments[i].ID {
					com := mapStoreComment(c, storeComments)
					com.Indent = 1
					if i+1 == len(comments) {
						comments = append(comments, com)
					} else {
						comments = slices.Insert(comments, i+1, com)
						comments[i+1].Indent = 1
					}
					break
				}
			}
		} else {
			comments = append(comments, mapStoreComment(c, storeComments))
		}
	}
	return comments
}

// mapStoreComment converts a DatabaseComment to a models.Comment, using
// the store comments list to find author info (already joined).
func mapStoreComment(c models.DatabaseComment, storeComments []store.Comment) models.Comment {
	comment := models.Comment{
		ID:       c.ID,
		Approved: c.Approved,
		ParentID: c.ParentID,
	}

	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	sanitizedHTML, err := sanitizer.SanitizeString(c.Content)
	if err != nil {
		// Fallback legacy sanitizer
		sanitizedHTML = strings.ReplaceAll(template.HTMLEscapeString(c.Content), "\n", "<br/>")
	}
	comment.Content = template.HTML(sanitizedHTML)

	// Find the matching store comment to get author info
	for _, sc := range storeComments {
		if sc.ID == c.ID {
			comment.Author = sc.AuthorUsername
			comment.AuthorUsername = sc.AuthorUsername
			comment.AuthorAvatar = sc.AuthorAvatar
			if sc.AuthorAvatar != "" {
				comment.AuthorHasAvatar = true
			}
			break
		}
	}

	t, err := time.Parse("2006-01-02 15:04:05.999Z07:00", c.Published)
	if err != nil {
		t = c.Created
	}
	comment.Created = timediff.TimeDiff(t)
	comment.Published = t.Format(time.DateTime)

	return comment
}

// countSchematicViewStore records a view for a schematic using the store.
// It applies IP-based rate limiting via cache, sends a Discord notification
// at 50 total views, and awards view-based achievements at thresholds.
func countSchematicViewStore(appStore *store.Store, schematicID string, discordService *discord.Service, clientIP string, cacheService *cache.Service, webhookSecret string, logger interface {
	Error(string, ...any)
	Info(string, ...any)
}) {
	ctx := stdctx.Background()

	// IP-based rate limiting: skip if same IP already viewed this schematic recently
	if clientIP != "" && cacheService != nil {
		ipKey := fmt.Sprintf("viewip:%s:%s", clientIP, schematicID)
		if _, already := cacheService.Get(ipKey); already {
			return
		}
		// Mark this IP+schematic combo for 1 hour
		cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
	}

	// Record the view (handles all period types)
	if err := appStore.ViewRatings.RecordView(ctx, schematicID); err != nil {
		logger.Error("failed to record view", "schematicID", schematicID, "error", err)
		return
	}

	// Check total view count for notifications and achievements
	totalViews, err := appStore.ViewRatings.GetTotalViewCount(ctx, schematicID)
	if err != nil {
		logger.Error("failed to get total view count", "schematicID", schematicID, "error", err)
		return
	}

	// Discord notification at 50 total views
	if totalViews == 50 && discordService != nil {
		s, sErr := appStore.Schematics.GetByID(ctx, schematicID)
		if sErr == nil && s != nil && s.Moderated {
			go discordService.PostWithUserWebhooks(fmt.Sprintf("New Schematic Posted: https://createmod.com/schematics/%s", s.Name), appStore.Webhooks, webhookSecret)
		}
	}

	// Award view-based achievements at thresholds
	s, err := appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil || s == nil || !s.Moderated {
		return
	}
	authorID := s.AuthorID
	if authorID == "" {
		return
	}

	award := func(key, title, desc, icon string) {
		ach, err := appStore.Achievements.GetByKey(ctx, key)
		if err != nil || ach == nil {
			return
		}
		has, err := appStore.Achievements.HasAchievement(ctx, authorID, ach.ID)
		if err != nil || has {
			return
		}
		_ = appStore.Achievements.Award(ctx, authorID, ach.ID)
	}

	switch totalViews {
	case 100:
		award("views_100", "100 Views", "One of your schematics reached 100 total views", "eye")
		_ = appStore.Users.UpdateUserPoints(ctx, authorID, 5)
	case 1000:
		award("views_1000", "1,000 Views", "One of your schematics reached 1,000 total views", "eye")
		_ = appStore.Users.UpdateUserPoints(ctx, authorID, 25)
	case 10000:
		award("views_10000", "10,000 Views", "One of your schematics reached 10,000 total views", "eye")
		_ = appStore.Users.UpdateUserPoints(ctx, authorID, 100)
	}
}

// buildModInfoListFromStore builds an enriched list of mod display info
// from namespaces using the store.
func buildModInfoListFromStore(appStore *store.Store, mods []string) []ModInfo {
	ctx := stdctx.Background()
	caser := cases.Title(language.English)
	list := make([]ModInfo, 0, len(mods))
	for _, ns := range mods {
		info := ModInfo{
			Namespace: ns,
			Name:      caser.String(strings.ReplaceAll(ns, "_", " ")),
		}
		meta, err := appStore.ModMetadata.GetByNamespace(ctx, ns)
		if err == nil && meta != nil {
			if meta.DisplayName != "" {
				info.Name = meta.DisplayName
			}
			info.IconURL = meta.IconURL
		}
		list = append(list, info)
	}
	return list
}

// tryFixEncodedSchematicNameStore searches for schematics whose name contains
// percent-encoded characters via the store. If one is found whose decoded
// name matches the requested path, it updates the name and returns the new
// name so the caller can redirect.
func tryFixEncodedSchematicNameStore(appStore *store.Store, requestedName string) (string, bool) {
	ctx := stdctx.Background()

	// Find schematics with literal percent in the name
	recs, err := appStore.Schematics.ListByNamePattern(ctx, "%", 200)
	if err != nil || len(recs) == 0 {
		return "", false
	}

	requestedSlug := slug.Make(requestedName)

	for _, rec := range recs {
		dbName := rec.Name
		if !pctEncodedRe.MatchString(dbName) {
			continue
		}
		// Decode the DB name to get the unicode version
		decoded, err := url.PathUnescape(dbName)
		if err != nil {
			continue
		}
		decodedSlug := slug.Make(decoded)
		// Compare using multiple strategies
		if decoded != requestedName && decodedSlug != requestedName && decodedSlug != requestedSlug && dbName != requestedName {
			continue
		}
		// Generate a clean name
		newName := cleanSlugName(dbName)
		if newName == "" || newName == dbName {
			continue
		}
		// Ensure the new name is unique
		existing, _ := appStore.Schematics.GetByName(ctx, newName)
		if existing != nil && existing.ID != rec.ID {
			// Append a suffix to make it unique
			for i := 2; i < 100; i++ {
				candidate := fmt.Sprintf("%s-%d", newName, i)
				ex, _ := appStore.Schematics.GetByName(ctx, candidate)
				if ex == nil {
					newName = candidate
					break
				}
			}
		}
		// Update the record
		if err := appStore.Schematics.UpdateName(ctx, rec.ID, newName); err != nil {
			continue
		}
		return newName, true
	}
	return "", false
}

// findSimilarByCategoryFromStore returns schematics that share at least one
// category with the given schematic, ordered by most views. Used as a
// fallback when the full-text search index is not yet available.
func findSimilarByCategoryFromStore(appStore *store.Store, cacheService *cache.Service, schematic models.Schematic, exclude map[string]struct{}, limit int) []models.Schematic {
	if len(schematic.Categories) == 0 {
		return nil
	}
	ctx := stdctx.Background()

	catIDs := make([]string, 0, len(schematic.Categories))
	for _, c := range schematic.Categories {
		catIDs = append(catIDs, c.ID)
	}

	excludeIDs := make([]string, 0, len(exclude))
	for id := range exclude {
		excludeIDs = append(excludeIDs, id)
	}

	// Fetch schematics sharing categories, excluding the specified IDs
	storeSchematics, err := appStore.Schematics.ListByCategoryIDs(ctx, catIDs, excludeIDs, limit)
	if err != nil {
		return nil
	}

	results := MapStoreSchematics(appStore, storeSchematics, cacheService)
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// findSimilarSchematicsFromStore uses the search service for IDs, then
// looks up schematics via the store.
func findSimilarSchematicsFromStore(appStore *store.Store, cacheService *cache.Service, schematic models.Schematic, author []models.Schematic, searchService *search.Service) []models.Schematic {
	const limit = 5

	// Build exclude set: current schematic + author schematics.
	exclude := make(map[string]struct{}, 1+len(author))
	exclude[schematic.ID] = struct{}{}
	for _, a := range author {
		exclude[a.ID] = struct{}{}
	}

	// Try Bleve full-text search first.
	keywordString := ""
	for _, t := range schematic.Tags {
		keywordString += " " + t.Name
	}
	for _, c := range schematic.Categories {
		keywordString += " " + c.Name
	}
	ids := searchService.Search(fmt.Sprintf("%s %s", schematic.Title, keywordString), search.BestMatchOrder, -1, "all", nil, "all", "all", false)

	wantIDs := make([]string, 0, limit)
	for _, id := range ids {
		if len(wantIDs) >= limit {
			break
		}
		if _, skip := exclude[id]; skip {
			continue
		}
		wantIDs = append(wantIDs, id)
	}

	// If search index returned results, query store and preserve search ranking.
	if len(wantIDs) > 0 {
		ctx := stdctx.Background()
		storeSchematics, err := appStore.Schematics.ListByIDs(ctx, wantIDs)
		if err != nil {
			return nil
		}
		schematicModels := MapStoreSchematics(appStore, storeSchematics, cacheService)
		// Re-sort to match the search ranking order.
		sortedModels := make([]models.Schematic, 0, len(schematicModels))
		for _, wantID := range wantIDs {
			for i := range schematicModels {
				if wantID == schematicModels[i].ID {
					sortedModels = append(sortedModels, schematicModels[i])
					break
				}
			}
		}
		return sortedModels
	}

	// Fallback: search index empty/unavailable, query store by shared categories.
	return findSimilarByCategoryFromStore(appStore, cacheService, schematic, exclude, limit)
}
