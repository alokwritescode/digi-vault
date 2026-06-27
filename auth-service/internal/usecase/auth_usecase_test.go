package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/alokwritescode/digi-vault/auth-service/config"
	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/alokwritescode/digi-vault/auth-service/internal/usecase"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/alokwritescode/digi-vault/shared/pkg/jwt"
	"github.com/alokwritescode/digi-vault/shared/pkg/logger"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *MockUserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) UpdateIsActive(ctx context.Context, userID uint64, isActive bool) error {
	return m.Called(ctx, userID, isActive).Error(0)
}

type MockTokenRepository struct{ mock.Mock }

func (m *MockTokenRepository) Store(ctx context.Context, token *domain.RefreshToken) error {
	return m.Called(ctx, token).Error(0)
}
func (m *MockTokenRepository) Find(ctx context.Context, jti string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, jti)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}
func (m *MockTokenRepository) Delete(ctx context.Context, jti string) error {
	return m.Called(ctx, jti).Error(0)
}
func (m *MockTokenRepository) DeleteAllForUser(ctx context.Context, userID uint64) error {
	return m.Called(ctx, userID).Error(0)
}

type MockOTPStore struct{ mock.Mock }

func (m *MockOTPStore) Set(ctx context.Context, phone, otp string, ttl time.Duration) error {
	return m.Called(ctx, phone, otp, ttl).Error(0)
}
func (m *MockOTPStore) Get(ctx context.Context, phone string) (string, error) {
	args := m.Called(ctx, phone)
	return args.String(0), args.Error(1)
}
func (m *MockOTPStore) Delete(ctx context.Context, phone string) error {
	return m.Called(ctx, phone).Error(0)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func testConfig() *config.Config {
	return &config.Config{
		JWTAccessSecret:  "test-access-secret-32-bytes-long!!",
		JWTRefreshSecret: "test-refresh-secret-32-bytes-longg",
		JWTAccessExpiry:  "15m",
		JWTRefreshExpiry: "168h",
		OTPTtl:           "5m",
		BcryptCost:       bcrypt.MinCost,
	}
}

func testUsecase(t *testing.T, ur *MockUserRepository, tr *MockTokenRepository, os *MockOTPStore) usecase.AuthUsecase {
	t.Helper()
	uc, err := usecase.NewAuthUsecase(ur, tr, os, testConfig(), logger.New())
	require.NoError(t, err)
	return uc
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.Phone == "+919876543210" &&
			u.Email == "alice@example.com" &&
			u.Password != "plainpass" && // must be bcrypt hash
			!u.IsActive
	})).Return(nil)

	assert.NoError(t, uc.Register(context.Background(), "+919876543210", "alice@example.com", "plainpass"))
	ur.AssertExpectations(t)
}

func TestRegister_DuplicateUser_ReturnsErrUserAlreadyExists(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("Create", mock.Anything, mock.Anything).Return(apperrors.ErrUserAlreadyExists)

	err := uc.Register(context.Background(), "+919876543210", "alice@example.com", "plainpass")
	assert.ErrorIs(t, err, apperrors.ErrUserAlreadyExists)
	ur.AssertExpectations(t)
}

// ─── SendOTP ─────────────────────────────────────────────────────────────────

func TestSendOTP_Success(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, Phone: "+919876543210", IsActive: false}, nil)
	os.On("Set", mock.Anything, "+919876543210",
		mock.MatchedBy(func(otp string) bool { return len(otp) == 6 }),
		5*time.Minute).Return(nil)

	assert.NoError(t, uc.SendOTP(context.Background(), "+919876543210"))
	ur.AssertExpectations(t)
	os.AssertExpectations(t)
}

func TestSendOTP_UserNotFound(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("FindByPhone", mock.Anything, "+919876543210").Return(nil, apperrors.ErrUserNotFound)

	err := uc.SendOTP(context.Background(), "+919876543210")
	assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	ur.AssertExpectations(t)
}

