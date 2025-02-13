package server

import (
	"bufio"
	"bytes"
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/migrate"
	"createmod/internal/pages"
	"createmod/internal/router"
	"createmod/internal/search"
	"createmod/internal/sitemap"
	_ "createmod/migrations"
	"errors"
	"fmt"
	"github.com/apokalyptik/phpass"
	"github.com/gosimple/slug"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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
	"net/http"
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
	conf           Config
	app            *pocketbase.PocketBase
	searchService  *search.Service
	sitemapService *sitemap.Service
	cacheService   *cache.Service
}

func New(conf Config) *Server {
	app := pocketbase.New()
	sitemapService := sitemap.New()
	return &Server{
		conf:           conf,
		app:            app,
		sitemapService: sitemapService,
		cacheService:   cache.New(),
	}
}

func (s *Server) Start() {
	log.Println("Launching...")

	migratecmd.MustRegister(s.app, s.app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		Automigrate: s.conf.AutoMigrate,
	})

	s.app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		log.Println("Bootstrapping...")
		return e.Next()
	})

	s.app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		log.Println("Running Before Serve Logic")

		s.searchService = search.New(nil, s.app)
		go func() {
			// MIGRATIONS
			if s.conf.MysqlDB != "" {
				gormdb, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", s.conf.MysqlUser, s.conf.MysqlPass, s.conf.MysqlHost, s.conf.MysqlDB)))
				if err == nil {
					migrate.Run(s.app, gormdb)
				} else {
					s.app.Logger().Debug(
						"MIGRATION SKIPPED - No MySQL Connection",
					)
				}
			}

			// END MIGRATIONS

			// SEARCH
			log.Println("Starting Search Server")
			schematics, err := s.app.FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
			if err != nil {
				s.app.Logger().Error(err.Error())
			}
			mappedSchematics := pages.MapResultsToSchematic(s.app, schematics, s.cacheService)
			s.app.Logger().Debug("search service mapped schematics", "mapped schematic count", len(mappedSchematics))
			s.searchService.BuildIndex(mappedSchematics)

			// END SEARCH

			s.app.OnModelCreate("schematics").BindFunc(func(e *core.ModelEvent) error {
				// Rebuild the search index every time a schematic is created
				err := e.Next()
				if err != nil {
					return err
				}
				go func() {
					schematics, err := s.app.FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
					if err != nil {
						s.app.Logger().Warn(err.Error())
					}
					s.searchService.BuildIndex(pages.MapResultsToSchematic(s.app, schematics, s.cacheService))
					s.sitemapService.Generate(s.app)
				}()
				return e.Next()
			})

			s.app.OnModelCreate("users").BindFunc(func(e *core.ModelEvent) error {
				// Rebuild the sitemap every time a user is created
				err := e.Next()
				if err != nil {
					return err
				}
				go s.sitemapService.Generate(s.app)
				return e.Next()
			})

			s.sitemapService.Generate(s.app)

		}()

		s.app.OnRecordCreateRequest("schematics").BindFunc(func(e *core.RecordRequestEvent) error {
			if e.Auth == nil {
				return fmt.Errorf("user is not authenticated")
			}
			s.app.Logger().Debug("setting author", "id", e.Auth.Id, "username", e.Auth.GetString("username"))
			e.Record.Set("author", e.Auth.Id)

			if err := validateAndPopulateSchematic(s.app, e.Record, e); err != nil {
				return err
			}
			return e.Next()
		})

		s.app.OnRecordCreateRequest("comments").BindFunc(func(e *core.RecordRequestEvent) error {
			if e.Auth == nil {
				return fmt.Errorf("user is not authenticated")
			}
			s.app.Logger().Debug("setting author", "id", e.Auth.Id, "username", e.Auth.GetString("username"))
			e.Record.Set("author", e.Auth.Id)

			if err := validateAndSaveComment(s.app, e.Record, e.Auth); err != nil {
				return err
			}
			return e.Next()
		})

		s.app.OnRecordCreateRequest("schematic_ratings").BindFunc(func(e *core.RecordRequestEvent) error {
			if e.Auth == nil {
				return fmt.Errorf("user is not authenticated")
			}
			schematicRatingsCollection, err := s.app.FindCollectionByNameOrId("schematic_ratings")
			if err != nil {
				return err
			}
			results, _ := s.app.FindRecordsByFilter(
				schematicRatingsCollection.Id,
				"schematic = {:schematic} && user = {:user}",
				"-created",
				10,
				0,
				dbx.Params{"schematic": e.Record.GetString("schematic"), "user": e.Auth.Id})
			if len(results) > 0 {
				for _, r := range results {
					// When a rating is changed we simply delete the old record
					err := s.app.Delete(r)
					if err != nil {
						return err
					}
				}
			}
			e.Record.Set("user", e.Auth.Id)
			e.Record.Set("rated_at", time.Now())
			return e.Next()
		})

		s.app.OnRecordAfterCreateSuccess("contact_form_submissions").BindFunc(func(e *core.RecordEvent) error {
			message := &mailer.Message{
				From: mail.Address{
					Address: s.app.Settings().Meta.SenderAddress,
					Name:    s.app.Settings().Meta.SenderName,
				},
				To:      []mail.Address{{Address: s.app.Settings().Meta.SenderAddress}},
				Subject: fmt.Sprintf("New CreateMod.com Contact Form Submission"),
				HTML:    fmt.Sprintf("<p>Email: " + e.Record.GetString("email") + "</p><p>Content: " + e.Record.GetString("content") + "</p>"),
			}

			return s.app.NewMailClient().Send(message)
		})

		// COOKIES
		s.app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
			s.app.Logger().Info("onRecordAuthRequest", "record", e.Record, "setCookie", auth.CookieName, "exp", e.Record.Collection().AuthToken.Duration)
			e.SetCookie(&http.Cookie{
				Name:     auth.CookieName,
				Value:    e.Token,
				Expires:  time.Now().Add(time.Second * time.Duration(e.Record.Collection().AuthToken.Duration)),
				Path:     "/",
				SameSite: http.SameSiteStrictMode,
			})
			return e.Next()
		})
		// END COOKIES

		// PASSWORD BACKWARDS COMPATIBILITY
		s.app.OnRecordAuthWithPasswordRequest("users").BindFunc(func(e *core.RecordAuthWithPasswordRequestEvent) error {
			if e.Record != nil && e.Record.GetString("old_password") != "" {
				p := phpass.New(nil)
				if p.Check([]byte(e.Password), []byte(e.Record.GetString("old_password"))) {
					hashedPassword, err := bcrypt.GenerateFromPassword([]byte(e.Password), 12)
					if err != nil {
						s.app.Logger().Warn("old password failled to hash", "error", err.Error())
						return e.Next()
					}
					e.Record.SetPassword(string(hashedPassword))
					e.Record.Set("old_password", "")
					if err = s.app.Save(e.Record); err != nil {
						s.app.Logger().Warn("old password invalid", "error", err.Error())
						return e.Next()
					}
				}
			}
			return e.Next()
		})
		// END PASSWORD BACKWARDS COMPATIBILITY

		// ROUTES

		router.Register(s.app, e.Router, s.searchService, s.cacheService)

		// END ROUTES

		log.Println("CreateMod.com Running!")
		return e.Next()
	})

	log.Println("Initializing...")
	if err := s.app.Start(); err != nil {
		panic(err)
	}
}

