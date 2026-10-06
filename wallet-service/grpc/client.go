package grpc

import (
	"context"
	"time"

	authv1 "son514/wallet-service/gen/auth"
	ledgerv1 "son514/wallet-service/gen/ledger"

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

func CreateLedgerEntries(
	ctx context.Context,
	client ledgerv1.LedgerServiceClient,
	entries []*ledgerv1.LedgerEntry,
) (*ledgerv1.CreateLedgerEntriesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return client.CreateLedgerEntries(ctx, &ledgerv1.CreateLedgerEntriesRequest{Entries: entries})
}
