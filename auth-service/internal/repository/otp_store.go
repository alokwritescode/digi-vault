package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
)

type redisOTPStore struct {
	client *redis.Client
}

// NewOTPStore returns a Redis-backed OTP store.
// The returned concrete type implicitly satisfies usecase.OTPStore.
func NewOTPStore(client *redis.Client) *redisOTPStore {
	return &redisOTPStore{client: client}
}

func (s *redisOTPStore) Set(ctx context.Context, phone, otp string, ttl time.Duration) error {
	if err := s.client.Set(ctx, "otp:"+phone, otp, ttl).Err(); err != nil {
		return fmt.Errorf("otp set: %w", err)
	}
	return nil
}

func (s *redisOTPStore) Get(ctx context.Context, phone string) (string, error) {
	val, err := s.client.Get(ctx, "otp:"+phone).Result()
	if errors.Is(err, redis.Nil) {
		return "", apperrors.ErrOTPExpired
	}
	if err != nil {
		return "", fmt.Errorf("otp get: %w", err)
	}
	return val, nil
}

func (s *redisOTPStore) Delete(ctx context.Context, phone string) error {
	return s.client.Del(ctx, "otp:"+phone).Err()
}
