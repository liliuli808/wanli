package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

type CommentListItem struct {
	ID             uuid.UUID
	UserID         *uuid.UUID
	Content        string
	AuthorNickname string
	CreatedAt      time.Time
}

type CookingRecordListItem struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Rating         *int
	PhotoURL       *string
	Feedback       *string
	AuthorNickname string
	CreatedAt      time.Time
}

// CookingFeedItem 是全站「做过」动态流的列表项（首页 Feed 用）。
type CookingFeedItem struct {
	ID              uuid.UUID
	Rating          *int
	PhotoURL        *string
	Feedback        *string
	AuthorNickname  string
	CreatedAt       time.Time
	RecipeVersionID uuid.UUID
	RecipeID        uuid.UUID
	RecipeTitle     string
	CoverURL        *string
}

// MyCookingRecordItem 是「我的做过」照片墙的列表项。
type MyCookingRecordItem struct {
	ID              uuid.UUID
	Rating          *int
	PhotoURL        *string
	Feedback        *string
	CreatedAt       time.Time
	RecipeVersionID uuid.UUID
	RecipeTitle     string
	CoverURL        *string
}

// UserVersionActivityItem 是发生在「我的版本」上的他人行为（评论/做过）。
type UserVersionActivityItem struct {
	ID              uuid.UUID
	ActorNickname   string
	Text            *string // 评论内容或做过反馈
	Rating          *int    // 仅做过有值
	CreatedAt       time.Time
	RecipeVersionID uuid.UUID
	RecipeTitle     string
}

type CommunityRepository interface {
	IsVersionPublicReadable(ctx context.Context, versionID uuid.UUID) (bool, error)
	CreateComment(ctx context.Context, comment *model.Comment) error
	ListComments(ctx context.Context, versionID uuid.UUID) ([]CommentListItem, error)
	SoftDeleteComment(ctx context.Context, userID, commentID uuid.UUID) (bool, error)
	CreateCookingRecord(ctx context.Context, record *model.CookingRecord) error
	ListCookingRecords(ctx context.Context, versionID uuid.UUID) ([]CookingRecordListItem, error)
	ListRecentCookingFeed(ctx context.Context, limit int) ([]CookingFeedItem, error)
	ListCookingRecordsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]MyCookingRecordItem, error)
	SoftDeleteCookingRecord(ctx context.Context, userID, recordID uuid.UUID) (bool, error)
	ListRecentCommentsOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]UserVersionActivityItem, error)
	ListRecentCookingOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]UserVersionActivityItem, error)
}

type GormCommunityRepository struct {
	db *gorm.DB
}

func NewGormCommunityRepository(db *gorm.DB) *GormCommunityRepository {
	return &GormCommunityRepository{db: db}
}

func (r *GormCommunityRepository) IsVersionPublicReadable(ctx context.Context, versionID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("recipe_versions AS version").
		Joins("JOIN recipes AS recipe ON recipe.id = version.recipe_id").
		Where(
			"version.id = ? AND version.is_public = TRUE AND version.status = ? AND recipe.status = ?",
			versionID, model.ContentStatusActive, model.ContentStatusActive,
		).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check public recipe version: %w", err)
	}
	return count > 0, nil
}

func (r *GormCommunityRepository) CreateComment(ctx context.Context, comment *model.Comment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *GormCommunityRepository) ListComments(ctx context.Context, versionID uuid.UUID) ([]CommentListItem, error) {
	var items []CommentListItem
	err := r.db.WithContext(ctx).
		Table("comments").
		Select(
			"comments.id, comments.user_id, comments.content, comments.created_at, "+
				"COALESCE(users.nickname, '已注销用户') AS author_nickname",
		).
		Joins("LEFT JOIN users ON users.id = comments.user_id").
		Where("comments.recipe_version_id = ? AND comments.status = ?", versionID, model.CommunityStatusActive).
		Order("comments.created_at DESC").
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return items, nil
}

func (r *GormCommunityRepository) SoftDeleteComment(
	ctx context.Context,
	userID, commentID uuid.UUID,
) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ? AND user_id = ? AND status = ?", commentID, userID, model.CommunityStatusActive).
		Update("status", model.CommunityStatusDeleted)
	if result.Error != nil {
		return false, fmt.Errorf("soft delete comment: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *GormCommunityRepository) CreateCookingRecord(ctx context.Context, record *model.CookingRecord) error {
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("create cooking record: %w", err)
	}
	return nil
}

