package server

import (
	"createmod/internal/migrate"
	"fmt"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"

	_ "createmod/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
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

	// serves static files from the provided public dir (if exists)
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// MIGRATIONS

		if s.conf.MysqlDB != "" {
			gormdb, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", s.conf.MysqlUser, s.conf.MysqlPass, s.conf.MysqlHost, s.conf.MysqlDB)))
			if err == nil {
				migrate.Run(app, gormdb)
			} else {
				log.Println("MIGRATION SKIPPED - No MySQL Connection")
			}
		}

		// END MIGRATIONS

		// ROUTES

		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))

		// END ROUTES
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
