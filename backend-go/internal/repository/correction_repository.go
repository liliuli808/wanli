package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/wanli/backend-go/internal/model"
)

var ErrCorrectionAlreadyReviewed = errors.New("correction proposal already reviewed")

// PendingProposalItem 是待审核纠错建议的列表项（附菜谱标题与提议人昵称）。
type PendingProposalItem struct {
	Proposal         *model.CorrectionProposal
	RecipeTitle      string
	ProposerNickname string
}

// ReviewedProposalItem 是我提交的、已被审核的纠错建议。
type ReviewedProposalItem struct {
	Proposal    model.CorrectionProposal
	RecipeTitle string
}

// CorrectionRepository 纠错建议的数据访问接口。
type CorrectionRepository interface {
	// CreateProposal 创建纠错建议。
	CreateProposal(ctx context.Context, proposal *model.CorrectionProposal) error
	// FindProposalByID 查询纠错建议；不存在时返回 (nil, nil)。
	FindProposalByID(ctx context.Context, id uuid.UUID) (*model.CorrectionProposal, error)
	// RejectProposal 将待审核建议标记为已拒绝。
	RejectProposal(ctx context.Context, id uuid.UUID, rejectionReason string) error
	// ApproveWithNewVersion 在一个事务中生成纠错合入的新版本并标记建议已通过。
	ApproveWithNewVersion(ctx context.Context, proposalID uuid.UUID, params DerivedVersionParams) (*model.RecipeVersion, error)
	// ListPendingByCreator 查询来源发布者名下所有待审核的纠错建议（按提交时间升序）。
	ListPendingByCreator(ctx context.Context, creatorID uuid.UUID) ([]PendingProposalItem, error)
	// ListReviewedByProposer 查询提议人提交的、已出审核结果（APPROVED/REJECTED）的建议。
	ListReviewedByProposer(ctx context.Context, proposerID uuid.UUID, limit int) ([]ReviewedProposalItem, error)
}

// GormCorrectionRepository 基于 GORM 的实现。
type GormCorrectionRepository struct {
	db *gorm.DB
}

// NewGormCorrectionRepository 创建纠错建议仓库。
func NewGormCorrectionRepository(db *gorm.DB) *GormCorrectionRepository {
	return &GormCorrectionRepository{db: db}
}

func (r *GormCorrectionRepository) CreateProposal(ctx context.Context, proposal *model.CorrectionProposal) error {
	if err := r.db.WithContext(ctx).Create(proposal).Error; err != nil {
		return fmt.Errorf("create correction proposal: %w", err)
	}
	return nil
}

func (r *GormCorrectionRepository) FindProposalByID(ctx context.Context, id uuid.UUID) (*model.CorrectionProposal, error) {
	var proposal model.CorrectionProposal
	err := r.db.WithContext(ctx).First(&proposal, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find correction proposal: %w", err)
	}
	return &proposal, nil
}

func (r *GormCorrectionRepository) RejectProposal(ctx context.Context, id uuid.UUID, rejectionReason string) error {
	result := r.db.WithContext(ctx).Model(&model.CorrectionProposal{}).
		Where("id = ? AND status = ?", id, model.CorrectionStatusPending).
		Updates(map[string]any{
			"status":           model.CorrectionStatusRejected,
			"rejection_reason": rejectionReason,
			"updated_at":       gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if result.Error != nil {
		return fmt.Errorf("reject correction proposal: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: proposal %s", ErrCorrectionAlreadyReviewed, id)
	}
	return nil
}

func (r *GormCorrectionRepository) ApproveWithNewVersion(ctx context.Context, proposalID uuid.UUID, params DerivedVersionParams) (*model.RecipeVersion, error) {
	var version *model.RecipeVersion
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		version, err = createDerivedVersionTx(tx, params)
		if err != nil {
			return err
		}
		if params.VersionType == model.VersionTypeSource {
			if err := tx.Model(&model.Recipe{}).
				Where("id = ?", params.RecipeID).
				Update("title", params.Parsed.Title).Error; err != nil {
				return fmt.Errorf("update canonical recipe title: %w", err)
			}
		}

		result := tx.Model(&model.CorrectionProposal{}).
			Where("id = ? AND status = ?", proposalID, model.CorrectionStatusPending).
			Updates(map[string]any{
				"status":              model.CorrectionStatusApproved,
				"approved_version_id": version.ID,
				"updated_at":          gorm.Expr("CURRENT_TIMESTAMP"),
			})
		if result.Error != nil {
			return fmt.Errorf("approve correction proposal: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("%w: proposal %s", ErrCorrectionAlreadyReviewed, proposalID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

// pendingProposalRow 是待审核列表查询的扫描行。
type pendingProposalRow struct {
	model.CorrectionProposal `gorm:"embedded"`
	RecipeTitle              string `gorm:"column:recipe_title"`
	ProposerNickname         string `gorm:"column:proposer_nickname"`
}

func (r *GormCorrectionRepository) ListPendingByCreator(ctx context.Context, creatorID uuid.UUID) ([]PendingProposalItem, error) {
	var rows []pendingProposalRow
	err := r.db.WithContext(ctx).
		Table("correction_proposals AS p").
		Select("p.*, r.title AS recipe_title, COALESCE(u.nickname, '') AS proposer_nickname").
		Joins("JOIN recipe_versions v ON v.id = p.recipe_version_id").
		Joins("JOIN recipes r ON r.id = v.recipe_id").
		Joins("LEFT JOIN users u ON u.id = p.proposer_id").
		Where("p.status = ? AND r.creator_id = ?", model.CorrectionStatusPending, creatorID).
		Order("p.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list pending correction proposals: %w", err)
	}

	items := make([]PendingProposalItem, 0, len(rows))
	for i := range rows {
		proposal := rows[i].CorrectionProposal
		items = append(items, PendingProposalItem{
			Proposal:         &proposal,
			RecipeTitle:      rows[i].RecipeTitle,
			ProposerNickname: rows[i].ProposerNickname,
		})
	}
	return items, nil
}

// reviewedProposalRow 是已审核列表查询的扫描行。
type reviewedProposalRow struct {
	model.CorrectionProposal `gorm:"embedded"`
	RecipeTitle              string `gorm:"column:recipe_title"`
}

func (r *GormCorrectionRepository) ListReviewedByProposer(ctx context.Context, proposerID uuid.UUID, limit int) ([]ReviewedProposalItem, error) {
	var rows []reviewedProposalRow
	err := r.db.WithContext(ctx).
		Table("correction_proposals AS cp").
		Select("cp.*, v.title AS recipe_title").
		Joins("JOIN recipe_versions v ON v.id = cp.recipe_version_id").
		Where("cp.proposer_id = ? AND cp.status IN ?", proposerID, []string{model.CorrectionStatusApproved, model.CorrectionStatusRejected}).
		Order("cp.updated_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list reviewed proposals: %w", err)
	}
	items := make([]ReviewedProposalItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ReviewedProposalItem{Proposal: row.CorrectionProposal, RecipeTitle: row.RecipeTitle})
	}
	return items, nil
}
