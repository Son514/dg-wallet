package main

// TODO: US-3 — Create a Wallet

import (
	"log"
	"net"
	"os"

	"son514/auth-service/database/db"
	grpcserver "son514/auth-service/grpc"
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

	go serveGRPC()

	router.Run()
}

func serveGRPC() {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	log.Println("gRPC listening on", listener.Addr())

	if err := grpcserver.Serve(listener); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
