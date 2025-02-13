package migrate

import (
	"bufio"
	"bytes"
	"createmod/model"
	"createmod/query"
	"errors"
	"fmt"
	"github.com/elliotchance/phpserialize"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/sunshineplan/imgconv"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func migrateSchematics(app *pocketbase.PocketBase, gormdb *gorm.DB, userOldId map[int64]string) map[int64]string {
	log.Println("Migrating schematics.")
	q := query.Use(gormdb)
	postRes, postErr := q.QeyKryWEpost.
		Where(q.QeyKryWEpost.PostParent.Eq(0)).
		Where(q.QeyKryWEpost.PostStatus.Eq("publish")).
		Where(q.QeyKryWEpost.PostType.Eq("schematics")).
		Find()
	oldSchematicIDs := make(map[int64]string, len(postRes))
	if postErr != nil {
		panic(postErr)
	}
	schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
	if err != nil {
		panic(err)
	}
	schematicCategoriesCollection, err := app.FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		panic(err)
	}
	schematicTagsCollection, err := app.FindCollectionByNameOrId("schematic_tags")
	if err != nil {
		panic(err)
	}
	minecraftVersionsCollection, err := app.FindCollectionByNameOrId("minecraft_versions")
	if err != nil {
		panic(err)
	}
	createmodVersionsCollection, err := app.FindCollectionByNameOrId("createmod_versions")
	if err != nil {
		panic(err)
	}
	for _, s := range postRes {
		record := core.NewRecord(schematicsCollection)

		author := fmt.Sprintf("%d", s.PostAuthor)

		if foundAuthor, ok := userOldId[s.PostAuthor]; ok {
			author = foundAuthor
		}

		postMetaRes, postMetaErr := q.QeyKryWEpostmetum.
			Where(q.QeyKryWEpostmetum.PostID.Eq(s.ID)).
			Find()
		if postMetaErr != nil {
			panic(postMetaErr)
		}

		filter, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"old_id = {:old_id}",
			"-created",
			1,
			0,
			dbx.Params{"old_id": s.ID})
		if !errors.Is(err, gorm.ErrRecordNotFound) && len(filter) != 0 {
			app.Logger().Debug(
				fmt.Sprintf("Schematic found or error: %v", err),
				"filter-len", len(filter),
			)
			if len(filter) == 1 {
				oldSchematicIDs[s.ID] = filter[0].Id
			}
			continue
		}

		form := forms.NewRecordUpsert(app, record)

		// keys from postmeta
		// schematicf_description text
		// has_dependencies 0 or 1
		// schematicf_file post id int, guid has download link
		// schematicf_tags wp encoded array of tag ids
		// schematicf_video text url of video
		// schematicf_gallery array of post ids
		// schematicf_featured_image post id, also thumbnail
		// schematicf_game_version text
		// schematicf_mod_version text
		// schematicf_category category id
		// schematicf_title text
		for _, m := range postMetaRes {
			switch m.MetaKey {
			case "schematicf_description":
				record.Set("description", m.MetaValue)
			case "has_dependencies":
				record.Set("has_dependencies", "0" != m.MetaValue)
			case "schematicf_file":
				processSchematicFile(m, q, record, form)
			case "schematicf_tags":
				processSchematicTags(app, m, q, record, schematicTagsCollection)
			case "schematicf_video":
				record.Set("video", m.MetaValue)
			case "schematicf_gallery":
				processSchematicGallery(m, q, record, form)
			case "schematicf_featured_image":
				processSchematicFeaturedImage(m, q, record, form)
			case "schematicf_game_version":
				processMinecraftVersion(app, m, record, minecraftVersionsCollection)
			case "schematicf_mod_version":
				processCreatemodVersion(app, m, record, createmodVersionsCollection)
			case "schematicf_category":
				processCategories(app, m, q, record, schematicCategoriesCollection)
			case "schematicf_title":
				record.Set("schematic_title", m.MetaValue)
			case "dependencies":
				record.Set("dependencies", m.MetaValue)
			}
		}

		record.Set("old_id", s.ID)
		record.Set("created", s.PostDateGmt)
		record.Set("author", author)
		record.Set("comment_count", s.CommentCount)
		record.Set("comment_status", s.CommentStatus)
		record.Set("content", s.PostContent)
		record.Set("content_filtered", s.PostContentFiltered)
		record.Set("excerpt", s.PostExcerpt)
		record.Set("guid", s.GUID)
		record.Set("menu_order", s.MenuOrder)
		record.Set("mime_type", s.PostMimeType)
		record.Set("modified", s.PostModified)
		record.Set("name", s.PostName)
		record.Set("password", s.PostPassword)
		record.Set("ping_status", s.PingStatus)
		record.Set("pinged", s.Pinged)
		record.Set("postdate", s.PostDateGmt)
		record.Set("status", s.PingStatus)
		record.Set("title", s.PostTitle)
		record.Set("to_ping", s.ToPing)
		record.Set("type", s.PostType)
		record.Set("updated", s.PostModifiedGmt)
		record.Set("parent", s.PostParent)

		filesToUpload := make(map[string][]*filesystem.File, 0)

		err = app.Save(record)
		if err != nil {
			log.Printf("ERROR for %s - %d: %v\n", s.PostName, s.ID, err)
			continue
		}
		record, err = convertToJpg(app, record, filesToUpload)

		if err != nil {
			app.Logger().Error(
				fmt.Sprintf("Could not convert to jpg: %v", err),
			)
			panic("Fatal error, stop migration")
		}

		err = app.Save(record)
		if err != nil {
			app.Logger().Error(
				fmt.Sprintf("Could not save: %v", err),
			)
			log.Printf("ERROR for %s - %d: %v\n", s.PostName, s.ID, err)
			continue
		}

		oldSchematicIDs[s.ID] = record.Id
	}
	log.Printf("%d schematics processed.\n", len(oldSchematicIDs))
	return oldSchematicIDs
}

