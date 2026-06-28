package domain

import (
	"fmt"
	"time"
)

// RefreshToken is a Redis-backed token — it has no DB table.
// Key format: refresh:{jti}
type RefreshToken struct {
	JTI       string    // UUID v4; used as the Redis key suffix
	UserID    uint64
	ExpiresAt time.Time
}

// RedisKey returns the canonical Redis key for this token.
func (r RefreshToken) RedisKey() string {
	return fmt.Sprintf("refresh:%s", r.JTI)
}

// IsExpired reports whether the token is past its expiry time.
func (r RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}
