package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/alokwritescode/digi-vault/auth-service/internal/repository"
	pb "github.com/alokwritescode/digi-vault/proto/auth"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/alokwritescode/digi-vault/shared/pkg/jwt"
)

// AuthServer implements pb.AuthServiceServer.
// Called by other services (api-gateway) over gRPC to validate tokens and fetch user profiles.
type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	userRepo        repository.UserRepository
	jwtAccessSecret string
}

func NewAuthServer(userRepo repository.UserRepository, jwtAccessSecret string) *AuthServer {
	return &AuthServer{
		userRepo:        userRepo,
		jwtAccessSecret: jwtAccessSecret,
	}
}

// ValidateToken parses and verifies a JWT access token.
// Returns is_valid: false (no error) so the caller can return 401 cleanly.
func (s *AuthServer) ValidateToken(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims, err := jwt.ParseToken(req.Token, s.jwtAccessSecret)
	if err != nil {
		return &pb.ValidateTokenResponse{IsValid: false}, nil
	}
	return &pb.ValidateTokenResponse{
		IsValid: true,
		UserId:  claims.UserID,
		Jti:     claims.JTI,
	}, nil
}

// GetUser fetches a user profile by ID for cross-service use.
func (s *AuthServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	return &pb.GetUserResponse{
		UserId:   user.ID,
		Phone:    user.Phone,
		Email:    user.Email,
		IsActive: user.IsActive,
	}, nil
}