func convertToJpg(app *pocketbase.PocketBase, record *core.Record, files map[string][]*filesystem.File) (*core.Record, error) {
	var galleryFilenames []string
	fs, err := app.NewFilesystem()
	if err != nil {
		return record, err
	}

	for fieldKey := range files {
		for i, file := range files[fieldKey] {
			path := record.BaseFilesPath() + "/" + file.Name

			if err := fs.UploadFile(file, path); err != nil {
				return record, err
			}

			if fieldKey == "featured_image" || fieldKey == "gallery" {
				r, err := fs.GetFile(path)
				if err != nil {
					return record, err
				}

				decode, err := imgconv.Decode(r)
				if err != nil {
					return record, err
				}

				var jpgBuffer bytes.Buffer
				err = imgconv.Write(bufio.NewWriter(&jpgBuffer), decode, &imgconv.FormatOption{
					Format: imgconv.JPEG,
					EncodeOption: []imgconv.EncodeOption{
						imgconv.Quality(80),
					},
				})

				filename := strings.TrimSuffix(file.Name, filepath.Ext(file.Name)) + ".jpg"
				if err != nil {
					return record, err
				}

				newFile, err := filesystem.NewFileFromBytes(jpgBuffer.Bytes(), filename)
				if err != nil {
					return record, err
				}

				err = r.Close()
				if err != nil {
					return record, err
				}

				if err := fs.Delete(path); err != nil {
					return record, err
				}

				path = record.BaseFilesPath() + "/" + filename
				if err := fs.UploadFile(newFile, path); err != nil {
					return record, err
				}
				files[fieldKey][i].Name = filename

				if fieldKey == "featured_image" {
					record.Set("featured_image", filename)
				} else {
					galleryFilenames = append(galleryFilenames, filename)
				}
			}
		}
	}
	record.Set("gallery", galleryFilenames)

	return record, nil
}

