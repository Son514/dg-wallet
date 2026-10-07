package main

// TODO: US-7 — Ledger Entries (Double-Entry)

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"son514/wallet-service/database"
	"son514/wallet-service/models"
	"strconv"
	"strings"

	authv1 "son514/wallet-service/gen/auth"
	ledgerv1 "son514/wallet-service/gen/ledger"
	grpcclient "son514/wallet-service/grpc"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

func topUpWallet(db *sql.DB, auth authv1.AuthServiceClient, ledger ledgerv1.LedgerServiceClient, c *gin.Context) {
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

	_, err = grpcclient.CreateLedgerEntries(c.Request.Context(), ledger, []*ledgerv1.LedgerEntry{
		{WalletId: 0, Amount: "-" + request.Amount, Type: "topup"},
		{WalletId: id, Amount: request.Amount, Type: "topup"},
	})
	if err != nil {
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

func transferMoney(db *sql.DB, auth authv1.AuthServiceClient, ledger ledgerv1.LedgerServiceClient, redisClient *redis.Client, c *gin.Context) {
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing Idempotency-Key header"})
		return
	}

	userID, ok := authenticatedUserID(auth, c)
	if !ok {
		return
	}

	cached, err := getTransferIdempotency(
		c.Request.Context(),
		redisClient,
		userID,
		idempotencyKey,
	)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if cached != nil {
		c.Data(cached.StatusCode, "application/json; charset=utf-8", []byte(cached.Body))
		return
	}

	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statusCode, response := executeTransfer(db, ledger, userID, c, requestBody)
	responseBody, err := json.Marshal(response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := storeTransferIdempotency(
		c.Request.Context(),
		redisClient,
		userID,
		idempotencyKey,
		statusCode,
		responseBody,
	); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.Data(statusCode, "application/json; charset=utf-8", responseBody)
}

func executeTransfer(
	db *sql.DB,
	ledger ledgerv1.LedgerServiceClient,
	userID int64,
	c *gin.Context,
	requestBody []byte,
) (int, gin.H) {
	var request transferRequest
	if err := json.Unmarshal(requestBody, &request); err != nil {
		return http.StatusBadRequest, gin.H{"error": err.Error()}
	}

	amount, err := strconv.ParseFloat(request.Amount, 64)
	if err != nil || amount <= 0 {
		return http.StatusBadRequest, gin.H{"error": "amount must be greater than zero"}
	}

	if request.FromWalletID == request.ToWalletID {
		return http.StatusBadRequest, gin.H{"error": "cannot transfer to self"}
	}

	wallet := models.NewWallet(userID)
	id, balance, err := wallet.TransferMoney(db, request.FromWalletID, request.ToWalletID, request.Amount)
	if err != nil {
		if errors.Is(err, models.ErrTransferToSelf) {
			return http.StatusBadRequest, gin.H{"error": err.Error()}
		}
		if errors.Is(err, models.ErrNotWalletOwner) {
			return http.StatusForbidden, gin.H{"error": err.Error()}
		}
		if errors.Is(err, models.ErrInsufficientBalance) {
			return http.StatusBadRequest, gin.H{"error": err.Error()}
		}
		if errors.Is(err, sql.ErrNoRows) {
			return http.StatusNotFound, gin.H{"error": "wallet not found"}
		}
		return http.StatusInternalServerError, gin.H{"error": err.Error()}
	}

	_, err = grpcclient.CreateLedgerEntries(c.Request.Context(), ledger, []*ledgerv1.LedgerEntry{
		{WalletId: request.FromWalletID, Amount: "-" + request.Amount, Type: "transfer_out"},
		{WalletId: request.ToWalletID, Amount: request.Amount, Type: "transfer_in"},
	})
	if err != nil {
		return http.StatusInternalServerError, gin.H{"error": err.Error()}
	}

	return http.StatusOK, gin.H{"wallet_id": id, "balance": balance}
}

func transactionHistory(db *sql.DB, auth authv1.AuthServiceClient, ledger ledgerv1.LedgerServiceClient, c *gin.Context) {
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
	if _, _, err := wallet.CheckBalance(db, walletID); err != nil {
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

	history, err := grpcclient.TransactionHistory(c.Request.Context(), ledger, walletID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	entries := make([]gin.H, 0, len(history.GetEntries()))
	for _, entry := range history.GetEntries() {
		entries = append(entries, gin.H{
			"entry_id":   entry.GetEntryId(),
			"amount":     entry.GetAmount(),
			"type":       entry.GetType(),
			"created_at": entry.GetCreatedAt().AsTime(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"wallet_id": walletID, "entries": entries})
}

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal(err)
	}
	redisClient := redis.NewClient(redisOptions)
	defer redisClient.Close()

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

	ledgerAddr := os.Getenv("LEDGER_GRPC_ADDR")
	if ledgerAddr == "" {
		ledgerAddr = "localhost:50052"
	}

	ledgerConn, err := grpcclient.NewClient(ledgerAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ledgerConn.Close()

	ledger := ledgerv1.NewLedgerServiceClient(ledgerConn)

	go serveGRPC(db, auth)

	router := gin.Default()
	/* --- Create a Wallet --- */
	router.POST("/wallets", func(c *gin.Context) {
		createWallet(db, auth, c)
	})

	/* --- Top-up to Wallet --- */
	router.POST("/wallets/:id/topup", func(c *gin.Context) {
		topUpWallet(db, auth, ledger, c)
	})

	/* --- Check Balance --- */
	router.GET("/wallets/:id", func(c *gin.Context) {
		checkBalance(db, auth, c)
	})

	/* --- Transfer Between Wallet --- */
	router.POST("/transfers", func(c *gin.Context) {
		transferMoney(db, auth, ledger, redisClient, c)
	})

	/* --- Transaction History --- */
	router.GET("/wallets/:id/transactions", func(c *gin.Context) {
		transactionHistory(db, auth, ledger, c)
	})

	router.Run(":8081")
}

func serveGRPC(db *sql.DB, auth authv1.AuthServiceClient) {
	port := os.Getenv("WALLET_GRPC_PORT")
	if port == "" {
		port = "50053"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	log.Println("wallet gRPC listening on", listener.Addr())

	if err := grpcclient.Serve(listener, db, auth); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
