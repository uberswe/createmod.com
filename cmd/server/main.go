package main

import (
	"createmod/server"
	"github.com/joho/godotenv"
)

const (
	MysqlHost = "MYSQL_HOST"
	MysqlDB   = "MYSQL_DB"
	MysqlUser = "MYSQL_USER"
	MysqlPass = "MYSQL_PASS"
)

func main() {
	envFile, err := godotenv.Read(".env")

	if err != nil {
		panic(err)
	}

	s := server.New(server.Config{
		MysqlHost: envFile[MysqlHost],
		MysqlDB:   envFile[MysqlDB],
		MysqlUser: envFile[MysqlUser],
		MysqlPass: envFile[MysqlPass],
	})
	s.Start()
}
