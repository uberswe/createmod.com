package server

import (
	"bufio"
	"bytes"
	"createmod/internal/aidescription"
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/moderation"
	"createmod/internal/pages"
	"createmod/internal/router"
	"createmod/internal/search"
	"createmod/internal/sitemap"
	_ "createmod/migrations"
	"errors"
	"fmt"
	"github.com/apokalyptik/phpass"
	"github.com/drexedam/gravatar"
	"github.com/gosimple/slug"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/sunshineplan/imgconv"
	"github.com/sym01/htmlsanitizer"
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
	AutoMigrate       bool
	CreateAdmin       bool
	DiscordWebhookUrl string
	OpenAIApiKey      string
}

type Server struct {
	conf                 Config
	app                  *pocketbase.PocketBase
	searchService        *search.Service
	sitemapService       *sitemap.Service
	cacheService         *cache.Service
	discordService       *discord.Service
	moderationService    *moderation.Service
	aiDescriptionService *aidescription.Service
}

func New(conf Config) *Server {
	app := pocketbase.New()
	sitemapService := sitemap.New()
	discordService := discord.New(conf.DiscordWebhookUrl)
	moderationService := moderation.NewService(conf.OpenAIApiKey, app.Logger())
	aiDescriptionService := aidescription.New(conf.OpenAIApiKey, app.Logger())
	return &Server{
		conf:                 conf,
		app:                  app,
		sitemapService:       sitemapService,
		cacheService:         cache.New(),
		discordService:       discordService,
		moderationService:    moderationService,
		aiDescriptionService: aiDescriptionService,
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

		log.Println("Starting Search Server")
		schematics, err := s.app.FindRecordsByFilter("schematics", "deleted = null && moderated = true", "-created", -1, 0)
		if err != nil {
			s.app.Logger().Error(err.Error())
		}
		mappedSchematics := pages.MapResultsToSchematic(s.app, schematics, s.cacheService)
		s.app.Logger().Debug("search service mapped schematics", "mapped schematic count", len(mappedSchematics))
		s.searchService = search.New(mappedSchematics, s.app)
		s.sitemapService.Generate(s.app)

		// Start the AI description service scheduler (polls every 30 minutes)
		s.aiDescriptionService.StartScheduler(s.app)

		s.app.OnRecordCreateExecute("schematics").BindFunc(func(e *core.RecordEvent) error {
			if !validNBT(e) {
				return fmt.Errorf("invalid NBT file")
			}

			// Rebuild the search index every time a schematic is created
			err := e.Next()
			if err != nil {
				return err
			}
			go func() {
				schematics, err := s.app.FindRecordsByFilter("schematics", "deleted = null && moderated = true", "-created", -1, 0)
				if err != nil {
					s.app.Logger().Warn(err.Error())
				}
				s.searchService.BuildIndex(pages.MapResultsToSchematic(s.app, schematics, s.cacheService))
				s.sitemapService.Generate(s.app)
			}()
			return nil
		})

		s.app.OnRecordUpdate("schematics").BindFunc(func(e *core.RecordEvent) error {
			if !validNBT(e) {
				return fmt.Errorf("invalid NBT file")
			}

			err = e.Next()
			if err != nil {
				return err
			}
			s.cacheService.DeleteSchematic(cache.SchematicKey(e.Record.Id))
			go func() {
				schematics, err := s.app.FindRecordsByFilter("schematics", "deleted = null && moderated = true", "-created", -1, 0)
				if err != nil {
					s.app.Logger().Warn(err.Error())
				}
				s.searchService.BuildIndex(pages.MapResultsToSchematic(s.app, schematics, s.cacheService))
				s.sitemapService.Generate(s.app)
			}()
			return nil
		})

		s.app.OnRecordDeleteExecute("schematics").BindFunc(func(e *core.RecordEvent) error {
			e.Record.Set("deleted", time.Now())
			err := e.App.Save(e.Record)
			if err != nil {
				return err
			}
			s.cacheService.DeleteSchematic(cache.SchematicKey(e.Record.Id))
			// Prevent actual deletion
			go func() {
				schematics, err := s.app.FindRecordsByFilter("schematics", "deleted = null && moderated = true", "-created", -1, 0)
				if err != nil {
					s.app.Logger().Warn(err.Error())
				}
				s.searchService.BuildIndex(pages.MapResultsToSchematic(s.app, schematics, s.cacheService))
				s.sitemapService.Generate(s.app)
			}()
			return nil
		})

		s.app.OnRecordCreateExecute("users").BindFunc(func(e *core.RecordEvent) error {
			avatarUrl := gravatar.New(e.Record.GetString("email")).
				Size(200).
				Default(gravatar.NotFound).
				Rating(gravatar.Pg).
				AvatarURL()
			e.Record.Set("avatar", avatarUrl)
			err := e.Next()
			if err != nil {
				return err
			}
			// Rebuild the sitemap every time a user is created
			go s.sitemapService.Generate(s.app)
			return nil
		})

		s.app.OnRecordUpdate("users").BindFunc(func(e *core.RecordEvent) error {
			avatarUrl := gravatar.New(e.Record.GetString("email")).
				Size(200).
				Default(gravatar.MysteryMan).
				Rating(gravatar.Pg).
				AvatarURL()
			e.Record.Set("avatar", avatarUrl)
			err := e.Next()
			if err != nil {
				return err
			}
			return nil
		})

		s.app.OnRecordDeleteExecute("users").BindFunc(func(e *core.RecordEvent) error {
			schemRes, err := e.App.FindRecordsByFilter("schematics", "deleted = null && author = {:author}", "-created", -1, 0, dbx.Params{"author": e.Record.Id})
			if err != nil {
				return err
			}
			// Delete all schematics if a user is deleted
			for _, r := range schemRes {
				s.cacheService.DeleteSchematic(cache.SchematicKey(r.Id))
				r.Set("deleted", time.Now())
				err = e.App.Save(r)
				if err != nil {
					return err
				}
			}
			e.Record.Set("deleted", time.Now())
			err = e.App.Save(e.Record)
			if err != nil {
				return err
			}
			// Prevent actual deletion
			go func() {
				s.sitemapService.Generate(s.app)
			}()
			return nil
		})

		s.app.OnRecordCreateRequest("schematics").BindFunc(func(e *core.RecordRequestEvent) error {
			if e.Auth == nil {
				return fmt.Errorf("user is not authenticated")
			}
			s.app.Logger().Debug("setting author", "id", e.Auth.Id, "username", e.Auth.GetString("username"))
			e.Record.Set("author", e.Auth.Id)
			e.Record.Set("postdate", time.Now())
			e.Record.Set("modified", time.Now())

			if err := validateAndPopulateSchematic(s.app, e.Record, e); err != nil {
				return err
			}

			// Initially set as unmoderated (pending moderation)
			e.Record.Set("moderated", false)

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

		s.app.OnRecordAfterCreateSuccess("schematics").BindFunc(func(e *core.RecordEvent) error {
			message := &mailer.Message{
				From: mail.Address{
					Address: s.app.Settings().Meta.SenderAddress,
					Name:    s.app.Settings().Meta.SenderName,
				},
				To:      []mail.Address{{Address: s.app.Settings().Meta.SenderAddress}},
				Subject: fmt.Sprintf("New CreateMod.com Schematic"),
				HTML:    fmt.Sprintf("<p>Title: <a href=\"https://createmod.com/schematics/" + e.Record.GetString("name") + "\">" + e.Record.GetString("title") + "</a></p><p>Description: " + e.Record.GetString("description") + "</p>"),
			}

			// Start asynchronous moderation check
			go func() {
				title := e.Record.GetString("title")
				description := e.Record.GetString("description")
				featuredImage := fmt.Sprintf("https://createmod.com/api/files/%s/%s", e.Record.BaseFilesPath(), e.Record.GetString("featured_image"))

				s.app.Logger().Debug("Starting async moderation check for schematic",
					"id", e.Record.Id,
					"title", title,
					"featured_image", featuredImage)

				result, err := s.moderationService.CheckSchematic(title, description, featuredImage)
				if err != nil {
					s.app.Logger().Error("Failed to check schematic content", "error", err, "id", e.Record.Id)
					// If moderation fails, we'll leave the schematic as unmoderated
					return
				}

				// Get a fresh copy of the record to update
				record, err := s.app.FindRecordById("schematics", e.Record.Id)
				if err != nil {
					s.app.Logger().Error("Failed to find schematic record for moderation update", "error", err, "id", e.Record.Id)
					return
				}

				if result.Approved {
					// Content is approved by moderation service, now check quality
					s.app.Logger().Debug("Schematic content approved by moderation service, checking quality", "id", e.Record.Id)

					// Perform quality check
					qualityResult, err := s.moderationService.CheckSchematicQuality(title, description)
					if err != nil {
						s.app.Logger().Error("Failed to check schematic quality", "error", err, "id", e.Record.Id)
						// If quality check fails, we'll approve the schematic as a fallback
						record.Set("moderated", true)
					} else if qualityResult.Approved {
						// Schematic passed quality check
						s.app.Logger().Debug("Schematic passed quality check", "id", e.Record.Id)
						record.Set("moderated", true)
					} else {
						// Schematic failed quality check
						s.app.Logger().Debug("Schematic failed quality check",
							"id", e.Record.Id,
							"reason", qualityResult.Reason)
						record.Set("moderated", false)
						record.Set("deleted_at", time.Now())
						record.Set("moderation_reason", qualityResult.Reason)
					}
				} else {
					// Content is not approved by moderation service
					s.app.Logger().Debug("Schematic content rejected by moderation service",
						"id", e.Record.Id,
						"reason", result.Reason)
					record.Set("moderated", false)
					record.Set("deleted_at", time.Now())
					record.Set("moderation_reason", result.Reason)
				}

				// Save the updated record
				if err := s.app.Save(record); err != nil {
					s.app.Logger().Error("Failed to save schematic record after moderation",
						"error", err,
						"id", e.Record.Id)
				}
			}()

			return s.app.NewMailClient().Send(message)
		})

		s.app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
			// prevent deleted users from logging in
			if !e.Record.GetDateTime("deleted").IsZero() {
				return fmt.Errorf("deleted user can not login")
			}
			// COOKIES
			e.SetCookie(&http.Cookie{
				Name:     auth.CookieName,
				Value:    e.Token,
				Expires:  time.Now().Add(time.Second * time.Duration(e.Record.Collection().AuthToken.Duration)),
				Path:     "/",
				SameSite: http.SameSiteStrictMode,
			})
			// END COOKIES
			return e.Next()
		})

		// PASSWORD BACKWARDS COMPATIBILITY
		s.app.OnRecordAuthWithPasswordRequest("users").BindFunc(func(e *core.RecordAuthWithPasswordRequestEvent) error {
			s.app.Logger().Debug("OnRecordAuthWithPasswordRequest", "record", e.Record)
			if e.Record != nil && e.Record.GetString("old_password") != "" {
				p := phpass.New(nil)
				if p.Check([]byte(e.Password), []byte(e.Record.GetString("old_password"))) {
					e.Record.SetPassword(e.Password)
					e.Record.Set("old_password", "")
				}
			}
			return e.Next()
		})
		// END PASSWORD BACKWARDS COMPATIBILITY

		// ROUTES

		router.Register(s.app, e.Router, s.searchService, s.cacheService, s.discordService)

		// END ROUTES

		log.Println("CreateMod.com Running!")
		return e.Next()
	})

	log.Println("Initializing...")
	if err := s.app.Start(); err != nil {
		panic(err)
	}
}

