package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	authgrpc "github.com/alokwritescode/digi-vault/auth-service/internal/grpc"
	pb "github.com/alokwritescode/digi-vault/proto/auth"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	sharedjwt "github.com/alokwritescode/digi-vault/shared/pkg/jwt"
)

const testSecret = "test-secret-at-least-32-bytes!!"

// ─── Mock ────────────────────────────────────────────────────────────────────

type MockUserRepo struct{ mock.Mock }

func (m *MockUserRepo) Create(ctx context.Context, u *domain.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *MockUserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) UpdateIsActive(ctx context.Context, id uint64, active bool) error {
	return m.Called(ctx, id, active).Error(0)
}
func (m *MockUserRepo) SoftDelete(ctx context.Context, id uint64) error {
	return m.Called(ctx, id).Error(0)
}

// ─── ValidateToken tests ─────────────────────────────────────────────────────

func TestValidateToken_ValidJWT_ReturnsIsValidTrue(t *testing.T) {
	token, err := sharedjwt.GenerateTokenWithJTI("jti-abc", 42, testSecret, time.Minute)
	assert.NoError(t, err)

	srv := authgrpc.NewAuthServer(&MockUserRepo{}, testSecret)
	resp, err := srv.ValidateToken(context.Background(), &pb.ValidateTokenRequest{Token: token})

	assert.NoError(t, err)
	assert.True(t, resp.IsValid)
	assert.Equal(t, uint64(42), resp.UserId)
	assert.Equal(t, "jti-abc", resp.Jti)
}

func TestValidateToken_InvalidToken_ReturnsIsValidFalse(t *testing.T) {
	srv := authgrpc.NewAuthServer(&MockUserRepo{}, testSecret)
	resp, err := srv.ValidateToken(context.Background(), &pb.ValidateTokenRequest{Token: "garbage"})

	assert.NoError(t, err)
	assert.False(t, resp.IsValid)
}

func TestValidateToken_ExpiredToken_ReturnsIsValidFalse(t *testing.T) {
	token, err := sharedjwt.GenerateTokenWithJTI("jti-xyz", 7, testSecret, -time.Second)
	assert.NoError(t, err)

	srv := authgrpc.NewAuthServer(&MockUserRepo{}, testSecret)
	resp, err := srv.ValidateToken(context.Background(), &pb.ValidateTokenRequest{Token: token})

	assert.NoError(t, err)
	assert.False(t, resp.IsValid)
}

// ─── GetUser tests ────────────────────────────────────────────────────────────

func TestGetUser_UserExists_ReturnsUserResponse(t *testing.T) {
	repo := &MockUserRepo{}
	repo.On("FindByID", mock.Anything, uint64(1)).Return(&domain.User{
		ID:       1,
		Phone:    "+911234567890",
		Email:    "user@example.com",
		IsActive: true,
	}, nil)

	srv := authgrpc.NewAuthServer(repo, testSecret)
	resp, err := srv.GetUser(context.Background(), &pb.GetUserRequest{UserId: 1})

	assert.NoError(t, err)
	assert.Equal(t, uint64(1), resp.UserId)
	assert.Equal(t, "+911234567890", resp.Phone)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.True(t, resp.IsActive)
	repo.AssertExpectations(t)
}

func TestGetUser_UserNotFound_ReturnsNotFoundCode(t *testing.T) {
	repo := &MockUserRepo{}
	repo.On("FindByID", mock.Anything, uint64(99)).Return(nil, apperrors.ErrUserNotFound)

	srv := authgrpc.NewAuthServer(repo, testSecret)
	_, err := srv.GetUser(context.Background(), &pb.GetUserRequest{UserId: 99})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	repo.AssertExpectations(t)
}

func TestGetUser_DBError_ReturnsInternalCode(t *testing.T) {
	repo := &MockUserRepo{}
	repo.On("FindByID", mock.Anything, uint64(5)).Return(nil, assert.AnError)

	srv := authgrpc.NewAuthServer(repo, testSecret)
	_, err := srv.GetUser(context.Background(), &pb.GetUserRequest{UserId: 5})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	repo.AssertExpectations(t)
}
