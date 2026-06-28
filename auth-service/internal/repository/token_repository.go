package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
)

// TokenRepository defines the contract for refresh token storage in Redis.
// No DB table — every operation hits Redis only.
type TokenRepository interface {
	Store(ctx context.Context, token *domain.RefreshToken) error
	Find(ctx context.Context, jti string) (*domain.RefreshToken, error)
	Delete(ctx context.Context, jti string) error
	DeleteAllForUser(ctx context.Context, userID uint64) error
}

type redisTokenRepository struct {
	client *redis.Client
}

// NewTokenRepository returns a TokenRepository backed by Redis.
func NewTokenRepository(client *redis.Client) TokenRepository {
	return &redisTokenRepository{client: client}
}

// Store writes the token to Redis with a TTL derived from token.ExpiresAt.
// Also adds the JTI to a per-user set for stolen-token detection.
func (r *redisTokenRepository) Store(ctx context.Context, token *domain.RefreshToken) error {
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return apperrors.ErrTokenExpired
	}

	userIDStr := strconv.FormatUint(token.UserID, 10)
	if err := r.client.Set(ctx, token.RedisKey(), userIDStr, ttl).Err(); err != nil {
		return fmt.Errorf("token store: %w", err)
	}

	userKey := fmt.Sprintf("user_tokens:%d", token.UserID)
	if err := r.client.SAdd(ctx, userKey, token.JTI).Err(); err != nil {
		return fmt.Errorf("token store user set: %w", err)
	}

	return nil
}

// Find retrieves a token by JTI. Returns ErrTokenRevoked if the key is missing
// (expired by Redis TTL or already deleted — both mean stolen-token territory).
func (r *redisTokenRepository) Find(ctx context.Context, jti string) (*domain.RefreshToken, error) {
	key := fmt.Sprintf("refresh:%s", jti)
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, apperrors.ErrTokenRevoked
	}
	if err != nil {
		return nil, fmt.Errorf("token find: %w", err)
	}

	userID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("token find: corrupt user id: %w", err)
	}

	return &domain.RefreshToken{JTI: jti, UserID: userID}, nil
}

// Delete removes a single refresh key. Idempotent — no error if key is absent.
func (r *redisTokenRepository) Delete(ctx context.Context, jti string) error {
	key := fmt.Sprintf("refresh:%s", jti)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("token delete: %w", err)
	}
	return nil
}

// DeleteAllForUser removes every refresh token for a user and the tracking set.
// Called during stolen-token detection so the attacker's session is fully invalidated.
func (r *redisTokenRepository) DeleteAllForUser(ctx context.Context, userID uint64) error {
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	jtis, err := r.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("token delete all: %w", err)
	}

	keys := make([]string, 0, len(jtis)+1)
	for _, jti := range jtis {
		keys = append(keys, fmt.Sprintf("refresh:%s", jti))
	}
	keys = append(keys, userKey)

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("token delete all: %w", err)
	}
	return nil
}
