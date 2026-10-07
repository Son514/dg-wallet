package grpc

import (
	"context"
	"time"

	walletv1 "son514/auth-service/gen/wallet"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const walletCallTimeout = 3 * time.Second

func NewWalletClient(addr string) (*googlegrpc.ClientConn, error) {
	return googlegrpc.NewClient(
		addr,
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func CreateWallet(
	ctx context.Context,
	client walletv1.WalletServiceClient,
	token string,
) (*walletv1.CreateWalletResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, walletCallTimeout)
	defer cancel()

	return client.CreateWallet(ctx, &walletv1.CreateWalletRequest{Token: token})
}
