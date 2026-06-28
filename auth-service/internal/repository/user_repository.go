package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
)

// UserRepository defines the contract for user persistence.
// Every caller depends on this interface — never the concrete type.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uint64) (*domain.User, error)
	UpdateIsActive(ctx context.Context, userID uint64, isActive bool) error
	SoftDelete(ctx context.Context, userID uint64) error
}

type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository returns a UserRepository backed by GORM + MySQL.
// Returns the interface so callers are never coupled to the concrete type.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) Create(ctx context.Context, user *domain.User) error {
	return translateUserError(r.db.WithContext(ctx).Create(user).Error)
}

func (r *gormUserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	var user domain.User
	if err := translateUserError(r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := translateUserError(r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var user domain.User
	if err := translateUserError(r.db.WithContext(ctx).First(&user, id).Error); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) UpdateIsActive(ctx context.Context, userID uint64, isActive bool) error {
	return translateUserError(
		r.db.WithContext(ctx).
			Model(&domain.User{}).
			Where("id = ?", userID).
			Update("is_active", isActive).Error,
	)
}

// SoftDelete sets deleted_at to now via a manual UPDATE.
// We use Update() explicitly because domain.User.DeletedAt is *time.Time (not gorm.DeletedAt),
// so db.Delete() would issue a hard DELETE instead of setting the timestamp.
func (r *gormUserRepository) SoftDelete(ctx context.Context, userID uint64) error {
	return translateUserError(
		r.db.WithContext(ctx).
			Model(&domain.User{}).
			Where("id = ?", userID).
			Update("deleted_at", time.Now()).Error,
	)
}

// translateUserError is the two-layer boundary translation:
// DB driver error (MySQL 1062) → domain sentinel
// GORM error (ErrRecordNotFound) → domain sentinel
// Anything else is wrapped and bubbled up.
func translateUserError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrUserNotFound
	}
	if strings.Contains(err.Error(), "1062") {
		return apperrors.ErrUserAlreadyExists
	}
	return fmt.Errorf("repository: %w", err)
}
