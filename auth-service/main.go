package main

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"son514/auth-service/database/db"
	walletv1 "son514/auth-service/gen/wallet"
	authgrpc "son514/auth-service/grpc"
	"son514/auth-service/routes"

	grpcserver "son514/auth-service/grpc"

	"github.com/gin-gonic/gin"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "auth-service")
}

func requestLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		}
		if requestID := c.GetString("request_id"); requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}
		if userID, ok := c.Get("user_id"); ok {
			attrs = append(attrs, "user_id", userID)
		}

		if c.Writer.Status() >= http.StatusInternalServerError {
			logger.Error("request", attrs...)
		} else {
			logger.Info("request", attrs...)
		}
	}
}

func main() {
	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	walletAddr := os.Getenv("WALLET_GRPC_ADDR")
	if walletAddr == "" {
		walletAddr = "localhost:50053"
	}

	walletConn, err := authgrpc.NewWalletClient(walletAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer walletConn.Close()

	walletClient := walletv1.NewWalletServiceClient(walletConn)

	logger := newLogger()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLog(logger))

	routes.Setup(router, database, walletClient)

	go serveGRPC(logger)

	router.Run()
}

func serveGRPC(logger *slog.Logger) {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	log.Println("gRPC listening on", listener.Addr())

	if err := grpcserver.Serve(listener, logger); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
