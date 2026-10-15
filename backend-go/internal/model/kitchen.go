package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	RestrictionTypeAllergy = "ALLERGY"
	RestrictionTypeDislike = "DISLIKE"
	RestrictionTypeMedical = "MEDICAL"
)

// KitchenProfile 对应 kitchen_profiles 表。
type KitchenProfile struct {
	UserID           uuid.UUID `gorm:"type:uuid;primary_key"`
	OwnedIngredients []string  `gorm:"type:jsonb;serializer:json"`
	OwnedTools       []string  `gorm:"type:jsonb;serializer:json"`
	UpdatedAt        time.Time
}

func (KitchenProfile) TableName() string {
	return "kitchen_profiles"
}

// DietaryRestriction 对应 dietary_restrictions 表。
type DietaryRestriction struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"`
	RestrictionType string    `gorm:"type:varchar(20);not null"`
	IngredientName  string    `gorm:"type:varchar(100);not null"`
	CreatedAt       time.Time
}

func (DietaryRestriction) TableName() string {
	return "dietary_restrictions"
}
