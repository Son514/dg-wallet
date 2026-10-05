package main

import (
	"log"
	"net/http"

	"son514/wallet-service/database"

	"github.com/gin-gonic/gin"
)

func createWallet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"method": "POST"})
}

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	router.POST("/wallets", createWallet)
	router.Run()
}
