package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
)

// Claims holds the custom JWT payload.
type Claims struct {
	UserID uint64 `json:"user_id"`
	JTI    string `json:"jti"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed access JWT for the given user.
func GenerateAccessToken(userID uint64, secret string, expiry time.Duration) (string, error) {
	return generate(userID, secret, expiry)
}

// GenerateRefreshToken creates a signed refresh JWT for the given user.
func GenerateRefreshToken(userID uint64, secret string, expiry time.Duration) (string, error) {
	return generate(userID, secret, expiry)
}

func generate(userID uint64, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		JTI:    uuid.NewString(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseToken validates the token signature and expiry, returning the claims.
func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", apperrors.ErrInvalidToken, t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidToken, err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, apperrors.ErrInvalidToken
	}
	return claims, nil
}

// ExtractBearerToken strips the "Bearer " prefix from an Authorization header value.
func ExtractBearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", apperrors.ErrMissingToken
	}
	token := strings.TrimPrefix(header, prefix)
	if token == "" {
		return "", apperrors.ErrMissingToken
	}
	return token, nil
}
