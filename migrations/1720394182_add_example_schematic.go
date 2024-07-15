package migrations

import (
	"fmt"
	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/joho/godotenv"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"math/rand"
	"time"
)

type schematicData struct {
	Word      string `faker:"word"`
	Sentence  string `faker:"sentence"`
	Paragraph string `faker:"paragraph"`
}

func init() {
	m.Register(func(db dbx.Builder) error {
		envFile, err := godotenv.Read(".env")
		if err != nil {
			return err
		}

		if envFile["DUMMY_DATA"] == "true" {
			categories := map[string]string{
				"player-transport":     "Player Transport",
				"builds":               "Builds",
				"farms":                "Farms",
				"other-contraptions":   "Other Contraptions",
				"minecart-contraption": "Minecart Contraption",
				"other-farms":          "Other Farms",
				"power-generation":     "Power Generation",
				"ore-processing":       "Ore Processing",
				"doors":                "Doors",
				"miner":                "Miner",
				"flying-machines":      "Flying Machines",
				"redstone":             "Redstone",
				"mob-farms":            "Mob Farms",
				"crop-farms":           "Crop Farms",
				"crane":                "Crane",
				"item-processor":       "Item Processor",
			}
			tags := map[string]string{
				"train":                  "Train",
				"wood":                   "Wood",
				"gantry-shaft":           "Gantry Shaft",
				"factory":                "Factory",
				"farm":                   "Farm",
				"elevator":               "Elevator",
				"house":                  "House",
				"base":                   "Base",
				"furnace-engine":         "Furnace Engine",
				"wheat":                  "Wheat",
				"gold":                   "Gold",
				"iron":                   "Iron",
				"kelp":                   "Kelp",
				"windmill":               "Windmill",
				"crane":                  "Crane",
				"multi-level":            "Multi Level",
				"customizable":           "Customizable",
				"above-and-beyond":       "Above And Beyond",
				"clock-tower":            "Clock Tower",
				"cutom":                  "Cutom",
				"storage-drawers":        "Storage Drawers",
				"modpack":                "Modpack",
				"blaze-farm":             "Blaze Farm",
				"quarry":                 "Quarry",
				"miner":                  "Miner",
				"warehouse":              "Warehouse",
				"bearing-flying-machine": "Bearing Flying Machine",
				"piston-flying-machine":  "Piston Flying Machine",
				"drill":                  "Drill",
				"gantry-flying-machine":  "Gantry Flying Machine",
				"vault-door":             "Vault Door",
				"villagers":              "Villagers",
				"cocoa-beans":            "Cocoa Beans",
				"nether-wart":            "Nether Wart",
				"mushrooms":              "Mushrooms",
				"apple":                  "Apple",
				"sugar-cane":             "Sugar Cane",
				"bamboo":                 "Bamboo",
				"carrot":                 "Carrot",
				"sea-pickle":             "Sea Pickle",
				"flower":                 "Flower",
				"cactus":                 "Cactus",
				"potato":                 "Potato",
				"pumpkins":               "Pumpkins",
				"sweet-berries":          "Sweet Berries",
				"zombie-farm":            "Zombie Farm",
				"spider-farm":            "Spider Farm",
				"skeleton-farm":          "Skeleton Farm",
				"castle-door":            "Castle Door",
				"build":                  "Build",
			}

			dao := daos.New(db)

			categoryRecords := make([]string, 0)
			tagRecords := make([]string, 0)

			// Add category
			schematicCategoriesCollection, err := dao.FindCollectionByNameOrId("schematic_categories")
			if err != nil {
				return err
			}
			for key, name := range categories {
				categoryRecord := models.NewRecord(schematicCategoriesCollection)
				categoryRecord.Set("key", key)
				categoryRecord.Set("name", name)
				if err := dao.SaveRecord(categoryRecord); err != nil {
					return err
				}
				categoryRecords = append(categoryRecords, categoryRecord.GetId())
			}
			// Add tags
			schematicTagsCollection, err := dao.FindCollectionByNameOrId("schematic_tags")
			if err != nil {
				return err
			}
			for key, name := range tags {
				tagRecord := models.NewRecord(schematicTagsCollection)
				tagRecord.Set("key", key)
				tagRecord.Set("name", name)
				if err := dao.SaveRecord(tagRecord); err != nil {
					return err
				}
				tagRecords = append(tagRecords, tagRecord.GetId())
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

			schematicFile, err := filesystem.NewFileFromPath("./web/static/dummy/test.nbt")
			fileFromPath, err := filesystem.NewFileFromPath("./web/static/dummy/19201080.png")

			// Make schematic
			for i := 0; i < 50; i++ {
				a := schematicData{}
				err = faker.FakeData(&a)
				if err != nil {
					return err
				}
				schematicsCollection, err := dao.FindCollectionByNameOrId("schematics")
				if err != nil {
					return err
				}
				record := models.NewRecord(schematicsCollection)
				record.RefreshId()

				app := pocketbase.New().App

				form := forms.NewRecordUpsert(app, record)
				record.Set("old_id", fmt.Sprintf("%d", i+1))
				record.Set("created", time.Now())
				record.Set("author", userRecord.Id)
				record.Set("comment_count", 0)
				record.Set("comment_status", "Open")
				record.Set("content", a.Paragraph)
				record.Set("content_filtered", a.Paragraph)
				record.Set("excerpt", a.Paragraph)
				record.Set("guid", uuid.NewString())
				record.Set("menu_order", 0)
				record.Set("mime_type", "")
				record.Set("modified", time.Now())
				record.Set("name", slug.Make(a.Sentence))
				record.Set("password", "")
				record.Set("postdate", time.Now())
				record.Set("title", a.Sentence)
				record.Set("updated", time.Now())
				record.Set("categories", []string{categoryRecords[rand.Intn(len(categoryRecords))]})
				record.Set("tags", []string{tagRecords[rand.Intn(len(tagRecords))]})

				if err != nil {
					return err
				}
				err = form.AddFiles("schematic_file", schematicFile)
				if err != nil {
					return err
				}
				record.Set("schematic_file", schematicFile.Name)

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
				err = dao.SaveRecord(record)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}, func(db dbx.Builder) error {
		// nothing to undo

		return nil
	})
}
