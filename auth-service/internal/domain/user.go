package domain

import "time"

// User represents an authenticated principal stored in dv_auth.users.
// Fields are declared explicitly — gorm.Model is NOT embedded.
type User struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Phone     string     `gorm:"column:phone;uniqueIndex;not null"`
	Email     string     `gorm:"column:email;uniqueIndex;not null"`
	Password  string     `gorm:"column:password;not null"`
	IsActive  bool       `gorm:"column:is_active;not null;default:false"`
	CreatedAt time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

// TableName tells GORM which table to use.
func (User) TableName() string { return "users" }
