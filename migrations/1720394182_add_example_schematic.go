package migrations

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"time"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		envFile, err := godotenv.Read(".env")
		if err != nil {
			return err
		}

		if envFile["DUMMY_DATA"] == "true" {
			dao := daos.New(db)

			// Add category
			schematicCategoriesCollection, err := dao.FindCollectionByNameOrId("schematic_categories")
			if err != nil {
				return err
			}
			categoryRecord := models.NewRecord(schematicCategoriesCollection)
			categoryRecord.Set("key", "exammple")
			categoryRecord.Set("name", "Example")
			if err := dao.SaveRecord(categoryRecord); err != nil {
				return err
			}
			// Add tags
			schematicTagsCollection, err := dao.FindCollectionByNameOrId("schematic_tags")
			if err != nil {
				return err
			}
			tagRecord := models.NewRecord(schematicTagsCollection)
			tagRecord.Set("key", "build")
			tagRecord.Set("name", "Build")
			if err := dao.SaveRecord(tagRecord); err != nil {
				return err
			}
			// Add user
			userCollection, err := dao.FindCollectionByNameOrId("users")
			if err != nil {
				return err
			}
			userRecord := models.NewRecord(userCollection)

			userRecord.Set("old_id", "1")
			userRecord.Set("created", time.Now())
			userRecord.Set("username", "dummytestuser")
			userRecord.Set("email", "testuser@createmod.com")
			userRecord.Set("name", "Test User")
			userRecord.Set("status", fmt.Sprintf("%d", 1))
			userRecord.Set("tokenKey", uuid.Must(uuid.NewRandom()).String())

			if err := dao.SaveRecord(userRecord); err != nil {
				panic(err)
			}
			// Make schematic
			schematicsCollection, err := dao.FindCollectionByNameOrId("schematics")
			if err != nil {
				return err
			}
			record := models.NewRecord(schematicsCollection)
			record.RefreshId()

			app := pocketbase.New().App

			form := forms.NewRecordUpsert(app, record)
			record.Set("old_id", "1")
			record.Set("created", time.Now())
			record.Set("author", userRecord.Id)
			record.Set("comment_count", 0)
			record.Set("comment_status", "Open")
			record.Set("content", "This is a test schematic. I wrote a longer text here so you can better see how posts with longer content might look.")
			record.Set("content_filtered", "This is a test schematic. I wrote a longer text here so you can better see how posts with longer content might look.")
			record.Set("excerpt", "This is a test schematic. I wrote a longer text here so you can better see how posts with longer content might look.")
			record.Set("guid", uuid.NewString())
			record.Set("menu_order", 0)
			record.Set("mime_type", "")
			record.Set("modified", time.Now())
			record.Set("name", "Test Schematic")
			record.Set("password", "")
			record.Set("postdate", time.Now())
			record.Set("title", "Test Schematic")
			record.Set("updated", time.Now())

			schematicFile, err := filesystem.NewFileFromPath("./web/static/dummy/test.nbt")
			if err != nil {
				return err
			}
			err = form.AddFiles("schematic_file", schematicFile)
			if err != nil {
				return err
			}
			record.Set("schematic_file", schematicFile.Name)

			fileFromPath, err := filesystem.NewFileFromPath("./web/static/dummy/19201080.png")
			if err != nil {
				return err
			}
			err = form.AddFiles("featured_image", fileFromPath)
			if err != nil {
				return err
			}
			record.Set("featured_image", fileFromPath.Name)

			var galleryFilenames []string
			if err != nil {
				return err
			}
			err = form.AddFiles("gallery", fileFromPath)
			if err != nil {
				return err
			}
			galleryFilenames = append(galleryFilenames, fileFromPath.Name)
			record.Set("gallery", galleryFilenames)

			fs, err := app.NewFilesystem()
			if err != nil {
				return err
			}

			filesToUpload := form.FilesToUpload()
			for fieldKey := range filesToUpload {
				for _, file := range filesToUpload[fieldKey] {
					path := record.BaseFilesPath() + "/" + file.Name
					if err := fs.UploadFile(file, path); err != nil {
						panic(err)
					}
				}
			}
			return dao.SaveRecord(record)
		}
		return nil
	}, func(db dbx.Builder) error {
		// nothing to undo

		return nil
	})
}
