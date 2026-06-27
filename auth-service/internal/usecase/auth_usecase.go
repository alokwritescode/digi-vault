package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/alokwritescode/digi-vault/auth-service/config"
	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/alokwritescode/digi-vault/auth-service/internal/repository"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/alokwritescode/digi-vault/shared/pkg/jwt"
)

// OTPStore is the contract for one-time password storage.
// Defined here (consumer side) so the repository package never imports usecase.
type OTPStore interface {
	Set(ctx context.Context, phone, otp string, ttl time.Duration) error
	Get(ctx context.Context, phone string) (string, error)
	Delete(ctx context.Context, phone string) error
}

// TokenPair holds an access + refresh token pair returned from Login and Refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AuthUsecase defines all authentication business operations.
type AuthUsecase interface {
	Register(ctx context.Context, phone, email, password string) error
	SendOTP(ctx context.Context, phone string) error
	VerifyOTP(ctx context.Context, phone, otp string) error
	Login(ctx context.Context, phone, password string) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, jti string) error
}

type authUsecase struct {
	userRepo         repository.UserRepository
	tokenRepo        repository.TokenRepository
	otpStore         OTPStore
	logger           *logrus.Logger
	jwtAccessSecret  string
	jwtRefreshSecret string
	jwtAccessExpiry  time.Duration
	jwtRefreshExpiry time.Duration
	otpTTL           time.Duration
	bcryptCost       int
}

// NewAuthUsecase constructs the usecase, parsing duration config strings eagerly
// so misconfiguration fails at startup, not at first request.
func NewAuthUsecase(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	otpStore OTPStore,
	cfg *config.Config,
	log *logrus.Logger,
) (AuthUsecase, error) {
	accessExpiry, err := time.ParseDuration(cfg.JWTAccessExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}
	refreshExpiry, err := time.ParseDuration(cfg.JWTRefreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}
	otpTTL, err := time.ParseDuration(cfg.OTPTtl)
	if err != nil {
		return nil, fmt.Errorf("invalid OTP_TTL: %w", err)
	}
	return &authUsecase{
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		otpStore:         otpStore,
		logger:           log,
		jwtAccessSecret:  cfg.JWTAccessSecret,
		jwtRefreshSecret: cfg.JWTRefreshSecret,
		jwtAccessExpiry:  accessExpiry,
		jwtRefreshExpiry: refreshExpiry,
		otpTTL:           otpTTL,
		bcryptCost:       cfg.BcryptCost,
	}, nil
}

func (u *authUsecase) Register(ctx context.Context, phone, email, password string) error {
	u.logger.WithContext(ctx).WithField("phone", phone).Info("usecase: Register")
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), u.bcryptCost)
	if err != nil {
		return fmt.Errorf("register: hash password: %w", err)
	}
	user := &domain.User{
		Phone:    phone,
		Email:    email,
		Password: string(hashed),
		IsActive: false,
	}
	if err := u.userRepo.Create(ctx, user); err != nil {
		return err
	}
	u.logger.WithContext(ctx).WithField("phone", phone).Info("usecase: Register: user created")
	return nil
}

func (u *authUsecase) SendOTP(ctx context.Context, phone string) error {
	u.logger.WithContext(ctx).WithField("phone", phone).Info("usecase: SendOTP")
	user, err := u.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		return err
	}
	if user.IsActive {
		return apperrors.ErrUserAlreadyActive
	}
	otp, err := generate6DigitOTP()
	if err != nil {
		return fmt.Errorf("send otp: generate: %w", err)
	}
	if err := u.otpStore.Set(ctx, phone, otp, u.otpTTL); err != nil {
		return fmt.Errorf("send otp: store: %w", err)
	}
	// In dev, log the OTP so it can be used without a real SMS gateway.
	u.logger.WithContext(ctx).WithField("otp", otp).Info("usecase: SendOTP: [DEV] otp generated")
	return nil
}

func (u *authUsecase) VerifyOTP(ctx context.Context, phone, otp string) error {
	u.logger.WithContext(ctx).WithField("phone", phone).Info("usecase: VerifyOTP")
	stored, err := u.otpStore.Get(ctx, phone)
	if err != nil {
		return err // ErrOTPExpired propagates
	}
	if stored != otp {
		return apperrors.ErrOTPInvalid
	}
	// Delete immediately — single-use token.
	if err := u.otpStore.Delete(ctx, phone); err != nil {
		return fmt.Errorf("verify otp: delete: %w", err)
	}
	user, err := u.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		return err
	}
	return u.userRepo.UpdateIsActive(ctx, user.ID, true)
}

func (u *authUsecase) Login(ctx context.Context, phone, password string) (*TokenPair, error) {
	u.logger.WithContext(ctx).WithField("phone", phone).Info("usecase: Login")
	user, err := u.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		// Never reveal which field failed — always ErrInvalidCredentials.
		return nil, apperrors.ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, apperrors.ErrUserNotVerified
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}
	return u.issueTokenPair(ctx, user.ID)
}

func (u *authUsecase) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	u.logger.WithContext(ctx).Info("usecase: Refresh")
	claims, err := jwt.ParseToken(refreshToken, u.jwtRefreshSecret)
	if err != nil {
		return nil, apperrors.ErrInvalidToken
	}
	_, err = u.tokenRepo.Find(ctx, claims.JTI)
	if err != nil {
		if errors.Is(err, apperrors.ErrTokenRevoked) {
			// Stolen token detected — nuke every session for this user.
			_ = u.tokenRepo.DeleteAllForUser(ctx, claims.UserID)
		}
		return nil, err
	}
	// Rotate: delete the old token before issuing new ones.
	if err := u.tokenRepo.Delete(ctx, claims.JTI); err != nil {
		return nil, err
	}
	return u.issueTokenPair(ctx, claims.UserID)
}

func (u *authUsecase) Logout(ctx context.Context, jti string) error {
	u.logger.WithContext(ctx).Info("usecase: Logout")
	return u.tokenRepo.Delete(ctx, jti)
}

// issueTokenPair generates a fresh access+refresh token pair and stores the
// refresh token in Redis.
func (u *authUsecase) issueTokenPair(ctx context.Context, userID uint64) (*TokenPair, error) {
	accessToken, err := jwt.GenerateAccessToken(userID, u.jwtAccessSecret, u.jwtAccessExpiry)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: access: %w", err)
	}
	refreshToken, err := jwt.GenerateRefreshToken(userID, u.jwtRefreshSecret, u.jwtRefreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: refresh: %w", err)
	}
	claims, err := jwt.ParseToken(refreshToken, u.jwtRefreshSecret)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: parse refresh: %w", err)
	}
	rt := &domain.RefreshToken{
		JTI:       claims.JTI,
		UserID:    userID,
		ExpiresAt: claims.ExpiresAt.Time,
	}
	if err := u.tokenRepo.Store(ctx, rt); err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func generate6DigitOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