func TestSendOTP_UserAlreadyActive(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, IsActive: true}, nil)

	err := uc.SendOTP(context.Background(), "+919876543210")
	assert.ErrorIs(t, err, apperrors.ErrUserAlreadyActive)
	ur.AssertExpectations(t)
}

// ─── VerifyOTP ───────────────────────────────────────────────────────────────

func TestVerifyOTP_Success(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	os.On("Get", mock.Anything, "+919876543210").Return("123456", nil)
	os.On("Delete", mock.Anything, "+919876543210").Return(nil)
	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, Phone: "+919876543210"}, nil)
	ur.On("UpdateIsActive", mock.Anything, uint64(1), true).Return(nil)

	assert.NoError(t, uc.VerifyOTP(context.Background(), "+919876543210", "123456"))
	ur.AssertExpectations(t)
	os.AssertExpectations(t)
}

func TestVerifyOTP_OTPExpired(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	os.On("Get", mock.Anything, "+919876543210").Return("", apperrors.ErrOTPExpired)

	err := uc.VerifyOTP(context.Background(), "+919876543210", "123456")
	assert.ErrorIs(t, err, apperrors.ErrOTPExpired)
	os.AssertExpectations(t)
}

func TestVerifyOTP_OTPInvalid(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	os.On("Get", mock.Anything, "+919876543210").Return("654321", nil)

	err := uc.VerifyOTP(context.Background(), "+919876543210", "123456")
	assert.ErrorIs(t, err, apperrors.ErrOTPInvalid)
	os.AssertExpectations(t)
}

func TestSendOTP_StoreError_Propagates(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, IsActive: false}, nil)
	os.On("Set", mock.Anything, "+919876543210", mock.Anything, mock.Anything).
		Return(apperrors.ErrInternalServer)

	err := uc.SendOTP(context.Background(), "+919876543210")
	assert.Error(t, err)
	ur.AssertExpectations(t)
	os.AssertExpectations(t)
}

func TestVerifyOTP_DeleteError_Propagates(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	os.On("Get", mock.Anything, "+919876543210").Return("123456", nil)
	os.On("Delete", mock.Anything, "+919876543210").Return(apperrors.ErrInternalServer)

	err := uc.VerifyOTP(context.Background(), "+919876543210", "123456")
	assert.Error(t, err)
	os.AssertExpectations(t)
}

func TestVerifyOTP_FindByPhoneError_Propagates(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	os.On("Get", mock.Anything, "+919876543210").Return("123456", nil)
	os.On("Delete", mock.Anything, "+919876543210").Return(nil)
	ur.On("FindByPhone", mock.Anything, "+919876543210").Return(nil, apperrors.ErrUserNotFound)

	err := uc.VerifyOTP(context.Background(), "+919876543210", "123456")
	assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	os.AssertExpectations(t)
	ur.AssertExpectations(t)
}

func TestLogin_TokenStoreError_Propagates(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, IsActive: true, Password: string(hashed)}, nil)
	tr.On("Store", mock.Anything, mock.Anything).Return(apperrors.ErrInternalServer)

	_, err := uc.Login(context.Background(), "+919876543210", "password")
	assert.Error(t, err)
	tr.AssertExpectations(t)
}

// ─── NewAuthUsecase — bad config ─────────────────────────────────────────────

func TestNewAuthUsecase_BadAccessExpiry(t *testing.T) {
	cfg := testConfig()
	cfg.JWTAccessExpiry = "not-a-duration"
	_, err := usecase.NewAuthUsecase(&MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}, cfg, logger.New())
	assert.Error(t, err)
}

func TestNewAuthUsecase_BadRefreshExpiry(t *testing.T) {
	cfg := testConfig()
	cfg.JWTRefreshExpiry = "not-a-duration"
	_, err := usecase.NewAuthUsecase(&MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}, cfg, logger.New())
	assert.Error(t, err)
}

