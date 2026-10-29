package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

// UserRepository 用户数据访问接口。
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	// UpdateProfile 更新用户昵称与头像，返回更新后的用户；用户不存在时返回 (nil, nil)。
	UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error)
}

// GormUserRepository 基于 GORM 的实现。
type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *GormUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *GormUserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return count > 0, nil
}

// GetBilibiliCredential 读取用户授权的 B 站凭证；未授权时返回 (nil, nil)。
func (r *GormUserRepository) GetBilibiliCredential(ctx context.Context, id uuid.UUID) (*model.BilibiliCredential, error) {
	var row struct {
		Enc  *string
		Mid  *string
		Name *string
		At   *time.Time
	}
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("bilibili_sessdata_enc AS enc, bilibili_mid AS mid, bilibili_uname AS name, bilibili_credential_updated_at AS at").
		Where("id = ?", id).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bilibili credential: %w", err)
	}
	if row.Enc == nil || *row.Enc == "" {
		return nil, nil
	}
	cred := &model.BilibiliCredential{EncSessdata: *row.Enc, UpdatedAt: row.At}
	if row.Mid != nil {
		cred.Mid = *row.Mid
	}
	if row.Name != nil {
		cred.Uname = *row.Name
	}
	return cred, nil
}

// SetBilibiliCredential 保存（覆盖）用户授权的 B 站凭证。
func (r *GormUserRepository) SetBilibiliCredential(ctx context.Context, id uuid.UUID, encSessdata, mid, uname string) error {
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"bilibili_sessdata_enc":          encSessdata,
			"bilibili_mid":                   mid,
			"bilibili_uname":                 uname,
			"bilibili_credential_updated_at": time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("set bilibili credential: %w", err)
	}
	return nil
}

// ClearBilibiliCredential 清除用户授权的 B 站凭证。
func (r *GormUserRepository) ClearBilibiliCredential(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"bilibili_sessdata_enc":          nil,
			"bilibili_mid":                   nil,
			"bilibili_uname":                 nil,
			"bilibili_credential_updated_at": nil,
		}).Error
	if err != nil {
		return fmt.Errorf("clear bilibili credential: %w", err)
	}
	return nil
}

func (r *GormUserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	user.Nickname = nickname
	user.AvatarURL = avatarURL
	if err := r.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return &user, nil
}
