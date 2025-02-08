package pages

import (
	"createmod/internal/models"
	"createmod/internal/search"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/mergestat/timediff"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

const schematicTemplate = "schematic.html"

type SchematicData struct {
	DefaultData
	Schematic     models.Schematic
	Comments      []models.Comment
	AuthorHasMore bool
	FromAuthor    []models.Schematic
	Similar       []models.Schematic
}

func SchematicHandler(app *pocketbase.PocketBase, searchService *search.Service) func(c echo.Context) error {
	return func(c echo.Context) error {
		schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.Dao().FindRecordsByFilter(
			schematicsCollection.Id,
			"name = {:name}",
			"-created",
			1,
			0,
			dbx.Params{"name": c.PathParam("name")})

		if len(results) != 1 {
			return c.Render(http.StatusNotFound, fourOhFourTemplate, nil)
		}

		d := SchematicData{
			Schematic: mapResultToSchematic(app, results[0]),
		}
		d.Title = d.Schematic.Title
		d.SubCategory = "Schematic"
		d.Categories = allCategories(app)
		d.Comments = findSchematicComments(app, d.Schematic.ID)
		d.FromAuthor = findAuthorSchematics(app, d.Schematic.ID, d.Schematic.Author.ID)
		d.Similar = findSimilarSchematics(app, d.Schematic, d.FromAuthor, searchService)
		d.AuthorHasMore = len(d.FromAuthor) > 0

		go countSchematicView(app, results[0])
		err = c.Render(http.StatusOK, schematicTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}

func findAuthorSchematics(app *pocketbase.PocketBase, id string, authorID string) []models.Schematic {
	schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
	if err != nil {
		return nil
	}
	results, err := app.Dao().FindRecordsByFilter(
		schematicsCollection.Id,
		"id != {:id} && author = {:authorID}",
		"@random",
		5,
		0,
		dbx.Params{"id": id, "authorID": authorID})
	return MapResultsToSchematic(app, results)
}

func findSimilarSchematics(app *pocketbase.PocketBase, schematic models.Schematic, author []models.Schematic, searchService *search.Service) []models.Schematic {
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
	ids := searchService.Search(fmt.Sprintf("%s%s", schematic.Title, keywordString), 1, -1, "all", "all")
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

	var schematics []models.DatabaseSchematic
	err := app.Dao().DB().
		Select("schematics.*").
		From("schematics").
		Where(dbx.In("id", interfaceIds...)).
		All(&schematics)

	if err != nil {
		return nil
	}
	schematicModels := models.DatabaseSchematicsToSchematics(schematics)
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
	commentsCollection, err := app.Dao().FindCollectionByNameOrId("comments")
	if err != nil {
		return nil
	}
	// Limit comments to 1000 for now, will add pagination later
	results, err := app.Dao().FindRecordsByFilter(
		commentsCollection.Id,
		"schematic = {:id} && approved = 1",
		"-created",
		1000,
		0,
		dbx.Params{"id": id})

	var comments []models.DatabaseComment

	for _, result := range results {
		comments = append(comments, models.DatabaseComment{
			ID:        result.GetId(),
			Created:   result.GetTime("created"),
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
		if cs[i].ParentID != "" {
			return false
		} else if cs[j].ParentID != "" {
			return true
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

	userRecord, err := app.Dao().FindRecordById("users", c.Author)
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
	fmt.Println(c.Created)
	comment.Created = timediff.TimeDiff(t)
	comment.Published = t.Format(time.DateTime)

	return comment
}

func countSchematicView(app *pocketbase.PocketBase, schematic *pbmodels.Record) {
	schematicViewsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_views")
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
		viewsRes, err := app.Dao().FindRecordsByFilter(
			schematicViewsCollection.Id,
			"schematic = {:schematic} && type = {:type} && period = {:period}",
			"-created",
			1,
			0,
			dbx.Params{
				"schematic": schematic.GetId(),
				"type":      t,
				"period":    p,
			})

		if err != nil || len(viewsRes) == 0 {
			if err != nil {
				app.Logger().Error(err.Error())
			}
			record := pbmodels.NewRecord(schematicViewsCollection)
			record.Set("schematic", schematic.GetId())
			record.Set("count", 1)
			record.Set("type", t)
			record.Set("period", p)

			if err = app.Dao().SaveRecord(record); err != nil {
				app.Logger().Error(err.Error())
				return
			}
			continue
		}

		viewRecord := viewsRes[0]
		viewRecord.Set("count", viewRecord.GetInt("count")+1)
		if err = app.Dao().SaveRecord(viewRecord); err != nil {
			app.Logger().Error(err.Error())
		}
	}
}

func MapResultsToSchematic(app *pocketbase.PocketBase, results []*pbmodels.Record) (schematics []models.Schematic) {
	for i := range results {
		schematics = append(schematics, mapResultToSchematic(app, results[i]))
	}
	return schematics
}

func mapResultToSchematic(app *pocketbase.PocketBase, result *pbmodels.Record) (schematic models.Schematic) {
	views := 0
	records, err := app.Dao().FindRecordsByFilter(
		"schematic_views",
		"period = 'total' && schematic = {:schematic}",
		"-updated",
		1,
		0,
		dbx.Params{"schematic": result.GetId()},
	)

	if err == nil && len(records) > 0 {
		views = records[0].GetInt("count")
	}

	rating := float64(0)
	totalRating := float64(0)

	ratings, err := app.Dao().FindRecordsByFilter(
		"schematic_ratings",
		"schematic = {:schematic}",
		"-updated",
		1000,
		0,
		dbx.Params{"schematic": result.GetId()},
	)
	if err == nil {
		for i := range ratings {
			totalRating += ratings[i].GetFloat("rating")
		}
		if len(ratings) > 0 {
			rating = totalRating / float64(len(ratings))
		}
	}
	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	sanitizedHTML, err := sanitizer.SanitizeString(result.GetString("content"))
	if err != nil {
		app.Logger().Debug("Failed to sanitize", "string", result.GetString("content"), "error", err)
		// Fallback legacy sanitizer
		sanitizedHTML = strings.ReplaceAll(template.HTMLEscapeString(result.GetString("content")), "\n", "<br/>")
	}

	s := models.Schematic{
		ID:               result.GetId(),
		Created:          result.Created.Time(),
		CreatedFormatted: result.Created.Time().Format("2006-01-02"),
		Author:           findUserFromID(app, result.GetString("author")),
		CommentCount:     result.GetInt("comment_count"),
		CommentStatus:    result.GetBool("comment_status"),
		Content:          result.GetString("content"),
		HTMLContent:      template.HTML(sanitizedHTML),
		Excerpt:          result.GetString("excerpt"),
		FeaturedImage:    result.GetString("featured_image"),
		Gallery:          result.GetStringSlice("gallery"),
		HasGallery:       len(result.GetStringSlice("gallery")) > 0,
		Title:            result.GetString("title"),
		Name:             result.GetString("name"),
		Video:            result.GetString("video"),
		HasDependencies:  result.GetBool("has_dependencies"),
		Dependencies:     result.GetString("dependencies"),
		HTMLDependencies: template.HTML(strings.ReplaceAll(template.HTMLEscapeString(result.GetString("dependencies")), "\n", "<br/>")),
		Categories:       findCategoriesFromIDs(app, result.GetStringSlice("categories")),
		Tags:             findTagsFromIDs(app, result.GetStringSlice("tags")),
		CreatemodVersion: findCreateModVersionFromID(app, result.GetString("createmod_version")),
		MinecraftVersion: findMinecraftVersionFromID(app, result.GetString("minecraft_version")),
		Views:            views,
		Rating:           fmt.Sprintf("%.1f", rating),
		SchematicFile:    fmt.Sprintf("/api/files/%s/%s", result.BaseFilesPath(), result.GetString("schematic_file")),
	}
	s.HasTags = len(s.Tags) > 0
	s.HasRating = s.Rating != ""
	return s
}

func findMinecraftVersionFromID(app *pocketbase.PocketBase, id string) string {
	record, err := app.Dao().FindRecordById("minecraft_versions", id)
	if err != nil {
		return ""
	}
	return record.GetString("version")
}

func findCreateModVersionFromID(app *pocketbase.PocketBase, id string) string {
	record, err := app.Dao().FindRecordById("createmod_versions", id)
	if err != nil {
		return ""
	}
	return record.GetString("version")
}

func findTagsFromIDs(app *pocketbase.PocketBase, s []string) []models.SchematicTag {
	tagsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_tags")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByIds(tagsCollection.Id, s)
	if err != nil {
		return nil
	}
	return mapResultToTags(records)
}

func mapResultToTags(records []*pbmodels.Record) (tags []models.SchematicTag) {
	for i := range records {
		tags = append(tags, models.SchematicTag{
			ID:   records[i].GetId(),
			Key:  records[i].GetString("key"),
			Name: records[i].GetString("name"),
		})
	}
	return tags
}

func findCategoriesFromIDs(app *pocketbase.PocketBase, s []string) []models.SchematicCategory {
	categoriesCollection, err := app.Dao().FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByIds(categoriesCollection.Id, s)
	if err != nil {
		return nil
	}
	return mapResultToCategories(records)
}

func mapResultToCategories(records []*pbmodels.Record) (categories []models.SchematicCategory) {
	for i := range records {
		categories = append(categories, models.SchematicCategory{
			ID:   records[i].GetId(),
			Key:  records[i].GetString("key"),
			Name: records[i].GetString("name"),
		})
	}
	return categories
}
