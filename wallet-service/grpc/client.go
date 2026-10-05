package grpc

import (
	"context"
	"time"

	authv1 "son514/wallet-service/gen/auth"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const timeout = 3 * time.Second

func NewClient(addr string) (*googlegrpc.ClientConn, error) {
	return googlegrpc.NewClient(
		addr,
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func ValidateToken(
	ctx context.Context,
	client authv1.AuthServiceClient,
	token string,
) (*authv1.ValidateTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return client.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
}