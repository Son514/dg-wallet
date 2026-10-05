package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"son514/wallet-service/database"
	"son514/wallet-service/models"
	"strconv"
	"strings"

	authv1 "son514/wallet-service/gen/auth"
	grpcclient "son514/wallet-service/grpc"

	"github.com/gin-gonic/gin"
)

func createWallet(db *sql.DB, auth authv1.AuthServiceClient, c *gin.Context) {
	token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}

	validated, err := grpcclient.ValidateToken(c.Request.Context(), auth, token)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	if !validated.GetValid() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": validated.GetReason()})
		return
	}

	userID, err := strconv.ParseInt(validated.GetUserId(), 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token has no user id"})
		return
	}

	wallet := models.NewWallet(userID)
	id, balance, err := wallet.CreateWallet(db)
	if err != nil {
		if errors.Is(err, models.ErrWalletExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"wallet_id": id, "balance": balance})
}

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	addr := os.Getenv("AUTH_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	conn, err := grpcclient.NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	auth := authv1.NewAuthServiceClient(conn)

	router := gin.Default()

	router.POST("/wallets", func(c *gin.Context) {
		createWallet(db, auth, c)
	})
	router.Run(":8081")
}
