package main

import (
	"createmod/server"
	"github.com/joho/godotenv"
	"log"
)

const (
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
		AutoMigrate: envFile[AutoMigrate] == "true",
		CreateAdmin: envFile[CreateAdmin] == "true",
	})
	s.Start()
}
