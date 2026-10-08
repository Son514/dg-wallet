package grpc

import (
	"context"
	"errors"
	"log"
	"net"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

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
	if requestID := incomingRequestID(ctx); requestID != "" {
		log.Printf("validate_token request_id=%s", requestID)
	}

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

func Serve(listener net.Listener) error {
	server := googlegrpc.NewServer()
	authv1.RegisterAuthServiceServer(server, &Server{})

	return server.Serve(listener)
}
