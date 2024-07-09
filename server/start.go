package server

import (
	"createmod/internal/migrate"
	"createmod/internal/router"
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

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// enable auto creation of migration files when making collection changes in the Admin UI
		Automigrate: s.conf.AutoMigrate,
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

		// ROUTES

		router.Register(app, e.Router)

		// END ROUTES
		return nil
	})

	if err := app.Start(); err != nil {
		panic(err)
	}
}
