package server

import (
	"createmod/internal/migrate"
	"createmod/internal/pages"
	"createmod/internal/router"
	"createmod/internal/search"
	_ "createmod/migrations"
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		Automigrate: s.conf.AutoMigrate,
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

	// END SEARCH

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// SEARCH
		schematics, err := app.Dao().FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
		if err != nil {
			panic(err)
		}
		mappedSchematics := pages.MapResultsToSchematic(app, schematics)
		app.Logger().Debug("search service mapped schematics", "mapped schematic count", len(mappedSchematics))
		searchService = search.New(mappedSchematics, app.Logger())

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

		// ROUTES

		router.Register(app, e.Router, searchService)

		// END ROUTES
		return nil
	})

	if err := app.Start(); err != nil {
		panic(err)
	}
}
