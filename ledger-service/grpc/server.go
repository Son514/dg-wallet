package grpc

import (
	"context"
	"database/sql"
	"log"
	"net"

	ledgerv1 "son514/ledger-service/gen/ledger"
	"son514/ledger-service/models"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	if requestID := incomingRequestID(ctx); requestID != "" {
		log.Printf("create_ledger_entries request_id=%s", requestID)
	}

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

func (s *Server) TransactionHistory(
	ctx context.Context,
	req *ledgerv1.TransactionHistoryRequest,
) (*ledgerv1.TransactionHistoryResponse, error) {
	if requestID := incomingRequestID(ctx); requestID != "" {
		log.Printf("transaction_history request_id=%s", requestID)
	}

	entries, err := models.TransactionHistory(s.db, req.GetWalletId())
	if err != nil {
		return nil, err
	}

	history := make([]*ledgerv1.TransactionHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		history = append(history, &ledgerv1.TransactionHistoryEntry{
			EntryId:   entry.ID,
			Amount:    entry.Amount,
			Type:      entry.Type,
			CreatedAt: timestamppb.New(entry.CreatedAt),
		})
	}

	return &ledgerv1.TransactionHistoryResponse{Entries: history}, nil
}

func incomingRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func Serve(listener net.Listener, db *sql.DB) error {
	server := googlegrpc.NewServer()
	ledgerv1.RegisterLedgerServiceServer(server, NewServer(db))
	return server.Serve(listener)
}
