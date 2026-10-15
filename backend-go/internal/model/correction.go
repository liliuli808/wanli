package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// 纠错建议状态。
const (
	CorrectionStatusPending  = "PENDING"
	CorrectionStatusApproved = "APPROVED"
	CorrectionStatusRejected = "REJECTED"
)

// CorrectionProposal 对应 correction_proposals 表。
// SuggestedChanges 为校对副本的全量内容（ingredients + steps），审核通过后据此生成新版本。
type CorrectionProposal struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RecipeVersionID   uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProposerID        uuid.UUID       `gorm:"type:uuid;not null"`
	SuggestedChanges  json.RawMessage `gorm:"type:jsonb;not null"`
	Reason            string          `gorm:"type:varchar(255);not null"`
	RejectionReason   *string         `gorm:"type:varchar(255)"`
	Status            string          `gorm:"type:varchar(20);not null;default:PENDING"`
	ApprovedVersionID *uuid.UUID      `gorm:"type:uuid"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (CorrectionProposal) TableName() string {
	return "correction_proposals"
}
