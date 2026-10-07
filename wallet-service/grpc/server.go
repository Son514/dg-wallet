package grpc

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strconv"

	authv1 "son514/wallet-service/gen/auth"
	walletv1 "son514/wallet-service/gen/wallet"
	"son514/wallet-service/models"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	walletv1.UnimplementedWalletServiceServer
	db   *sql.DB
	auth authv1.AuthServiceClient
}

func NewServer(db *sql.DB, auth authv1.AuthServiceClient) *Server {
	return &Server{db: db, auth: auth}
}

func (s *Server) CreateWallet(
	ctx context.Context,
	req *walletv1.CreateWalletRequest,
) (*walletv1.CreateWalletResponse, error) {
	validated, err := ValidateToken(ctx, s.auth, req.GetToken())
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "token validation failed: %v", err)
	}
	if !validated.GetValid() {
		return nil, status.Error(codes.Unauthenticated, validated.GetReason())
	}

	userID, err := strconv.ParseInt(validated.GetUserId(), 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "token has no valid user id")
	}

	id, balance, err := models.NewWallet(userID).CreateWallet(s.db)
	if err != nil {
		if errors.Is(err, models.ErrWalletExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create wallet: %v", err)
	}

	return &walletv1.CreateWalletResponse{
		WalletId: id,
		Balance:  balance,
	}, nil
}

func Serve(listener net.Listener, db *sql.DB, auth authv1.AuthServiceClient) error {
	server := googlegrpc.NewServer()
	walletv1.RegisterWalletServiceServer(server, NewServer(db, auth))
	return server.Serve(listener)
}
