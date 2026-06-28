package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alokwritescode/digi-vault/auth-service/internal/repository"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
)

func TestOTPStore_SetAndGet_ReturnsStoredValue(t *testing.T) {
	client, _ := newTestRedis(t)
	store := repository.NewOTPStore(client)
	ctx := context.Background()

	assert.NoError(t, store.Set(ctx, "+911234567890", "123456", 5*time.Minute))

	otp, err := store.Get(ctx, "+911234567890")
	assert.NoError(t, err)
	assert.Equal(t, "123456", otp)
}

func TestOTPStore_Get_KeyMissing_ReturnsErrOTPExpired(t *testing.T) {
	client, _ := newTestRedis(t)
	store := repository.NewOTPStore(client)

	_, err := store.Get(context.Background(), "+911234567890")
	assert.ErrorIs(t, err, apperrors.ErrOTPExpired)
}

func TestOTPStore_Get_KeyExpired_ReturnsErrOTPExpired(t *testing.T) {
	client, mr := newTestRedis(t)
	store := repository.NewOTPStore(client)
	ctx := context.Background()

	assert.NoError(t, store.Set(ctx, "+911234567890", "123456", time.Second))
	mr.FastForward(2 * time.Second)

	_, err := store.Get(ctx, "+911234567890")
	assert.ErrorIs(t, err, apperrors.ErrOTPExpired)
}

func TestOTPStore_Delete_RemovesKey(t *testing.T) {
	client, _ := newTestRedis(t)
	store := repository.NewOTPStore(client)
	ctx := context.Background()

	assert.NoError(t, store.Set(ctx, "+911234567890", "123456", 5*time.Minute))
	assert.NoError(t, store.Delete(ctx, "+911234567890"))

	_, err := store.Get(ctx, "+911234567890")
	assert.ErrorIs(t, err, apperrors.ErrOTPExpired)
}

func TestOTPStore_Set_RedisError_ReturnsError(t *testing.T) {
	client, mr := newTestRedis(t)
	store := repository.NewOTPStore(client)
	mr.SetError("ERR internal")

	err := store.Set(context.Background(), "+91123", "000000", time.Minute)
	assert.Error(t, err)
}

func TestOTPStore_Get_RedisError_ReturnsWrappedError(t *testing.T) {
	client, mr := newTestRedis(t)
	store := repository.NewOTPStore(client)
	assert.NoError(t, mr.Set("otp:+91123", "111111"))
	mr.SetError("ERR internal")

	_, err := store.Get(context.Background(), "+91123")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, apperrors.ErrOTPExpired)
}
