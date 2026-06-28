package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all auth-service configuration loaded from environment or .env file.
type Config struct {
	AppPort  string
	AppEnv   string
	DBHost   string
	DBPort   string
	DBUser   string
	DBPassword string
	DBName   string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessExpiry  string
	JWTRefreshExpiry string
	OTPTtl     string
	BcryptCost int
	GRPCPort   string
}

// Load reads configuration from environment variables (and an optional .env in CWD).
func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	setDefaults(v)
	return build(v)
}

// LoadFromFile reads configuration from the specified file path.
func LoadFromFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	v.AutomaticEnv()
	setDefaults(v)
	return build(v)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("APP_PORT", "8081")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("DB_PORT", "3306")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	v.SetDefault("OTP_TTL", "5m")
	v.SetDefault("BCRYPT_COST", 12)
	v.SetDefault("GRPC_PORT", "9091")
}

func build(v *viper.Viper) (*Config, error) {
	required := []string{
		"APP_PORT", "DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"REDIS_HOST", "JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET",
	}
	for _, key := range required {
		if v.GetString(key) == "" {
			return nil, fmt.Errorf("missing required config: %s", key)
		}
	}

	return &Config{
		AppPort:          v.GetString("APP_PORT"),
		AppEnv:           v.GetString("APP_ENV"),
		DBHost:           v.GetString("DB_HOST"),
		DBPort:           v.GetString("DB_PORT"),
		DBUser:           v.GetString("DB_USER"),
		DBPassword:       v.GetString("DB_PASSWORD"),
		DBName:           v.GetString("DB_NAME"),
		RedisHost:        v.GetString("REDIS_HOST"),
		RedisPort:        v.GetString("REDIS_PORT"),
		RedisPassword:    v.GetString("REDIS_PASSWORD"),
		JWTAccessSecret:  v.GetString("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: v.GetString("JWT_REFRESH_SECRET"),
		JWTAccessExpiry:  v.GetString("JWT_ACCESS_EXPIRY"),
		JWTRefreshExpiry: v.GetString("JWT_REFRESH_EXPIRY"),
		OTPTtl:           v.GetString("OTP_TTL"),
		BcryptCost:       v.GetInt("BCRYPT_COST"),
		GRPCPort:         v.GetString("GRPC_PORT"),
	}, nil
}
