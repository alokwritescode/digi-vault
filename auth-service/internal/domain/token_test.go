package domain_test

import (
	"testing"
	"time"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestRefreshToken_Fields(t *testing.T) {
	exp := time.Now().Add(7 * 24 * time.Hour)
	rt := domain.RefreshToken{
		JTI:       "550e8400-e29b-41d4-a716-446655440000",
		UserID:    42,
		ExpiresAt: exp,
	}

	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", rt.JTI)
	assert.Equal(t, uint64(42), rt.UserID)
	assert.Equal(t, exp, rt.ExpiresAt)
}

func TestRefreshToken_RedisKey(t *testing.T) {
	rt := domain.RefreshToken{JTI: "abc-123"}
	assert.Equal(t, "refresh:abc-123", rt.RedisKey())
}

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		expiry  time.Time
		expired bool
	}{
		{"future", time.Now().Add(time.Hour), false},
		{"past", time.Now().Add(-time.Second), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := domain.RefreshToken{ExpiresAt: tt.expiry}
			assert.Equal(t, tt.expired, rt.IsExpired())
		})
	}
}
