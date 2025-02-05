package server

import (
	"bufio"
	"bytes"
	"createmod/internal/migrate"
	"createmod/internal/pages"
	"createmod/internal/router"
	"createmod/internal/search"
	_ "createmod/migrations"
	"errors"
	"fmt"
	"github.com/apokalyptik/phpass"
	"github.com/gosimple/slug"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/sunshineplan/imgconv"
	"github.com/sym01/htmlsanitizer"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Config struct {
	MysqlHost   string
	MysqlDB     string
	MysqlUser   string
	MysqlPass   string
	AutoMigrate bool
	CreateAdmin bool
}

type Server struct {
	conf Config
}

func New(conf Config) *Server {
	return &Server{conf: conf}
}

func (s *Server) Start() {
	app := pocketbase.New()
	var searchService *search.Service
	log.Println("Launching...")

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		Automigrate: s.conf.AutoMigrate,
	})

	app.OnBeforeBootstrap().Add(func(e *core.BootstrapEvent) error {
		log.Println("Bootstrapping...")
		return nil
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		log.Println("Running Before Serve Logic")

		go func() {
			// MIGRATIONS
			if s.conf.MysqlDB != "" {
				gormdb, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", s.conf.MysqlUser, s.conf.MysqlPass, s.conf.MysqlHost, s.conf.MysqlDB)))
				if err == nil {
					migrate.Run(app, gormdb)
				} else {
					app.Logger().Debug(
						"MIGRATION SKIPPED - No MySQL Connection",
					)
				}
			}

			// END MIGRATIONS

			// SEARCH
			log.Println("Starting Search Server")
			schematics, err := app.Dao().FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
			if err != nil {
				panic(err)
			}
			mappedSchematics := pages.MapResultsToSchematic(app, schematics)
			app.Logger().Debug("search service mapped schematics", "mapped schematic count", len(mappedSchematics))
			searchService.BuildIndex(mappedSchematics)

			// END SEARCH

			app.OnModelAfterCreate("schematics").Add(func(e *core.ModelEvent) error {
				// Rebuild the search index every time a schematic is created
				schematics, err := app.Dao().FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
				if err != nil {
					app.Logger().Warn(err.Error())
					return err
				}
				searchService.BuildIndex(pages.MapResultsToSchematic(app, schematics))
				return nil
			})

		}()

		app.OnRecordBeforeCreateRequest("schematics").Add(func(e *core.RecordCreateEvent) error {
			info := apis.RequestInfo(e.HttpContext)
			if info.AuthRecord == nil {
				return fmt.Errorf("user is not authenticated")
			}
			app.Logger().Debug("setting author", "id", info.AuthRecord.GetId(), "username", info.AuthRecord.GetString("username"))
			e.Record.Set("author", info.AuthRecord.GetId())

			if err := validateAndPopulateSchematic(app, e.Record, e.UploadedFiles); err != nil {
				return err
			}
			return nil
		})

		app.OnRecordBeforeCreateRequest("comments").Add(func(e *core.RecordCreateEvent) error {
			info := apis.RequestInfo(e.HttpContext)
			if info.AuthRecord == nil {
				return fmt.Errorf("user is not authenticated")
			}
			app.Logger().Debug("setting author", "id", info.AuthRecord.GetId(), "username", info.AuthRecord.GetString("username"))
			e.Record.Set("author", info.AuthRecord.GetId())

			if err := validateAndSaveComment(app, e.Record, info.AuthRecord); err != nil {
				return err
			}
			return nil
		})

		app.OnRecordBeforeCreateRequest("schematic_ratings").Add(func(e *core.RecordCreateEvent) error {
			info := apis.RequestInfo(e.HttpContext)
			if info.AuthRecord == nil {
				return fmt.Errorf("user is not authenticated")
			}
			schematicRatingsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_ratings")
			if err != nil {
				return err
			}
			results, err := app.Dao().FindRecordsByFilter(
				schematicRatingsCollection.Id,
				"schematic = {:schematic} && user = {:user}",
				"-created",
				10,
				0,
				dbx.Params{"schematic": e.Record.GetString("schematic"), "user": info.AuthRecord.GetId()})
			if err != nil {
				return err
			}
			if len(results) > 0 {
				for _, r := range results {
					// When a rating is changed we simply delete the old record
					err := app.Dao().Delete(r)
					if err != nil {
						return err
					}
				}
			}
			e.Record.Set("rated_at", time.Now())
			return nil
		})

		// PASSWORD BACKWARDS COMPATIBILITY
		app.OnRecordBeforeAuthWithPasswordRequest("users").Add(func(e *core.RecordAuthWithPasswordEvent) error {
			if e.Record != nil && e.Record.GetString("old_password") != "" {
				p := phpass.New(nil)
				if p.Check([]byte(e.Password), []byte(e.Record.GetString("old_password"))) {
					hashedPassword, err := bcrypt.GenerateFromPassword([]byte(e.Password), 12)
					if err != nil {
						app.Logger().Warn("old password failled to hash", "error", err.Error())
						return nil
					}
					e.Record.Set(schema.FieldNamePasswordHash, string(hashedPassword))
					e.Record.Set("old_password", "")
					if err = app.Dao().SaveRecord(e.Record); err != nil {
						app.Logger().Warn("old password invalid", "error", err.Error())
						return nil
					}
				}
			}
			return nil
		})
		// END PASSWORD BACKWARDS COMPATIBILITY

		searchService = search.New(nil, app.Logger())

		// ROUTES

		router.Register(app, e.Router, searchService)

		// END ROUTES

		log.Println("CreateMod.com Running!")
		return nil
	})

	log.Println("Initializing...")
	if err := app.Start(); err != nil {
		panic(err)
	}
}

