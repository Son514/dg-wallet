package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func createWallet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"method": "POST"})
}

func main() {
	router := gin.Default()

	router.POST("/wallets", createWallet)
	router.Run()
}
