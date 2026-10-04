package main

// TODO: US-3 — Create a Wallet

import (
	"log"

	"son514/auth-service/database/db"
	"son514/auth-service/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	router := gin.Default()

	routes.Setup(router, database)

	router.Run()
}
