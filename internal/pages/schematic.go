package pages

import (
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/models"
	"createmod/internal/promotion"
	"createmod/internal/search"
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/mergestat/timediff"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	template2 "github.com/pocketbase/pocketbase/tools/template"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

var schematicTemplates = []string{
	"./template/dist/schematic.html",
	"./template/dist/include/schematic_card.html",
}

type SchematicData struct {
	DefaultData
	Schematic     models.Schematic
	Comments      []models.Comment
	AuthorHasMore bool
	// IsAuthor of the current schematic, for edit and delete actions
	IsAuthor   bool
	FromAuthor []models.Schematic
	Similar    []models.Schematic
	Promotion  template.HTML
}

func SchematicHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template2.Registry, promotionService *promotion.Service, discordService *discord.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"name = {:name} && deleted = null",
			"-created",
			1,
			0,
			dbx.Params{"name": e.Request.PathValue("name")})

		if len(results) != 1 {
			html, err := registry.LoadFiles(fourOhFourTemplate).Render(nil)
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

		countSchematicView(app, results[0], discordService)
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
		"id != {:id} && author = {:authorID} && deleted = null && moderated = true",
		sortBy,
		limit,
		0,
		dbx.Params{"id": id, "authorID": authorID})
	return MapResultsToSchematic(app, results, cacheService)
}

func findSimilarSchematics(app *pocketbase.PocketBase, cacheService *cache.Service, schematic models.Schematic, author []models.Schematic, searchService *search.Service) []models.Schematic {
	// Does title and content give the best match? Maybe tags + category?
	keywordString := ""
	for _, t := range schematic.Tags {
		keywordString += " "
		keywordString = keywordString + t.Name
	}
	for _, c := range schematic.Categories {
		keywordString += " "
		keywordString = keywordString + c.Name
	}
	ids := searchService.Search(fmt.Sprintf("%s %s", schematic.Title, keywordString), 1, -1, "all", "all", "all", "all")
	interfaceIds := make([]interface{}, 0, len(ids))
	limit := 5
	count := 0
	for _, id := range ids {
		if count > limit {
			break
		}
		if id == schematic.ID {
			continue
		}
		found := false
		for _, a := range author {
			if id == a.ID {
				found = true
			}
		}
		if found {
			continue
		}
		interfaceIds = append(interfaceIds, id)
		count++
	}

	var res []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*").
		From("schematics").
		Where(dbx.NewExp("deleted = null && moderated = true")).
		Where(dbx.In("id", interfaceIds...)).
		All(&res)
	if err != nil {
		return nil
	}
	schematicModels := MapResultsToSchematic(app, res, cacheService)
	sortedModels := make([]models.Schematic, 0)
	for id := range ids {
		for i := range schematicModels {
			if ids[id] == schematicModels[i].ID {
				sortedModels = append(sortedModels, schematicModels[i])
			}
		}
	}
	return sortedModels
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

func countSchematicView(app *pocketbase.PocketBase, schematic *core.Record, discordService *discord.Service) {
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
		viewRecord.Set("count", count+1)
		if err = app.Save(viewRecord); err != nil {
			app.Logger().Error(err.Error())
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
		Rating:               fmt.Sprintf("%.1f", rating),
		RatingCount:          ratingCount,
		SchematicFile:        fmt.Sprintf("/api/files/%s/%s", result.BaseFilesPath(), result.GetString("schematic_file")),
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
