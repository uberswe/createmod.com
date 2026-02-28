package pages

import (
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/models"
	"createmod/internal/nbtparser"
	"createmod/internal/promotion"
	"createmod/internal/search"
	"createmod/internal/translation"
	"encoding/json"
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/gosimple/slug"
	"github.com/mergestat/timediff"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	template2 "github.com/pocketbase/pocketbase/tools/template"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
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
	// Translation fields
	IsTranslated     bool
	OriginalLanguage string
}

func SchematicHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template2.Registry, promotionService *promotion.Service, discordService *discord.Service, translationService *translation.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"name = {:name} && deleted = '' && (scheduled_at = null || scheduled_at <= {:now})",
			"-created",
			1,
			0,
			dbx.Params{"name": e.Request.PathValue("name"), "now": time.Now()})

		if len(results) != 1 {
			// Try to find and fix a schematic with percent-encoded characters in its name
			if newName, found := tryFixEncodedSchematicName(app, e.Request.PathValue("name")); found {
				return e.Redirect(http.StatusMovedPermanently, "/schematics/"+newName)
			}
			html, err := registry.LoadFiles(fourOhFourTemplates...).Render(nil)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		d := SchematicData{
			Schematic: mapResultToSchematic(app, results[0], cacheService),
		}
		d.Populate(e)
		d.Title = d.Schematic.Title
		d.Slug = fmt.Sprintf("schematics/%s", d.Schematic.Name)
		d.Description = strip.StripTags(d.Schematic.Content)
		d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematic.ID, d.Schematic.FeaturedImage)
		d.SubCategory = "Schematic"
		d.Categories = allCategories(app, cacheService)
		d.Comments = findSchematicComments(app, d.Schematic.ID)
		d.FromAuthor = findAuthorSchematics(app, cacheService, d.Schematic.ID, d.Schematic.Author.ID, 5, "@random")
		d.Similar = findSimilarSchematics(app, cacheService, d.Schematic, d.FromAuthor, searchService)
		d.AuthorHasMore = len(d.FromAuthor) > 0
		d.IsAuthor = d.Schematic.Author.ID == d.UserID
		d.Promotion = promotionService.RandomPromotion()

		// Parse materials from stored JSON
		materialsJSON := results[0].GetString("materials")
		if materialsJSON != "" {
			var materials []nbtparser.Material
			if err := json.Unmarshal([]byte(materialsJSON), &materials); err == nil {
				d.Materials = materials
			}
		}

		// Load mods from the schematic record
		d.Mods = d.Schematic.Mods

		// Construct Bloxelizer URL (only for free schematics with a file)
		schematicFileName := results[0].GetString("schematic_file")
		if schematicFileName != "" && !d.Schematic.Paid {
			scheme := "http"
			if e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https") {
				scheme = "https"
			}
			host := e.Request.Host
			fileURL := fmt.Sprintf("%s://%s/api/files/schematics/%s/%s", scheme, host, d.Schematic.ID, schematicFileName)
			d.BloxelizerURL = "https://bloxelizer.com/viewer?url=" + url.QueryEscape(fileURL)
		}

		// Load collections for the current user (for Add to collection dropdown)
		if e.Auth != nil {
			if coll, err := app.FindCollectionByNameOrId("collections"); err == nil && coll != nil {
				recs, _ := app.FindRecordsByFilter(coll.Id, "author = {:a} && deleted = ''", "+title", 200, 0, dbx.Params{"a": e.Auth.Id})
				opts := make([]CollectionOption, 0, len(recs))
				for _, r := range recs {
					t := r.GetString("title")
					if t == "" {
						t = r.GetString("name")
					}
					opts = append(opts, CollectionOption{ID: r.Id, Slug: r.GetString("slug"), Title: t})
				}
				d.UserCollections = opts
			}
		}

		// Load recent version history (up to 10)
		verRecs, err := app.FindRecordsByFilter("schematic_versions", "schematic = {:id}", "-version", 10, 0, dbx.Params{"id": d.Schematic.ID})
		if err == nil && len(verRecs) > 0 {
			versions := make([]models.SchematicVersion, 0, len(verRecs))
			for i := range verRecs {
				versions = append(versions, models.SchematicVersion{
					Version: verRecs[i].GetInt("version"),
					Created: verRecs[i].GetDateTime("created").Time(),
					Note:    verRecs[i].GetString("note"),
				})
			}
			d.Versions = versions
			d.HasVersions = true
		}

		// Translation: show translated title/description if user's language differs from detected language
		detectedLang := results[0].GetString("detected_language")
		if detectedLang == "" {
			detectedLang = "en"
		}
		d.OriginalLanguage = detectedLang
		showOriginal := e.Request.URL.Query().Get("lang") == "original"
		if !showOriginal && translationService != nil && d.Language != "" && d.Language != "en" {
			// User's UI language is not English - try to show a translation
			t := translationService.GetTranslation(app, cacheService, d.Schematic.ID, d.Language)
			if t != nil && t.Title != "" {
				d.Schematic.Title = t.Title
				d.Title = t.Title
				if t.Content != "" {
					d.Schematic.Content = t.Content
					d.Schematic.HTMLContent = template.HTML(t.Content)
				}
				d.IsTranslated = true
			}
		} else if showOriginal && translationService != nil && detectedLang != "en" {
			// User clicked "show original" - display the original language text
			t := translationService.GetTranslation(app, cacheService, d.Schematic.ID, detectedLang)
			if t != nil && t.Title != "" {
				d.Schematic.Title = t.Title
				d.Title = t.Title
				if t.Content != "" {
					d.Schematic.Content = t.Content
					d.Schematic.HTMLContent = template.HTML(t.Content)
				}
			}
		}

		countSchematicView(app, results[0], discordService, e.RealIP(), cacheService)
		html, err := registry.LoadFiles(schematicTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func findAuthorSchematics(app *pocketbase.PocketBase, cacheService *cache.Service, id string, authorID string, limit int, sortBy string) []models.Schematic {
	schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
	if err != nil {
		return nil
	}
	results, err := app.FindRecordsByFilter(
		schematicsCollection.Id,
		"id != {:id} && author = {:authorID} && deleted = '' && moderated = true && (scheduled_at = null || scheduled_at <= {:now})",
		sortBy,
		limit,
		0,
		dbx.Params{"id": id, "authorID": authorID, "now": time.Now()})
	return MapResultsToSchematic(app, results, cacheService)
}

func findSimilarSchematics(app *pocketbase.PocketBase, cacheService *cache.Service, schematic models.Schematic, author []models.Schematic, searchService *search.Service) []models.Schematic {
	const limit = 5

	// Build exclude set: current schematic + author's schematics.
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
	ids := searchService.Search(fmt.Sprintf("%s %s", schematic.Title, keywordString), search.BestMatchOrder, -1, "all", nil, "all", "all")

	interfaceIds := make([]interface{}, 0, limit)
	for _, id := range ids {
		if len(interfaceIds) >= limit {
			break
		}
		if _, skip := exclude[id]; skip {
			continue
		}
		interfaceIds = append(interfaceIds, id)
	}

	// If search index returned results, query DB and preserve search ranking.
	if len(interfaceIds) > 0 {
		var res []*core.Record
		err := app.RecordQuery("schematics").
			Select("schematics.*").
			From("schematics").
			Where(dbx.NewExp("(deleted = '' OR deleted IS NULL) AND moderated = true AND (scheduled_at IS NULL OR scheduled_at <= DATETIME('now'))")).
			Where(dbx.In("id", interfaceIds...)).
			All(&res)
		if err != nil {
			return nil
		}
		schematicModels := MapResultsToSchematic(app, res, cacheService)
		// Re-sort to match the search ranking order.
		sortedModels := make([]models.Schematic, 0, len(schematicModels))
		for _, wantID := range interfaceIds {
			for i := range schematicModels {
				if wantID.(string) == schematicModels[i].ID {
					sortedModels = append(sortedModels, schematicModels[i])
					break
				}
			}
		}
		return sortedModels
	}

	// Fallback: search index empty/unavailable — query DB by shared categories.
	return findSimilarByCategory(app, cacheService, schematic, exclude, limit)
}

// findSimilarByCategory returns schematics that share at least one category
// with the given schematic, ordered by most views. Used as a fallback when the
// full-text search index is not yet available.
func findSimilarByCategory(app *pocketbase.PocketBase, cacheService *cache.Service, schematic models.Schematic, exclude map[string]struct{}, limit int) []models.Schematic {
	if len(schematic.Categories) == 0 {
		return nil
	}
	catIDs := make([]interface{}, 0, len(schematic.Categories))
	for _, c := range schematic.Categories {
		catIDs = append(catIDs, c.ID)
	}

	// PocketBase stores multi-relation fields as JSON arrays in SQLite.
	// We use a LIKE-based OR query to find schematics that contain any of
	// the same category IDs.
	conditions := make([]dbx.Expression, 0, len(catIDs))
	for _, cid := range catIDs {
		conditions = append(conditions, dbx.NewExp("categories LIKE {:cat"+cid.(string)+"}", dbx.Params{"cat" + cid.(string): "%" + cid.(string) + "%"}))
	}

	excludeIDs := make([]interface{}, 0, len(exclude))
	for id := range exclude {
		excludeIDs = append(excludeIDs, id)
	}

	q := app.RecordQuery("schematics").
		Select("schematics.*").
		From("schematics").
		Where(dbx.NewExp("(deleted = '' OR deleted IS NULL) AND moderated = true AND (scheduled_at IS NULL OR scheduled_at <= DATETIME('now'))")).
		Where(dbx.Or(conditions...)).
		OrderBy("views DESC").
		Limit(int64(limit + len(exclude)))

	if len(excludeIDs) > 0 {
		q = q.Where(dbx.NotIn("id", excludeIDs...))
	}

	var res []*core.Record
	if err := q.All(&res); err != nil {
		return nil
	}
	results := MapResultsToSchematic(app, res, cacheService)
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func findSchematicComments(app *pocketbase.PocketBase, id string) []models.Comment {
	commentsCollection, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		return nil
	}
	// Limit comments to 1000 for now, will add pagination later
	results, err := app.FindRecordsByFilter(
		commentsCollection.Id,
		"schematic = {:id} && approved = 1",
		"-created",
		1000,
		0,
		dbx.Params{"id": id})

	var comments []models.DatabaseComment

	for _, result := range results {
		comments = append(comments, models.DatabaseComment{
			ID:        result.Id,
			Created:   result.GetDateTime("created").Time(),
			Published: result.GetString("published"),
			Author:    result.GetString("author"),
			Schematic: result.GetString("schematic"),
			Karma:     result.GetInt("karma"),
			Approved:  result.GetBool("approved"),
			Type:      result.GetString("type"),
			ParentID:  result.GetString("parent"),
			Content:   result.GetString("content"),
		})
	}
	return MapResultsToComment(app, comments)
}

func MapResultsToComment(app *pocketbase.PocketBase, cs []models.DatabaseComment) []models.Comment {
	var comments []models.Comment
	// comments that are replies should come last
	sort.Slice(cs, func(i, j int) bool {
		if cs[j].ParentID != "" && cs[i].ParentID == "" {
			return true
		} else if cs[i].ParentID != "" && cs[j].ParentID == "" {
			return false
		}
		t1, err := time.Parse("2006-01-02 15:04:05.999Z07:00", cs[i].Published)
		if err != nil {
			t1 = cs[i].Created
		}
		t2, err := time.Parse("2006-01-02 15:04:05.999Z07:00", cs[j].Published)
		if err != nil {
			t2 = cs[j].Created
		}
		return t1.Before(t2)
	})
	for _, c := range cs {
		if c.ParentID != "" {
			for i := range comments {
				if c.ParentID == comments[i].ID {
					if i+1 == len(comments) {
						com := mapResultToComment(app, c)
						com.Indent = 1
						comments = append(comments, com)
						break
					} else {
						comments = slices.Insert(comments, i+1, mapResultToComment(app, c))
						comments[i+1].Indent = 1
						break
					}
				}
			}
		} else {
			comments = append(comments, mapResultToComment(app, c))
		}
	}
	return comments
}

func mapResultToComment(app *pocketbase.PocketBase, c models.DatabaseComment) models.Comment {
	comment := models.Comment{
		ID:       c.ID,
		Approved: c.Approved,
		ParentID: c.ParentID,
	}

	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	sanitizedHTML, err := sanitizer.SanitizeString(c.Content)
	if err != nil {
		app.Logger().Debug("Failed to sanitize", "string", c.Content, "error", err)
		// Fallback legacy sanitizer
		sanitizedHTML = strings.ReplaceAll(template.HTMLEscapeString(c.Content), "\n", "<br/>")
	}

	comment.Content = template.HTML(sanitizedHTML)

	userRecord, err := app.FindRecordById("users", c.Author)
	if err != nil {
		return comment
	}
	comment.Author = userRecord.GetString("name")
	comment.AuthorUsername = userRecord.GetString("username")
	if comment.Author == "" {
		comment.Author = comment.AuthorUsername
	}
	comment.AuthorAvatar = userRecord.GetString("avatar")
	if comment.AuthorAvatar != "" {
		comment.AuthorHasAvatar = true
	}

	t, err := time.Parse("2006-01-02 15:04:05.999Z07:00", c.Published)
	if err != nil {
		t = c.Created
	}
	comment.Created = timediff.TimeDiff(t)
	comment.Published = t.Format(time.DateTime)

	return comment
}

func countSchematicView(app *pocketbase.PocketBase, schematic *core.Record, discordService *discord.Service, clientIP string, cacheService *cache.Service) {
	// IP-based rate limiting: skip if same IP already viewed this schematic recently
	if clientIP != "" && cacheService != nil {
		ipKey := fmt.Sprintf("viewip:%s:%s", clientIP, schematic.Id)
		if _, already := cacheService.Get(ipKey); already {
			return
		}
		// Mark this IP+schematic combo for 1 hour
		cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
	}

	schematicViewsCollection, err := app.FindCollectionByNameOrId("schematic_views")
	if err != nil {
		app.Logger().Error(err.Error())
		return
	}

	now := time.Now()

	year, week := now.ISOWeek()
	month := now.Month()
	day := now.Day()

	types := map[int]string{
		4: "total",
		3: fmt.Sprintf("%d", year),
		2: fmt.Sprintf("%d%02d", year, month),
		1: fmt.Sprintf("%d%02d", year, week),
		0: fmt.Sprintf("%d%02d%02d", year, month, day),
	}

	for t, p := range types {
		viewsRes, err := app.FindRecordsByFilter(
			schematicViewsCollection.Id,
			"schematic = {:schematic} && type = {:type} && period = {:period}",
			"-created",
			1,
			0,
			dbx.Params{
				"schematic": schematic.Id,
				"type":      t,
				"period":    p,
			})

		if err != nil || len(viewsRes) == 0 {
			if err != nil {
				app.Logger().Error(err.Error())
			}
			record := core.NewRecord(schematicViewsCollection)
			record.Set("schematic", schematic.Id)
			record.Set("count", 1)
			record.Set("type", t)
			record.Set("period", p)

			if err = app.Save(record); err != nil {
				app.Logger().Error(err.Error())
				return
			}
			continue
		}

		viewRecord := viewsRes[0]
		count := viewRecord.GetInt("count")
		moderated := schematic.GetBool("moderated")
		if count == 50 && t == 4 && moderated {
			go sendToDiscord(schematic, discordService)
		}
		// increment first
		newCount := count + 1
		viewRecord.Set("count", newCount)
		if err = app.Save(viewRecord); err != nil {
			app.Logger().Error(err.Error())
		} else {
			// award view-based achievements at thresholds for total views
			if t == 4 && moderated {
				authorID := schematic.GetString("author")
				if authorID != "" {
					award := func(key, title, desc, icon string) {
						achID := ""
						if achColl, err := app.FindCollectionByNameOrId("achievements"); err == nil && achColl != nil {
							if a, _ := app.FindRecordsByFilter(achColl.Id, "key = {:k}", "-created", 1, 0, dbx.Params{"k": key}); len(a) > 0 {
								achID = a[0].Id
							} else {
								rec := core.NewRecord(achColl)
								rec.Set("key", key)
								rec.Set("title", title)
								rec.Set("description", desc)
								rec.Set("icon", icon)
								if err := app.Save(rec); err == nil {
									achID = rec.Id
								}
							}
							if achID != "" {
								if uaColl, err := app.FindCollectionByNameOrId("user_achievements"); err == nil && uaColl != nil {
									if ua, _ := app.FindRecordsByFilter(uaColl.Id, "user = {:u} && achievement = {:a}", "-created", 1, 0, dbx.Params{"u": authorID, "a": achID}); len(ua) == 0 {
										rec := core.NewRecord(uaColl)
										rec.Set("user", authorID)
										rec.Set("achievement", achID)
										_ = app.Save(rec)
									}
								}
							}
						}
					}
					// thresholds
					switch newCount {
					case 100:
						award("views_100", "100 Views", "One of your schematics reached 100 total views", "eye")
						// points: +5 for 100 total views
						if u, err := app.FindRecordById("_pb_users_auth_", authorID); err == nil && u != nil {
							u.Set("points", u.GetInt("points")+5)
							_ = app.Save(u)
						}
					case 1000:
						award("views_1000", "1,000 Views", "One of your schematics reached 1,000 total views", "eye")
						// points: +25 for 1,000 total views
						if u, err := app.FindRecordById("_pb_users_auth_", authorID); err == nil && u != nil {
							u.Set("points", u.GetInt("points")+25)
							_ = app.Save(u)
						}
					case 10000:
						award("views_10000", "10,000 Views", "One of your schematics reached 10,000 total views", "eye")
						// points: +100 for 10,000 total views
						if u, err := app.FindRecordById("_pb_users_auth_", authorID); err == nil && u != nil {
							u.Set("points", u.GetInt("points")+100)
							_ = app.Save(u)
						}
					}
				}
			}
		}
	}
}

func sendToDiscord(schematic *core.Record, discordService *discord.Service) {
	discordService.Post(fmt.Sprintf("New Schematic Posted: https://createmod.com/schematics/%s", schematic.GetString("name")))
}

func MapResultsToSchematic(app *pocketbase.PocketBase, results []*core.Record, cacheService *cache.Service) (schematics []models.Schematic) {
	for i := range results {
		if results[i] == nil || results[i].Id == "" || !results[i].GetDateTime("deleted").IsZero() {
			continue
		}
		sk := cache.SchematicKey(results[i].Id)
		schematic, found := cacheService.GetSchematic(sk)
		if !found {
			schematic = mapResultToSchematic(app, results[i], cacheService)
			schematics = append(schematics, schematic)
			cacheService.SetSchematic(sk, schematic)
		} else {
			schematics = append(schematics, schematic)
		}
	}
	return schematics
}

func mapResultToSchematic(app *pocketbase.PocketBase, result *core.Record, cacheService *cache.Service) (schematic models.Schematic) {
	schematicId := result.Id
	vk := cache.ViewKey(schematicId)
	views, found := cacheService.GetInt(vk)
	if !found {
		records, err := app.FindRecordsByFilter(
			"schematic_views",
			"period = 'total' && schematic = {:schematic}",
			"-updated",
			1,
			0,
			dbx.Params{"schematic": schematicId},
		)

		if err == nil && len(records) > 0 {
			views = records[0].GetInt("count")
			if views > 0 {
				cacheService.SetInt(vk, views)
			}
		}
	}
	dk := cache.DownloadKey(schematicId)
	downloads, found := cacheService.GetInt(dk)
	if !found {
		dlRecords, err := app.FindRecordsByFilter(
			"schematic_downloads",
			"period = 'total' && schematic = {:schematic}",
			"-updated",
			1,
			0,
			dbx.Params{"schematic": schematicId},
		)
		if err == nil && len(dlRecords) > 0 {
			downloads = dlRecords[0].GetInt("count")
			if downloads > 0 {
				cacheService.SetInt(dk, downloads)
			}
		}
	}

	rk := cache.RatingKey(schematicId)
	rck := cache.RatingCountKey(schematicId)
	rating, found := cacheService.GetFloat(rk)
	ratingCount, found2 := cacheService.GetInt(rck)
	if !found || !found2 {
		totalRating := float64(0)

		ratings, err := app.FindRecordsByFilter(
			"schematic_ratings",
			"schematic = {:schematic}",
			"-updated",
			1000,
			0,
			dbx.Params{"schematic": schematicId},
		)
		if err == nil {
			for i := range ratings {
				totalRating += ratings[i].GetFloat("rating")
			}
			if len(ratings) > 0 {
				rating = totalRating / float64(len(ratings))
				cacheService.SetFloat(rk, rating)
				ratingCount = len(ratings)
				cacheService.SetInt(rck, ratingCount)
			}
		}
	}

	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	sanitizedHTML, err := sanitizer.SanitizeString(strings.ReplaceAll(result.GetString("content"), "\n", "<br/>"))
	if err != nil {
		app.Logger().Debug("Failed to sanitize", "string", result.GetString("content"), "error", err)
		// Fallback legacy sanitizer
		sanitizedHTML = template.HTMLEscapeString(strings.ReplaceAll(result.GetString("content"), "\n", "<br/>"))
	}

	s := models.Schematic{
		ID:                   schematicId,
		Created:              result.GetDateTime("created").Time(),
		CreatedFormatted:     result.GetDateTime("postdate").Time().Format(time.DateTime),
		CreatedHumanReadable: timediff.TimeDiff(result.GetDateTime("postdate").Time()),
		Author:               findUserFromID(app, result.GetString("author")),
		CommentCount:         result.GetInt("comment_count"),
		CommentStatus:        result.GetBool("comment_status"),
		Content:              result.GetString("content"),
		HTMLContent:          template.HTML(sanitizedHTML),
		Excerpt:              result.GetString("excerpt"),
		FeaturedImage:        result.GetString("featured_image"),
		Gallery:              result.GetStringSlice("gallery"),
		HasGallery:           len(result.GetStringSlice("gallery")) > 0,
		Title:                result.GetString("title"),
		Name:                 result.GetString("name"),
		Video:                result.GetString("video"),
		HasDependencies:      result.GetBool("has_dependencies"),
		Dependencies:         result.GetString("dependencies"),
		HTMLDependencies:     template.HTML(strings.ReplaceAll(template.HTMLEscapeString(result.GetString("dependencies")), "\n", "<br/>")),
		Categories:           findCategoriesFromIDs(app, result.GetStringSlice("categories")),
		Tags:                 findTagsFromIDs(app, result.GetStringSlice("tags")),
		CreatemodVersion:     findCreateModVersionFromID(app, result.GetString("createmod_version")),
		MinecraftVersion:     findMinecraftVersionFromID(app, result.GetString("minecraft_version")),
		Views:                views,
		Downloads:            downloads,
		Rating:               fmt.Sprintf("%.1f", rating),
		RatingCount:          ratingCount,
		SchematicFile:        fmt.Sprintf("/api/files/%s/%s", result.BaseFilesPath(), result.GetString("schematic_file")),
		AIDescription:        result.GetString("ai_description"),
		Paid:                 result.GetBool("paid"),
		Featured:             result.GetBool("featured"),
		Materials:            result.GetString("materials"),
		ExternalURL:          result.GetString("external_url"),
		BlockCount:           result.GetInt("block_count"),
		DimX:                 result.GetInt("dim_x"),
		DimY:                 result.GetInt("dim_y"),
		DimZ:                 result.GetInt("dim_z"),
	}

	// Parse mods from stored JSON
	modsRaw := result.Get("mods")
	if modsRaw != nil {
		if b, err := json.Marshal(modsRaw); err == nil {
			var mods []string
			if err := json.Unmarshal(b, &mods); err == nil {
				s.Mods = mods
			}
		}
	}
	if len(result.GetStringSlice("categories")) > 0 {
		s.CategoryId = result.GetStringSlice("categories")[0]
	}
	s.HasTags = len(s.Tags) > 0
	s.HasRating = s.Rating != ""
	return s
}

func findMinecraftVersionFromID(app *pocketbase.PocketBase, id string) string {
	record, err := app.FindRecordById("minecraft_versions", id)
	if err != nil {
		return ""
	}
	return record.GetString("version")
}

func findCreateModVersionFromID(app *pocketbase.PocketBase, id string) string {
	record, err := app.FindRecordById("createmod_versions", id)
	if err != nil {
		return ""
	}
	return record.GetString("version")
}

func findTagsFromIDs(app *pocketbase.PocketBase, s []string) []models.SchematicTag {
	tagsCollection, err := app.FindCollectionByNameOrId("schematic_tags")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByIds(tagsCollection.Id, s)
	if err != nil {
		return nil
	}
	return mapResultToTags(records)
}

func mapResultToTags(records []*core.Record) (tags []models.SchematicTag) {
	for i := range records {
		tags = append(tags, models.SchematicTag{
			ID:   records[i].Id,
			Key:  records[i].GetString("key"),
			Name: records[i].GetString("name"),
		})
	}
	return tags
}

func findCategoriesFromIDs(app *pocketbase.PocketBase, s []string) []models.SchematicCategory {
	categoriesCollection, err := app.FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByIds(categoriesCollection.Id, s)
	if err != nil {
		return nil
	}
	return mapResultToCategories(records)
}

func mapResultToCategories(records []*core.Record) (categories []models.SchematicCategory) {
	for i := range records {
		categories = append(categories, models.SchematicCategory{
			ID:   records[i].Id,
			Key:  records[i].GetString("key"),
			Name: records[i].GetString("name"),
		})
	}
	return categories
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

// tryFixEncodedSchematicName searches for schematics whose name contains
// percent-encoded characters. If one is found whose decoded name matches
// the requested path, it updates the record's name to a clean slug and
// returns the new name so the caller can redirect.
func tryFixEncodedSchematicName(app *pocketbase.PocketBase, requestedName string) (string, bool) {
	coll, err := app.FindCollectionByNameOrId("schematics")
	if err != nil {
		return "", false
	}
	// Find schematics with literal '%' in the name that are not deleted.
	// PocketBase's ~ operator treats % as LIKE wildcard, so we use a raw query.
	var recs []*core.Record
	err = app.RecordQuery(coll).
		Where(dbx.And(
			dbx.NewExp("schematics.name LIKE {:pct} ESCAPE '\\'", dbx.Params{"pct": "%\\%%"}),
			dbx.NewExp("(schematics.deleted = '' OR schematics.deleted IS NULL)"),
		)).
		Limit(200).
		All(&recs)
	if err != nil || len(recs) == 0 {
		return "", false
	}

	requestedSlug := slug.Make(requestedName)

	for _, rec := range recs {
		dbName := rec.GetString("name")
		if !pctEncodedRe.MatchString(dbName) {
			continue
		}
		// Decode the DB name to get the unicode version
		decoded, err := url.PathUnescape(dbName)
		if err != nil {
			continue
		}
		decodedSlug := slug.Make(decoded)
		// The browser decodes %cc%b6 etc. before sending, then Go's router
		// decodes the path again. Compare using multiple strategies:
		// 1. Direct match with decoded unicode string
		// 2. Slugified versions (strips combining chars etc.)
		// 3. Direct match with raw DB name
		if decoded != requestedName && decodedSlug != requestedName && decodedSlug != requestedSlug && dbName != requestedName {
			continue
		}
		// Generate a clean name
		newName := cleanSlugName(dbName)
		if newName == "" || newName == dbName {
			continue
		}
		// Ensure the new name is unique
		existing, _ := app.FindRecordsByFilter(coll.Id, "name = {:n} && id != {:id}", "", 1, 0, dbx.Params{"n": newName, "id": rec.Id})
		if len(existing) > 0 {
			// Append a suffix to make it unique
			for i := 2; i < 100; i++ {
				candidate := fmt.Sprintf("%s-%d", newName, i)
				existing, _ = app.FindRecordsByFilter(coll.Id, "name = {:n} && id != {:id}", "", 1, 0, dbx.Params{"n": candidate, "id": rec.Id})
				if len(existing) == 0 {
					newName = candidate
					break
				}
			}
		}
		// Update the record
		rec.Set("name", newName)
		if err := app.Save(rec); err != nil {
			app.Logger().Error("failed to fix encoded schematic name", "id", rec.Id, "old", dbName, "new", newName, "error", err)
			continue
		}
		app.Logger().Info("fixed encoded schematic name", "id", rec.Id, "old", dbName, "new", newName)
		return newName, true
	}
	return "", false
}