func TestNewAuthUsecase_BadOTPTTL(t *testing.T) {
	cfg := testConfig()
	cfg.OTPTtl = "not-a-duration"
	_, err := usecase.NewAuthUsecase(&MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}, cfg, logger.New())
	assert.Error(t, err)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, IsActive: true, Password: string(hashed)}, nil)
	tr.On("Store", mock.Anything, mock.Anything).Return(nil)

	pair, err := uc.Login(context.Background(), "+919876543210", "password")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	ur.AssertExpectations(t)
	tr.AssertExpectations(t)
}

func TestLogin_UserNotFound_ReturnsErrInvalidCredentials(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("FindByPhone", mock.Anything, "+919876543210").Return(nil, apperrors.ErrUserNotFound)

	_, err := uc.Login(context.Background(), "+919876543210", "password")
	// Must never reveal which field failed — ErrInvalidCredentials always.
	assert.ErrorIs(t, err, apperrors.ErrInvalidCredentials)
	ur.AssertExpectations(t)
}

func TestLogin_WrongPassword_ReturnsErrInvalidCredentials(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, IsActive: true, Password: string(hashed)}, nil)

	_, err := uc.Login(context.Background(), "+919876543210", "wrong")
	assert.ErrorIs(t, err, apperrors.ErrInvalidCredentials)
	ur.AssertExpectations(t)
}

func TestLogin_UserNotVerified(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	ur.On("FindByPhone", mock.Anything, "+919876543210").
		Return(&domain.User{ID: 1, IsActive: false, Password: "anything"}, nil)

	_, err := uc.Login(context.Background(), "+919876543210", "password")
	assert.ErrorIs(t, err, apperrors.ErrUserNotVerified)
	ur.AssertExpectations(t)
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	cfg := testConfig()
	refreshExpiry, _ := time.ParseDuration(cfg.JWTRefreshExpiry)
	refreshTok, _ := jwt.GenerateRefreshToken(42, cfg.JWTRefreshSecret, refreshExpiry)
	claims, _ := jwt.ParseToken(refreshTok, cfg.JWTRefreshSecret)

	tr.On("Find", mock.Anything, claims.JTI).
		Return(&domain.RefreshToken{JTI: claims.JTI, UserID: 42}, nil)
	tr.On("Delete", mock.Anything, claims.JTI).Return(nil)
	tr.On("Store", mock.Anything, mock.Anything).Return(nil)

	pair, err := uc.Refresh(context.Background(), refreshTok)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	tr.AssertExpectations(t)
}

func TestRefresh_InvalidToken_ReturnsErrInvalidToken(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	_, err := uc.Refresh(context.Background(), "not.a.valid.jwt")
	assert.ErrorIs(t, err, apperrors.ErrInvalidToken)
}

func TestRefresh_TokenRevoked_DeletesAllForUser(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	cfg := testConfig()
	refreshExpiry, _ := time.ParseDuration(cfg.JWTRefreshExpiry)
	refreshTok, _ := jwt.GenerateRefreshToken(42, cfg.JWTRefreshSecret, refreshExpiry)
	claims, _ := jwt.ParseToken(refreshTok, cfg.JWTRefreshSecret)

	tr.On("Find", mock.Anything, claims.JTI).Return(nil, apperrors.ErrTokenRevoked)
	tr.On("DeleteAllForUser", mock.Anything, uint64(42)).Return(nil)

	_, err := uc.Refresh(context.Background(), refreshTok)
	assert.ErrorIs(t, err, apperrors.ErrTokenRevoked)
	tr.AssertExpectations(t)
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestLogout_Success(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	tr.On("Delete", mock.Anything, "some-jti").Return(nil)

	assert.NoError(t, uc.Logout(context.Background(), "some-jti"))
	tr.AssertExpectations(t)
}

func TestLogout_RepoError_Propagates(t *testing.T) {
	ur, tr, os := &MockUserRepository{}, &MockTokenRepository{}, &MockOTPStore{}
	uc := testUsecase(t, ur, tr, os)

	tr.On("Delete", mock.Anything, "some-jti").Return(apperrors.ErrInternalServer)

	err := uc.Logout(context.Background(), "some-jti")
	assert.Error(t, err)
	tr.AssertExpectations(t)
}