func processCreatemodVersion(app *pocketbase.PocketBase, m *model.QeyKryWEpostmetum, record *core.Record, collection *core.Collection) {
	cmRecord := core.NewRecord(collection)
	cmRes, err := app.FindRecordsByFilter(
		collection.Id,
		"version = {:version}",
		"-created",
		1,
		0,
		dbx.Params{"version": m.MetaValue})
	if err != nil || len(cmRes) == 0 {
		cmRecord.Set("version", m.MetaValue)
		if err := app.Save(cmRecord); err != nil {
			panic(err)
		}
	} else {
		cmRecord = cmRes[0]
	}

	record.Set("createmod_version", []string{cmRecord.Id})
}

func processMinecraftVersion(app *pocketbase.PocketBase, m *model.QeyKryWEpostmetum, record *core.Record, collection *core.Collection) {
	mcvRecord := core.NewRecord(collection)
	mcvRes, err := app.FindRecordsByFilter(
		collection.Id,
		"version = {:version}",
		"-created",
		1,
		0,
		dbx.Params{"version": m.MetaValue})
	if err != nil || len(mcvRes) == 0 {
		mcvRecord.Set("version", m.MetaValue)
		if err := app.Save(mcvRecord); err != nil {
			panic(err)
		}
	} else {
		mcvRecord = mcvRes[0]
	}

	record.Set("minecraft_version", []string{mcvRecord.Id})
}

func processSchematicFile(m *model.QeyKryWEpostmetum, q *query.Query, record *core.Record, upser *forms.RecordUpsert) {
	fileID, err := strconv.Atoi(m.MetaValue)
	if err != nil {
		panic(err)
	}
	postFileRes, postFileErr := q.QeyKryWEpost.
		Where(q.QeyKryWEpost.ID.Eq(int64(fileID))).
		First()
	if postFileErr != nil {
		panic(postFileErr)
	}

	filename := fmt.Sprintf("migration_data/%s", strings.TrimPrefix(postFileRes.GUID, "https://createmod.com/"))
	if filename == "" {
		panic("empty guid")
	}
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(filename), 0700)
		if err != nil {
			panic(err)
		}
		out, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer out.Close()

		resp, err := http.Get(postFileRes.GUID)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			panic(err)
		}
	}
	fileFromPath, err := filesystem.NewFileFromPath(filename)
	if err != nil {
		log.Printf("ERROR for %s: %v\n", filename, err)
		return
	}
	record.Set("schematic_file", fileFromPath.Name)
}

func processSchematicTags(app *pocketbase.PocketBase, m *model.QeyKryWEpostmetum, q *query.Query, record *core.Record, schematicTagsCollection *core.Collection) {
	var tagRecords []string
	var tags []interface{}
	var tagIDs []int
	if m.MetaValue == "" {
		return
	}
	err := phpserialize.Unmarshal([]byte(m.MetaValue), &tags)
	if err != nil {
		panic(err)
	}
	for ti := range tags {
		tagID, err := strconv.Atoi(fmt.Sprintf("%v", tags[ti]))
		if err != nil {
			panic(err)
		}
		tagIDs = append(tagIDs, tagID)
	}
	for _, tagID := range tagIDs {
		termRes, termErr := q.QeyKryWEterm.
			Where(q.QeyKryWEterm.TermID.Eq(int64(tagID))).
			First()
		if termErr != nil {
			// skip
			continue
		}
		tagRecord := core.NewRecord(schematicTagsCollection)
		tagRes, err := app.FindRecordsByFilter(
			schematicTagsCollection.Id,
			"key = {:key}",
			"-created",
			1,
			0,
			dbx.Params{"key": termRes.Slug})
		if err != nil || len(tagRes) == 0 {
			// Create the category
			tagRecord.Set("key", termRes.Slug)
			tagRecord.Set("name", termRes.Name)
			if err := app.Save(tagRecord); err != nil {
				log.Printf("ERROR for %s - %s: %v\n", termRes.Slug, termRes.Name, err)
				continue
			}
		} else {
			tagRecord = tagRes[0]
		}
		tagRecords = append(tagRecords, tagRecord.Id)
	}

	if len(tagRecords) > 0 {
		record.Set("tags", tagRecords)
	}
}