func (r *GormCommunityRepository) ListCookingRecords(
	ctx context.Context,
	versionID uuid.UUID,
) ([]CookingRecordListItem, error) {
	var items []CookingRecordListItem
	err := r.db.WithContext(ctx).
		Table("cooking_records").
		Select(
			"cooking_records.id, cooking_records.user_id, cooking_records.rating, "+
				"cooking_records.photo_url, cooking_records.feedback, cooking_records.created_at, "+
				"users.nickname AS author_nickname",
		).
		Joins("JOIN users ON users.id = cooking_records.user_id").
		Where(
			"cooking_records.recipe_version_id = ? AND cooking_records.status = ?",
			versionID, model.CommunityStatusActive,
		).
		Order("cooking_records.created_at DESC").
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list cooking records: %w", err)
	}
	return items, nil
}

func (r *GormCommunityRepository) ListRecentCookingFeed(ctx context.Context, limit int) ([]CookingFeedItem, error) {
	items := make([]CookingFeedItem, 0)
	err := r.db.WithContext(ctx).
		Table("cooking_records AS cr").
		Select(`cr.id, cr.rating, cr.photo_url, cr.feedback, cr.created_at,
			COALESCE(u.nickname, '') AS author_nickname,
			v.id AS recipe_version_id, r.id AS recipe_id, v.title AS recipe_title,
			vs.cover_url AS cover_url`).
		Joins("JOIN users u ON u.id = cr.user_id").
		Joins("JOIN recipe_versions v ON v.id = cr.recipe_version_id AND v.is_public = TRUE AND v.status = ?", model.ContentStatusActive).
		Joins("JOIN recipes r ON r.id = v.recipe_id AND r.status = ?", model.ContentStatusActive).
		Joins("LEFT JOIN video_sources vs ON vs.id = r.video_source_id").
		Where("cr.status = ?", model.CommunityStatusActive).
		Order("cr.created_at DESC").
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list recent cooking feed: %w", err)
	}
	return items, nil
}

func (r *GormCommunityRepository) ListCookingRecordsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]MyCookingRecordItem, error) {
	items := make([]MyCookingRecordItem, 0)
	err := r.db.WithContext(ctx).
		Table("cooking_records AS cr").
		Select(`cr.id, cr.rating, cr.photo_url, cr.feedback, cr.created_at,
			v.id AS recipe_version_id, v.title AS recipe_title, vs.cover_url AS cover_url`).
		Joins("JOIN recipe_versions v ON v.id = cr.recipe_version_id AND v.status = ?", model.ContentStatusActive).
		Joins("JOIN recipes r ON r.id = v.recipe_id AND r.status = ?", model.ContentStatusActive).
		Joins("LEFT JOIN video_sources vs ON vs.id = r.video_source_id").
		Where("cr.user_id = ? AND cr.status = ?", userID, model.CommunityStatusActive).
		Order("cr.created_at DESC").
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list cooking records by user: %w", err)
	}
	return items, nil
}

func (r *GormCommunityRepository) ListRecentCommentsOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]UserVersionActivityItem, error) {
	items := make([]UserVersionActivityItem, 0)
	err := r.db.WithContext(ctx).
		Table("comments AS c").
		Select(`c.id, COALESCE(u.nickname, '') AS actor_nickname, c.content AS text,
			c.created_at, v.id AS recipe_version_id, v.title AS recipe_title`).
		Joins("JOIN recipe_versions v ON v.id = c.recipe_version_id AND v.author_id = ? AND v.status = ?", authorID, model.ContentStatusActive).
		Joins("LEFT JOIN users u ON u.id = c.user_id").
		Where("c.status = ? AND (c.user_id IS NULL OR c.user_id != ?)", model.CommunityStatusActive, authorID).
		Order("c.created_at DESC").
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list comments on user versions: %w", err)
	}
	return items, nil
}

func (r *GormCommunityRepository) ListRecentCookingOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]UserVersionActivityItem, error) {
	items := make([]UserVersionActivityItem, 0)
	err := r.db.WithContext(ctx).
		Table("cooking_records AS cr").
		Select(`cr.id, COALESCE(u.nickname, '') AS actor_nickname, cr.feedback AS text,
			cr.rating, cr.created_at, v.id AS recipe_version_id, v.title AS recipe_title`).
		Joins("JOIN recipe_versions v ON v.id = cr.recipe_version_id AND v.author_id = ? AND v.status = ?", authorID, model.ContentStatusActive).
		Joins("JOIN users u ON u.id = cr.user_id").
		Where("cr.status = ? AND cr.user_id != ?", model.CommunityStatusActive, authorID).
		Order("cr.created_at DESC").
		Limit(limit).
		Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list cooking records on user versions: %w", err)
	}
	return items, nil
}

func (r *GormCommunityRepository) SoftDeleteCookingRecord(
	ctx context.Context,
	userID, recordID uuid.UUID,
) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.CookingRecord{}).
		Where("id = ? AND user_id = ? AND status = ?", recordID, userID, model.CommunityStatusActive).
		Update("status", model.CommunityStatusDeleted)
	if result.Error != nil {
		return false, fmt.Errorf("soft delete cooking record: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}
