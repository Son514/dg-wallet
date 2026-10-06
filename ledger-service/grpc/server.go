package grpc

import (
	"context"
	"database/sql"
	"net"

	ledgerv1 "son514/ledger-service/gen/ledger"
	"son514/ledger-service/models"

	googlegrpc "google.golang.org/grpc"
)

type Server struct {
	ledgerv1.UnimplementedLedgerServiceServer
	db *sql.DB
}

func NewServer(db *sql.DB) *Server {
	return &Server{db: db}
}

func (s *Server) CreateLedgerEntries(
	ctx context.Context,
	req *ledgerv1.CreateLedgerEntriesRequest,
) (*ledgerv1.CreateLedgerEntriesResponse, error) {
	entries := make([]models.LedgerEntry, 0, len(req.GetEntries()))
	for _, e := range req.GetEntries() {
		entries = append(entries, models.LedgerEntry{
			WalletID: e.GetWalletId(),
			Amount:   e.GetAmount(),
			Type:     e.GetType(),
		})
	}

	ids, err := models.CreateLedgerEntries(s.db, entries)
	if err != nil {
		return nil, err
	}

	return &ledgerv1.CreateLedgerEntriesResponse{EntryIds: ids}, nil
}

func Serve(listener net.Listener, db *sql.DB) error {
	server := googlegrpc.NewServer()
	ledgerv1.RegisterLedgerServiceServer(server, NewServer(db))
	return server.Serve(listener)
}
