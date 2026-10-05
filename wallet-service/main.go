package main

import (
	"database/sql"
	"log"
	"net/http"

	"son514/wallet-service/database"
	"son514/wallet-service/models"

	"github.com/gin-gonic/gin"
)

func createWallet(db *sql.DB, c *gin.Context) {
	wallet := models.NewWallet()
	id, err := wallet.CreateWallet(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"wallet_id": id})
}

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	router.POST("/wallets", func(c *gin.Context) {
		createWallet(db, c)
	})
	router.Run()
}
