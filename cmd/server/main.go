package main

import (
	"createmod/server"
	"github.com/joho/godotenv"
	"log"
)

const (
	MysqlHost   = "MYSQL_HOST"
	MysqlDB     = "MYSQL_DB"
	MysqlUser   = "MYSQL_USER"
	MysqlPass   = "MYSQL_PASS"
	AutoMigrate = "AUTO_MIGRATE"
	CreateAdmin = "CREATE_ADMIN"
)

func main() {
	envFile, err := godotenv.Read(".env")

	if err != nil {
		// Continue without env but print error
		log.Println(err)
	}

	s := server.New(server.Config{
		MysqlHost:   envFile[MysqlHost],
		MysqlDB:     envFile[MysqlDB],
		MysqlUser:   envFile[MysqlUser],
		MysqlPass:   envFile[MysqlPass],
		AutoMigrate: envFile[AutoMigrate] == "true",
		CreateAdmin: envFile[CreateAdmin] == "true",
	})
	s.Start()
}
