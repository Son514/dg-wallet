package main

// TODO: Transfer Between Wallets

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

type topUpRequest struct {
	Amount string `json:"amount"`
}

type transferRequest struct {
	FromWalletID int64  `json:"from_wallet_id"`
	ToWalletID   int64  `json:"to_wallet_id"`
	Amount       string `json:"amount"`
}

// authenticatedUserID resolves the caller's id from the bearer token. On
// failure it writes the response itself and reports false, so callers just
// return.
func authenticatedUserID(auth authv1.AuthServiceClient, c *gin.Context) (int64, bool) {
	token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return 0, false
	}

	validated, err := grpcclient.ValidateToken(c.Request.Context(), auth, token)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return 0, false
	}

	if !validated.GetValid() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": validated.GetReason()})
		return 0, false
	}

	userID, err := strconv.ParseInt(validated.GetUserId(), 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token has no user id"})
		return 0, false
	}

	return userID, true
}

func createWallet(db *sql.DB, auth authv1.AuthServiceClient, c *gin.Context) {
	userID, ok := authenticatedUserID(auth, c)
	if !ok {
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

func topUpWallet(db *sql.DB, auth authv1.AuthServiceClient, c *gin.Context) {
	walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet id must be a number"})
		return
	}

	userID, ok := authenticatedUserID(auth, c)
	if !ok {
		return
	}

	var request topUpRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	amount, err := strconv.ParseFloat(request.Amount, 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than zero"})
		return
	}

	wallet := models.NewWallet(userID)
	id, balance, err := wallet.TopUpWallet(db, walletID, request.Amount)
	if err != nil {
		if errors.Is(err, models.ErrNotWalletOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet_id": id, "balance": balance})
}

func checkBalance(db *sql.DB, auth authv1.AuthServiceClient, c *gin.Context) {
	walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet id must be a number"})
		return
	}

	userID, ok := authenticatedUserID(auth, c)
	if !ok {
		return
	}

	wallet := models.NewWallet(userID)
	id, balance, err := wallet.CheckBalance(db, walletID)
	if err != nil {
		if errors.Is(err, models.ErrNotWalletOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet_id": id, "balance": balance})
}

func transferMoney(db *sql.DB, auth authv1.AuthServiceClient, c *gin.Context) {
	userID, ok := authenticatedUserID(auth, c)
	if !ok {
		return
	}

	var request transferRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	amount, err := strconv.ParseFloat(request.Amount, 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than zero"})
		return
	}

	wallet := models.NewWallet(userID)
	id, balance, err := wallet.TransferMoney(db, request.FromWalletID, request.ToWalletID, request.Amount)
	if err != nil {
		if errors.Is(err, models.ErrNotWalletOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, models.ErrInsufficientBalance) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet_id": id, "balance": balance})
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
	/* --- Create a Wallet --- */
	router.POST("/wallets", func(c *gin.Context) {
		createWallet(db, auth, c)
	})

	/* --- Top-up to Wallet --- */
	router.POST("/wallets/:id/topup", func(c *gin.Context) {
		topUpWallet(db, auth, c)
	})

	/* --- Check Balance --- */
	router.GET("/wallets/:id", func(c *gin.Context) {
		checkBalance(db, auth, c)
	})

	/* --- Transfer Between Wallet --- */
	router.POST("/transfers", func(c *gin.Context) {
		transferMoney(db, auth, c)
	})

	router.Run(":8081")
}
