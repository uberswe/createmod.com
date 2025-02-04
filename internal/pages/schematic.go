package pages

import (
	"createmod/internal/models"
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
	Schematic models.Schematic
	Comments  []models.Comment
}

func SchematicHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
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

		go countSchematicView(app, results[0])
		err = c.Render(http.StatusOK, schematicTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}

func findSchematicComments(app *pocketbase.PocketBase, id string) []models.Comment {
	schematicsCollection, err := app.Dao().FindCollectionByNameOrId("comments")
	if err != nil {
		return nil
	}
	// Limit comments to 1000 for now, will add pagination later
	results, err := app.Dao().FindRecordsByFilter(
		schematicsCollection.Id,
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
		if cs[i].ParentID == "" {
			return true
		}
		return false
	})
	for _, c := range cs {
		if c.ParentID != "" {
			for i := range comments {
				if c.ParentID == comments[i].ID {
					comments = slices.Insert(comments, i+1, mapResultToComment(app, c))
					comments[i+1].Indent = 1
					break
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

	sanitizedHTML, err := htmlsanitizer.SanitizeString(c.Content)
	if err != nil {
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

	sanitizedHTML, err := htmlsanitizer.SanitizeString(result.GetString("content"))
	if err != nil {
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
