package grpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "son514/auth-service/gen/auth"
	"son514/auth-service/jwt"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
}

func (s *Server) ValidateToken(
	ctx context.Context,
	req *authv1.ValidateTokenRequest,
) (*authv1.ValidateTokenResponse, error) {
	validated, err := jwt.Validate(req.GetToken())
	if err != nil {
		return &authv1.ValidateTokenResponse{
			Valid:  false,
			Reason: reason(err),
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		Valid:  true,
		UserId: validated.UserID,
		Email:  validated.Email,
	}, nil
}

func incomingRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func reason(err error) string {
	switch {
	case errors.Is(err, jwtlib.ErrTokenExpired):
		return "token is expired"
	case errors.Is(err, jwtlib.ErrTokenSignatureInvalid):
		return "token signature is invalid"
	case errors.Is(err, jwtlib.ErrTokenMalformed):
		return "token is malformed"
	default:
		return "token is invalid"
	}
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
		if validated, ok := resp.(*authv1.ValidateTokenResponse); ok && validated.GetUserId() != "" {
			attrs = append(attrs, "user_id", validated.GetUserId())
		}

		if err != nil {
			logger.Error("grpc_request", attrs...)
		} else {
			logger.Info("grpc_request", attrs...)
		}
		return resp, err
	}
}

func Serve(listener net.Listener, logger *slog.Logger) error {
	server := googlegrpc.NewServer(googlegrpc.ChainUnaryInterceptor(unaryRequestLog(logger)))
	authv1.RegisterAuthServiceServer(server, &Server{})

	return server.Serve(listener)
}
