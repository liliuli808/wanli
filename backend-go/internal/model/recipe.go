package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// 视频来源状态。
const (
	VideoStatusActive  = "ACTIVE"  // 正常
	VideoStatusExpired = "EXPIRED" // 来源失效/视频下架
)

// 菜谱版本类型。
const (
	VersionTypeSource     = "SOURCE"      // 来源解析版
	VersionTypeAdapted    = "ADAPTED"     // 社区改编版
	VersionTypeKitchenFit = "KITCHEN_FIT" // 个人适配版
)

// 内容状态（recipes / recipe_versions 通用）。
const (
	ContentStatusActive  = "ACTIVE"
	ContentStatusBlocked = "BLOCKED"
)

// VideoSource 对应 video_sources 表，以 BVID 为唯一索引。
type VideoSource struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BVID       string    `gorm:"column:bvid;type:varchar(50);uniqueIndex;not null"`
	Title      string    `gorm:"type:varchar(255);not null"`
	UpNickname string    `gorm:"type:varchar(100);not null"`
	VideoURL   string    `gorm:"type:varchar(512);not null"`
	CoverURL   *string   `gorm:"type:varchar(512)"`
	Status     string    `gorm:"type:varchar(20);not null;default:ACTIVE"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (VideoSource) TableName() string {
	return "video_sources"
}

// Recipe 对应 recipes 表，表示一道菜的根身份。
type Recipe struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title         string     `gorm:"type:varchar(100);not null"`
	VideoSourceID *uuid.UUID `gorm:"type:uuid"`
	CreatorID     *uuid.UUID `gorm:"type:uuid"`
	Status        string     `gorm:"type:varchar(20);not null;default:ACTIVE"`
	CreatedAt     time.Time
}

func (Recipe) TableName() string {
	return "recipes"
}

// RecipeVersion 对应 recipe_versions 表。
// 草稿即 is_public = FALSE 的 SOURCE 版本，发布（阶段 3）时置为 TRUE。
type RecipeVersion struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RecipeID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	ParentID         *uuid.UUID `gorm:"type:uuid"`
	AuthorID         *uuid.UUID `gorm:"type:uuid"`
	VersionType      string     `gorm:"type:varchar(20);not null"`
	Title            string     `gorm:"type:varchar(100);not null"`
	IsPublic         bool       `gorm:"not null;default:false"`
	Status           string     `gorm:"type:varchar(20);not null;default:ACTIVE"`
	Servings         *string    `gorm:"type:varchar(50)"`
	EstimatedMinutes *int
	Instructions     *string         `gorm:"type:text"`
	ChangeLog        *string         `gorm:"type:varchar(255)"`
	AdaptationNotes  json.RawMessage `gorm:"type:jsonb"`
	CreatedAt        time.Time
	Ingredients      []RecipeIngredient `gorm:"foreignKey:RecipeVersionID"`
	Steps            []RecipeStep       `gorm:"foreignKey:RecipeVersionID"`
}

func (RecipeVersion) TableName() string {
	return "recipe_versions"
}

// RecipeIngredient 对应 recipe_ingredients 表。
type RecipeIngredient struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RecipeVersionID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name            string    `gorm:"type:varchar(100);not null"`
	Amount          *string   `gorm:"type:varchar(50)"`
	Category        string    `gorm:"type:varchar(20);not null"`
	IsExplicit      bool      `gorm:"not null;default:true"`
	SubstituteFor   *string   `gorm:"type:varchar(100)"`
}

func (RecipeIngredient) TableName() string {
	return "recipe_ingredients"
}

// RecipeStep 对应 recipe_steps 表。
type RecipeStep struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RecipeVersionID uuid.UUID `gorm:"type:uuid;not null;index"`
	StepOrder       int       `gorm:"not null"`
	Description     string    `gorm:"type:text;not null"`
	VideoTimestamp  *int
	DurationSeconds *int
	Heat            *string `gorm:"type:varchar(50)"`
}

func (RecipeStep) TableName() string {
	return "recipe_steps"
}
