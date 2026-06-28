package config_test

import (
	"os"
	"testing"

	"github.com/alokwritescode/digi-vault/auth-service/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FromEnvVars(t *testing.T) {
	t.Setenv("APP_PORT", "8081")
	t.Setenv("APP_ENV", "test")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USER", "root")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "dv_auth")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("JWT_ACCESS_SECRET", "access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Setenv("JWT_ACCESS_EXPIRY", "15m")
	t.Setenv("JWT_REFRESH_EXPIRY", "168h")
	t.Setenv("OTP_TTL", "5m")
	t.Setenv("BCRYPT_COST", "12")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "test", cfg.AppEnv)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "3306", cfg.DBPort)
	assert.Equal(t, "root", cfg.DBUser)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "dv_auth", cfg.DBName)
	assert.Equal(t, "localhost", cfg.RedisHost)
	assert.Equal(t, "6379", cfg.RedisPort)
	assert.Equal(t, "access-secret", cfg.JWTAccessSecret)
	assert.Equal(t, "refresh-secret", cfg.JWTRefreshSecret)
	assert.Equal(t, "15m", cfg.JWTAccessExpiry)
	assert.Equal(t, "168h", cfg.JWTRefreshExpiry)
	assert.Equal(t, "5m", cfg.OTPTtl)
	assert.Equal(t, 12, cfg.BcryptCost)
}

func TestLoad_FromEnvFile(t *testing.T) {
	// Write a temp .env file and point Viper at it.
	envContent := `APP_PORT=9090
					APP_ENV=staging
					DB_HOST=db.internal
					DB_PORT=3306
					DB_USER=admin
					DB_PASSWORD=dbpass
					DB_NAME=dv_auth
					REDIS_HOST=redis.internal
					REDIS_PORT=6379
					REDIS_PASSWORD=redispass
					JWT_ACCESS_SECRET=acc-sec
					JWT_REFRESH_SECRET=ref-sec
					JWT_ACCESS_EXPIRY=15m
					JWT_REFRESH_EXPIRY=168h
					OTP_TTL=5m
					BCRYPT_COST=10`
					
	f, err := os.CreateTemp(t.TempDir(), ".env")
	require.NoError(t, err)
	_, err = f.WriteString(envContent)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cfg, err := config.LoadFromFile(f.Name())
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, "admin", cfg.DBUser)
	assert.Equal(t, "redispass", cfg.RedisPassword)
	assert.Equal(t, 10, cfg.BcryptCost)
}

func TestLoad_MissingRequired_Fails(t *testing.T) {
	// Unset required fields — should return an error.
	for _, k := range []string{
		"APP_PORT", "DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"REDIS_HOST", "JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET",
	} {
		t.Setenv(k, "")
	}

	_, err := config.Load()
	assert.Error(t, err)
}