func processCategories(app *pocketbase.PocketBase, m *model.QeyKryWEpostmetum, q *query.Query, record *core.Record, schematicCategoriesCollection *core.Collection) {
	postID, err := strconv.Atoi(m.MetaValue)
	if err != nil {
		panic(err)
	}
	termRes, termErr := q.QeyKryWEterm.
		Where(q.QeyKryWEterm.TermID.Eq(int64(postID))).
		First()
	if termErr != nil {
		panic(termErr)
	}
	categoryRecord := core.NewRecord(schematicCategoriesCollection)
	categoryRes, err := app.FindRecordsByFilter(
		schematicCategoriesCollection.Id,
		"key = {:key}",
		"-created",
		1,
		0,
		dbx.Params{"key": termRes.Slug})
	if err != nil || len(categoryRes) == 0 {
		// Create the category
		categoryRecord.Set("key", termRes.Slug)
		categoryRecord.Set("name", termRes.Name)
		if err := app.Save(categoryRecord); err != nil {
			panic(err)
		}
	} else {
		categoryRecord = categoryRes[0]
	}

	record.Set("categories", []string{categoryRecord.Id})
}

func processSchematicGallery(m *model.QeyKryWEpostmetum, q *query.Query, record *core.Record, form *forms.RecordUpsert) {
	var galleryFilenames []string
	var postIDs []interface{}
	var postInts []int
	if m.MetaValue == "" {
		return
	}
	err := phpserialize.Unmarshal([]byte(m.MetaValue), &postIDs)
	if err != nil {
		panic(err)
	}
	for pi := range postIDs {
		tagID, err := strconv.Atoi(fmt.Sprintf("%v", postIDs[pi]))
		if err != nil {
			// skip
			continue
		}
		postInts = append(postInts, tagID)
	}
	for _, fileID := range postInts {
		postFileRes, postFileErr := q.QeyKryWEpost.
			Where(q.QeyKryWEpost.ID.Eq(int64(fileID))).
			First()
		if postFileErr != nil {
			panic(postFileErr)
		}

		filename := fmt.Sprintf("migration_data/%s", strings.TrimPrefix(postFileRes.GUID, "https://createmod.com/"))
		if filename == "" {
			panic("empty guid")
		}
		if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(filepath.Dir(filename), 0700)
			if err != nil {
				panic(err)
			}
			out, err := os.Create(filename)
			if err != nil {
				panic(err)
			}
			defer out.Close()

			resp, err := http.Get(postFileRes.GUID)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				panic(err)
			}
		}
		fileFromPath, err := filesystem.NewFileFromPath(filename)
		if err != nil {
			panic(err)
		}
		galleryFilenames = append(galleryFilenames, fileFromPath.Name)
	}
	record.Set("gallery", galleryFilenames)
}

func processSchematicFeaturedImage(m *model.QeyKryWEpostmetum, q *query.Query, record *core.Record, form *forms.RecordUpsert) {
	fileID, err := strconv.Atoi(m.MetaValue)
	if err != nil {
		panic(err)
	}
	postFileRes, postFileErr := q.QeyKryWEpost.
		Where(q.QeyKryWEpost.ID.Eq(int64(fileID))).
		First()
	if postFileErr != nil {
		panic(postFileErr)
	}

	filename := fmt.Sprintf("migration_data/%s", strings.TrimPrefix(postFileRes.GUID, "https://createmod.com/"))
	if filename == "" {
		panic("empty guid")
	}
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(filename), 0700)
		if err != nil {
			panic(err)
		}
		out, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer out.Close()

		resp, err := http.Get(postFileRes.GUID)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			panic(err)
		}
	}
	fileFromPath, err := filesystem.NewFileFromPath(filename)
	if err != nil {
		panic(err)
	}
	record.Set("featured_image", fileFromPath.Name)
}