func validateAndSaveComment(app *pocketbase.PocketBase, record *models.Record, authRecord *models.Record) error {
	replyToUser := ""
	if record.GetString("parent") != "" {
		// Validate parent is a comment for the same schematic
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
			dbx.Params{"id": record.GetString("schematic")})

		for _, result := range results {
			if result.GetString("id") == record.GetString("parent") {
				replyToUser = result.GetString("author")
			}
		}
		if replyToUser == "" {
			return errors.New("Tried to reply to an invalid comment")
		}
	}

	// Validate that schematic exists
	schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
	if err != nil {
		return err
	}
	results, err := app.Dao().FindRecordsByFilter(
		schematicsCollection.Id,
		"id = {:id}",
		"-created",
		1,
		0,
		dbx.Params{"id": record.GetString("schematic")})

	if len(results) != 1 {
		return errors.New("Tried to comment on an invalid schematic")
	}

	// Sanitize content
	content := record.GetString("content")
	if content == "" {
		return fmt.Errorf("comment can not be empty")
	}
	// Sanitize description
	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	description, err := sanitizer.SanitizeString(content)
	if err != nil {
		return err
	}
	record.Set("content", description)
	record.Set("published", time.Now().Format("2006-01-02 15:04:05.999Z07:00"))
	record.Set("type", "comment")
	record.Set("approved", true)

	message := &mailer.Message{}

	if replyToUser == "" {

		u, err := app.Dao().FindRecordById("users", results[0].GetString("author"))
		if err != nil {
			return err
		}

		message = &mailer.Message{
			From: mail.Address{
				Address: app.Settings().Meta.SenderAddress,
				Name:    app.Settings().Meta.SenderName,
			},
			To:      []mail.Address{{Address: u.Email()}},
			Subject: fmt.Sprintf("New comment on %s", results[0].GetString("title")),
			HTML:    fmt.Sprintf("<p>A new comment has been posted on your CreateMod.com schematic: <a href=\"https://www.createmod.com/schematics/%s\">https://www.createmod.com/schematics/%s</a><p>", results[0].GetString("name"), results[0].GetString("name")),
		}
	} else {
		u, err := app.Dao().FindRecordById("users", replyToUser)
		if err != nil {
			return err
		}

		message = &mailer.Message{
			From: mail.Address{
				Address: app.Settings().Meta.SenderAddress,
				Name:    app.Settings().Meta.SenderName,
			},
			To:      []mail.Address{{Address: u.Email()}},
			Subject: fmt.Sprintf("New reply on %s", results[0].GetString("title")),
			HTML:    fmt.Sprintf("<p>A new reply has been posted to your comment on CreateMod.com: <a href=\"https://www.createmod.com/schematics/%s\">https://www.createmod.com/schematics/%s</a><p>", results[0].GetString("name"), results[0].GetString("name")),
		}
	}

	return app.NewMailClient().Send(message)
}

