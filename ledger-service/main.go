package main

import (
	"database/sql"
	"log"
	"log/slog"
	"net"
	"os"

	"son514/ledger-service/database"
	grpcserver "son514/ledger-service/grpc"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "ledger-service")
}

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	serveGRPC(db, newLogger())
}

func serveGRPC(db *sql.DB, logger *slog.Logger) {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50052"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	log.Println("gRPC listening on", listener.Addr())

	if err := grpcserver.Serve(listener, db, logger); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
