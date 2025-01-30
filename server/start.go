package server

import (
	"createmod/internal/migrate"
	"createmod/internal/pages"
	"createmod/internal/router"
	"createmod/internal/search"
	_ "createmod/migrations"
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
	"github.com/sym01/htmlsanitizer"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"math/rand"
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
	app.Logger().Info("CreateMod.com Starting...")

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		Automigrate: s.conf.AutoMigrate,
	})

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

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
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
		app.Logger().Info("Starting Search Server")
		schematics, err := app.Dao().FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
		if err != nil {
			panic(err)
		}
		mappedSchematics := pages.MapResultsToSchematic(app, schematics)
		app.Logger().Debug("search service mapped schematics", "mapped schematic count", len(mappedSchematics))
		searchService = search.New(mappedSchematics, app.Logger())

		// END SEARCH

		// ROUTES

		router.Register(app, e.Router, searchService)

		// END ROUTES

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

	if err := app.Start(); err != nil {
		panic(err)
	}
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
	s := htmlsanitizer.NewHTMLSanitizer()
	s.Tags = []*htmlsanitizer.Tag{
		{"a", []string{"rel", "target", "referrerpolicy"}, []string{"href"}},
		{Name: "p"},
		{Name: "b"},
		{Name: "i"},
		{Name: "u"},
		{Name: "br"},
		{Name: "ul"},
		{Name: "ol"},
		{Name: "li"},
	}

	description, err := s.SanitizeString(description)
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

	// return nil if all is ok
	return nil
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
