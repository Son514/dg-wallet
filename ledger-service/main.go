package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	"son514/ledger-service/database"
	grpcserver "son514/ledger-service/grpc"
)

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	serveGRPC(db)
}

func serveGRPC(db *sql.DB) {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50052"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	log.Println("gRPC listening on", listener.Addr())

	if err := grpcserver.Serve(listener, db); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
