package repository_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/alokwritescode/digi-vault/auth-service/internal/repository"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRedis spins up an in-memory Redis via miniredis.
// No real Redis server is needed — miniredis responds to the go-redis client
// identically to a real server.
func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t) // auto-closes when t finishes
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, mr
}

// ─── Store ───────────────────────────────────────────────────────────────────

func TestTokenStore_SetsRefreshKeyWithTTL(t *testing.T) {
	client, mr := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	token := &domain.RefreshToken{
		JTI:       "abc-123",
		UserID:    42,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	require.NoError(t, repo.Store(context.Background(), token))

	val, err := mr.Get("refresh:abc-123")
	require.NoError(t, err, "refresh key must exist after Store")
	assert.Equal(t, strconv.FormatUint(42, 10), val)
	assert.Greater(t, mr.TTL("refresh:abc-123"), time.Duration(0), "TTL must be positive")
}

func TestTokenStore_AddsJTIToUserSet(t *testing.T) {
	client, mr := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	token := &domain.RefreshToken{
		JTI:       "abc-123",
		UserID:    42,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	require.NoError(t, repo.Store(context.Background(), token))

	members, err := mr.SMembers("user_tokens:42")
	require.NoError(t, err)
	assert.Contains(t, members, "abc-123", "jti must be tracked in user_tokens set")
}

func TestTokenStore_ExpiredToken_ReturnsErrTokenExpired(t *testing.T) {
	client, _ := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	token := &domain.RefreshToken{
		JTI:       "expired-jti",
		UserID:    1,
		ExpiresAt: time.Now().Add(-time.Second), // already in the past
	}

	err := repo.Store(context.Background(), token)
	assert.ErrorIs(t, err, apperrors.ErrTokenExpired)
}

// ─── Find ────────────────────────────────────────────────────────────────────

func TestTokenFind_ReturnsCorrectUserID(t *testing.T) {
	client, _ := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	token := &domain.RefreshToken{
		JTI:       "find-jti-456",
		UserID:    99,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Store(context.Background(), token))

	got, err := repo.Find(context.Background(), "find-jti-456")
	require.NoError(t, err)
	assert.Equal(t, uint64(99), got.UserID)
	assert.Equal(t, "find-jti-456", got.JTI)
}

func TestTokenFind_MissingKey_ReturnsErrTokenRevoked(t *testing.T) {
	client, _ := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	_, err := repo.Find(context.Background(), "nonexistent-jti")
	assert.ErrorIs(t, err, apperrors.ErrTokenRevoked)
}

// ─── Delete ──────────────────────────────────────────────────────────────────

func TestTokenDelete_RemovesRefreshKey(t *testing.T) {
	client, mr := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	token := &domain.RefreshToken{
		JTI:       "delete-jti-789",
		UserID:    10,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Store(context.Background(), token))

	require.NoError(t, repo.Delete(context.Background(), "delete-jti-789"))

	_, err := mr.Get("refresh:delete-jti-789")
	assert.Error(t, err, "refresh key must not exist after Delete")
}

func TestTokenDelete_NonExistentKey_NoError(t *testing.T) {
	client, _ := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	// DEL on a missing key is idempotent in Redis — must not error
	err := repo.Delete(context.Background(), "ghost-jti")
	assert.NoError(t, err)
}

// ─── DeleteAllForUser ─────────────────────────────────────────────────────────

func TestTokenDeleteAllForUser_RemovesAllTokensAndSet(t *testing.T) {
	client, mr := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	// Store two tokens for the same user
	for _, jti := range []string{"tok-a", "tok-b"} {
		require.NoError(t, repo.Store(context.Background(), &domain.RefreshToken{
			JTI: jti, UserID: 77, ExpiresAt: time.Now().Add(time.Hour),
		}))
	}

	require.NoError(t, repo.DeleteAllForUser(context.Background(), 77))

	// Both individual refresh keys must be gone
	_, errA := mr.Get("refresh:tok-a")
	_, errB := mr.Get("refresh:tok-b")
	assert.Error(t, errA, "refresh:tok-a must be deleted")
	assert.Error(t, errB, "refresh:tok-b must be deleted")

	// The user_tokens set must also be gone
	members, _ := mr.SMembers("user_tokens:77")
	assert.Empty(t, members)
}

func TestTokenDeleteAllForUser_NoTokens_NoError(t *testing.T) {
	client, _ := newTestRedis(t)
	repo := repository.NewTokenRepository(client)

	// user 999 has no tokens — should not error
	err := repo.DeleteAllForUser(context.Background(), 999)
	assert.NoError(t, err)
}