func validNBT(e *core.RecordEvent) bool {
	files := e.Record.GetUnsavedFiles("schematic_file")
	for _, f := range files {
		if f.Size == 0 || !strings.HasSuffix(f.OriginalName, ".nbt") {
			return false
		}
	}
	// no files may be submitted on update
	return true
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

	record, err = convertToJpg(app, record)

	if err != nil {
		return err
	}

	// return nil if all is ok
	return nil
}

func convertToJpg(app *pocketbase.PocketBase, record *core.Record) (*core.Record, error) {
	var err error
	unsavedFiles := record.GetUnsavedFiles("featured_image")
	record, err = convertInLoop("featured_image", unsavedFiles, record)
	if err != nil {
		return record, err
	}
	unsavedFiles = record.GetUnsavedFiles("gallery")
	record, err = convertInLoop("gallery", unsavedFiles, record)
	if err != nil {
		return record, err
	}

	return record, nil
}

func convertInLoop(key string, unsavedFiles []*filesystem.File, record *core.Record) (*core.Record, error) {
	var convertedFiles []*filesystem.File
	for _, f := range unsavedFiles {
		rsc, err := f.Reader.Open()
		if err != nil {
			return record, err
		}
		decode, err := imgconv.Decode(rsc)
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

		filename := strings.TrimSuffix(f.Name, filepath.Ext(f.Name)) + ".jpg"
		if err != nil {
			return record, err
		}

		newFile, err := filesystem.NewFileFromBytes(jpgBuffer.Bytes(), filename)
		if err != nil {
			return record, err
		}

		convertedFiles = append(convertedFiles, newFile)

	}
	record.Set(key, convertedFiles)
	return record, nil
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