func validateAndSaveComment(app *pocketbase.PocketBase, record *core.Record, authrecord *core.Record) error {
	replyToUser := ""
	if record.GetString("parent") != "" {
		// Validate parent is a comment for the same schematic
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
	schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
	if err != nil {
		return err
	}
	results, err := app.FindRecordsByFilter(
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

		u, err := app.FindRecordById("users", results[0].GetString("author"))
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
			HTML:    fmt.Sprintf("<p>A new comment has been posted on your CreateMod.com schematic: <a href=\"https://www.createmod.com/schematics/%s\">https://www.createmod.com/schematics/%s</a></p>", results[0].GetString("name"), results[0].GetString("name")),
		}
	} else {
		u, err := app.FindRecordById("users", replyToUser)
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

func validateAndPopulateSchematic(app *pocketbase.PocketBase, record *core.Record, e *core.RecordRequestEvent) error {
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

	files := make(map[string][]*filesystem.File, 0)

	// Check valid schematic file
	// TODO this can be improved by parsing the file
	if sf, err := e.FindUploadedFiles("schematic_file"); err == nil {
		if len(sf) == 0 || sf[0].Size <= 1 {
			return fmt.Errorf("schematic file invalid")
		}
		files["schematic_file"] = sf
	} else {
		return fmt.Errorf("schematic file missing or invalid")
	}

	// Check valid featured image
	if fi, err := e.FindUploadedFiles("featured_image"); err == nil {
		if len(fi) == 0 || fi[0].Size <= 1 {
			return fmt.Errorf("featured image invalid")
		}
		files["featured_image"] = fi
	} else {
		return fmt.Errorf("featured image missing or invalid")
	}

	if g, err := e.FindUploadedFiles("gallery"); err == nil {
		files["gallery"] = g
	}

	// convert to jpg in background
	go convertToJpg(app, record, files)

	// return nil if all is ok
	return nil
}

func convertToJpg(app *pocketbase.PocketBase, record *core.Record, files map[string][]*filesystem.File) {
	var galleryFilenames []string
	fs, err := app.NewFilesystem()
	if err != nil {
		return
	}

	for fieldKey := range files {
		if fieldKey == "featured_image" || fieldKey == "gallery" {
			for i, file := range files[fieldKey] {
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

				err = r.Close()
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
	}
	record.Set("gallery", galleryFilenames)
	err = app.Save(record)
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
	records, err := app.FindRecordsByFilter("schematics", "name={:slug}", "-created", 1, 0, dbx.Params{"slug": s})
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