func validateAndPopulateSchematic(app *pocketbase.PocketBase, record *models.Record, files map[string][]*filesystem.File) error {
	// Title and slug
	schematicSlug := slug.Make(record.GetString("title"))
	if schematicSlug == "" || !strings.ContainsFunc(schematicSlug, anyLetter) {
		return fmt.Errorf("title is invalid, please use alphanumeric characters")
	}
	// Make slug unique
	uSlug := uniqueSlug(app, schematicSlug)
	app.Logger().Debug("slug failed", "slug", schematicSlug, "unique", uSlug)
	if uSlug == "" {
		return fmt.Errorf("could not generate a unique slug")
	}
	record.Set("name", uSlug)

	// Validate description
	description := record.GetString("description")
	if description == "" {
		return fmt.Errorf("description can not be empty")
	}
	// Sanitize description
	sanitizer := htmlsanitizer.NewHTMLSanitizer()
	description, err := sanitizer.SanitizeString(description)
	if err != nil {
		return err
	}
	record.Set("description", description)

	// Check valid video url
	vidUrl := record.GetString("video")
	if vidUrl != "" {
		vidUrl = ToYoutubeEmbedUrl(vidUrl)
		record.Set("video", vidUrl)
	}

	// Check valid schematic file
	// TODO this can be improved by parsing the file
	if sf, ok := files["schematic_file"]; ok {
		if len(sf) == 0 || sf[0].Size <= 1 {
			return fmt.Errorf("schematic file invalid")
		}
	} else {
		return fmt.Errorf("schematic file missing or invalid")
	}

	// Check valid featured image
	if fi, ok := files["featured_image"]; ok {
		if len(fi) == 0 || fi[0].Size <= 1 {
			return fmt.Errorf("featured image invalid")
		}
	} else {
		return fmt.Errorf("featured image missing or invalid")
	}

	// convert to jpg in background
	go convertToJpg(app, record, files)

	// return nil if all is ok
	return nil
}

func convertToJpg(app *pocketbase.PocketBase, record *models.Record, files map[string][]*filesystem.File) {
	var galleryFilenames []string
	fs, err := app.NewFilesystem()
	if err != nil {
		return
	}

	for fieldKey := range files {
		for i, file := range files[fieldKey] {
			//Skip schematics
			if filepath.Ext(file.Name) == ".nbt" {
				continue
			}
			path := record.BaseFilesPath() + "/" + file.Name

			if err := fs.UploadFile(file, path); err != nil {
				return
			}

			r, err := fs.GetFile(path)
			if err != nil {
				return
			}

			decode, err := imgconv.Decode(r)
			if err != nil {
				return
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
				return
			}

			newFile, err := filesystem.NewFileFromBytes(jpgBuffer.Bytes(), filename)
			if err != nil {
				return
			}

			if err := fs.Delete(path); err != nil {
				return
			}

			path = record.BaseFilesPath() + "/" + filename
			if err := fs.UploadFile(newFile, path); err != nil {
				return
			}
			files[fieldKey][i].Name = filename

			if fieldKey == "featured_image" {
				record.Set("featured_image", filename)
			} else {
				galleryFilenames = append(galleryFilenames, filename)
			}
		}
	}
	record.Set("gallery", galleryFilenames)
	err = app.Dao().Save(record)
	if err != nil {
		return
	}
}

func ToYoutubeEmbedUrl(url string) string {
	r, err := regexp.Compile("(?:youtube\\.com\\/(?:[^\\/]+\\/.+\\/|(?:v|e(?:mbed)?)\\/|.*[?&]v=)|youtu\\.be\\/)([^\"&?\\/\\s]{11})")
	if err != nil {
		panic(err)
	}
	matches := r.FindAllStringSubmatch(url, 1)
	if len(matches) == 1 && len(matches[0]) == 2 {
		return fmt.Sprintf("https://www.youtube.com/embed/%s", matches[0][1])
	}
	return ""
}

func uniqueSlug(app *pocketbase.PocketBase, s string) string {
	records, err := app.Dao().FindRecordsByFilter("schematics", "name={:slug}", "-created", 1, 0, dbx.Params{"slug": s})
	if err != nil {
		return ""
	}
	if len(records) > 0 {
		return uniqueSlug(app, fmt.Sprintf("%s%s", s, randSeq(4)))
	}
	return s
}

func anyLetter(r rune) bool {
	return unicode.IsLetter(r)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randSeq(n int) string {
	letters := []rune("bcdfghjklmnpqrstvwxz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
