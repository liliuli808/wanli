package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	CommunityStatusActive  = "ACTIVE"
	CommunityStatusDeleted = "DELETED"
)

type Comment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RecipeVersionID uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID          *uuid.UUID `gorm:"type:uuid"`
	Content         string     `gorm:"type:text;not null"`
	Status          string     `gorm:"type:varchar(20);not null;default:ACTIVE"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Comment) TableName() string {
	return "comments"
}

type CookingRecord struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RecipeVersionID uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"`
	Rating          *int
	PhotoURL        *string `gorm:"type:varchar(512)"`
	Feedback        *string `gorm:"type:text"`
	Status          string  `gorm:"type:varchar(20);not null;default:ACTIVE"`
	CreatedAt       time.Time
}

func (CookingRecord) TableName() string {
	return "cooking_records"
}
