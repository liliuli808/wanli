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

	// 用户授权的 B 站登录态；SESSDATA 加密存储，永不通过 API 返回。
	BilibiliSessdataEnc         *string    `gorm:"column:bilibili_sessdata_enc;type:text" json:"-"`
	BilibiliMid                 *string    `gorm:"column:bilibili_mid;type:varchar(32)" json:"-"`
	BilibiliUname               *string    `gorm:"column:bilibili_uname;type:varchar(100)" json:"-"`
	BilibiliCredentialUpdatedAt *time.Time `gorm:"column:bilibili_credential_updated_at" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// BilibiliCredential 是用户授权的 B 站凭证（仅后端使用，含加密后的 SESSDATA）。
type BilibiliCredential struct {
	EncSessdata string
	Mid         string
	Uname       string
	UpdatedAt   *time.Time
}
