package model

import (
	"time"

	"github.com/google/uuid"
)

// User 对应 users 表。
type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email          string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	HashedPassword string    `gorm:"type:varchar(255);not null"`
	Nickname       string    `gorm:"type:varchar(50);not null"`
	AvatarURL      *string   `gorm:"type:varchar(512)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (User) TableName() string {
	return "users"
}
