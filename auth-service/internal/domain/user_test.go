package domain_test

import (
	"testing"
	"time"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestUser_Fields(t *testing.T) {
	now := time.Now()
	u := domain.User{
		ID:        1,
		Phone:     "+919876543210",
		Email:     "alice@example.com",
		Password:  "$2a$12$hashedpassword",
		IsActive:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, uint64(1), u.ID)
	assert.Equal(t, "+919876543210", u.Phone)
	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, "$2a$12$hashedpassword", u.Password)
	assert.False(t, u.IsActive)
	assert.Nil(t, u.DeletedAt, "soft delete field should be nil when not deleted")
}

func TestUser_SoftDelete(t *testing.T) {
	now := time.Now()
	u := domain.User{DeletedAt: &now}
	assert.NotNil(t, u.DeletedAt)
}

func TestUser_TableName(t *testing.T) {
	u := domain.User{}
	assert.Equal(t, "users", u.TableName())
}

func TestUser_NoGormModelEmbedding(t *testing.T) {
	// Verify User has explicit field declarations matching the DB schema.
	// If gorm.Model were embedded, the ID field type would be uint (not uint64).
	u := domain.User{ID: 1<<32 + 5}
	assert.Equal(t, uint64(1<<32+5), u.ID, "ID must be uint64 to match BIGINT UNSIGNED")
}
