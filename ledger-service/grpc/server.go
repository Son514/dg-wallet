package grpc

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"time"

	ledgerv1 "son514/ledger-service/gen/ledger"
	"son514/ledger-service/models"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func unaryRequestLog(logger *slog.Logger) googlegrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *googlegrpc.UnaryServerInfo,
		handler googlegrpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		attrs := []any{
			"rpc", info.FullMethod,
			"status", status.Code(err).String(),
			"latency_ms", time.Since(start).Milliseconds(),
		}
		if requestID := incomingRequestID(ctx); requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}

		if err != nil {
			logger.Error("grpc_request", attrs...)
		} else {
			logger.Info("grpc_request", attrs...)
		}
		return resp, err
	}
}

func Serve(listener net.Listener, db *sql.DB, logger *slog.Logger) error {
	server := googlegrpc.NewServer(googlegrpc.ChainUnaryInterceptor(unaryRequestLog(logger)))
	ledgerv1.RegisterLedgerServiceServer(server, NewServer(db))
	return server.Serve(listener)
}
