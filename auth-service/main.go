package main

import (
	"log"
	"net"
	"os"
	"son514/auth-service/database/db"
	walletv1 "son514/auth-service/gen/wallet"
	authgrpc "son514/auth-service/grpc"
	"son514/auth-service/routes"

	grpcserver "son514/auth-service/grpc"

	"github.com/gin-gonic/gin"
)

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

	router := gin.Default()

	routes.Setup(router, database, walletClient)

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
