package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

// 消息类型。
type MessageType string

const (
	MessageTypeComment            MessageType = "comment"
	MessageTypeCookingRecord      MessageType = "cooking_record"
	MessageTypeCorrectionReviewed MessageType = "correction_reviewed"
	MessageTypeCorrectionPending  MessageType = "correction_pending"
)

const messageListLimit = 50

// MessageItem 是消息中心的一条聚合消息。
type MessageItem struct {
	Type            MessageType
	ID              uuid.UUID
	ActorNickname   string
	Text            *string
	Rating          *int
	RecipeVersionID *uuid.UUID
	RecipeTitle     string
	CreatedAt       time.Time
}

type MessageService struct {
	communityRepo  repository.CommunityRepository
	correctionRepo repository.CorrectionRepository
}

func NewMessageService(communityRepo repository.CommunityRepository, correctionRepo repository.CorrectionRepository) *MessageService {
	return &MessageService{communityRepo: communityRepo, correctionRepo: correctionRepo}
}

// ListMessages 聚合四类事件，按时间倒序，封顶 50 条。
func (s *MessageService) ListMessages(ctx context.Context, userID uuid.UUID) ([]MessageItem, error) {
	items := make([]MessageItem, 0)

	comments, err := s.communityRepo.ListRecentCommentsOnUserVersions(ctx, userID, messageListLimit)
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		versionID := c.RecipeVersionID
		items = append(items, MessageItem{
			Type: MessageTypeComment, ID: c.ID, ActorNickname: c.ActorNickname,
			Text: c.Text, RecipeVersionID: &versionID, RecipeTitle: c.RecipeTitle, CreatedAt: c.CreatedAt,
		})
	}

	cookings, err := s.communityRepo.ListRecentCookingOnUserVersions(ctx, userID, messageListLimit)
	if err != nil {
		return nil, err
	}
	for _, c := range cookings {
		versionID := c.RecipeVersionID
		items = append(items, MessageItem{
			Type: MessageTypeCookingRecord, ID: c.ID, ActorNickname: c.ActorNickname,
			Text: c.Text, Rating: c.Rating, RecipeVersionID: &versionID, RecipeTitle: c.RecipeTitle, CreatedAt: c.CreatedAt,
		})
	}

	reviewed, err := s.correctionRepo.ListReviewedByProposer(ctx, userID, messageListLimit)
	if err != nil {
		return nil, err
	}
	for _, r := range reviewed {
		versionID := r.Proposal.RecipeVersionID
		text := "你的纠错建议已通过审核"
		if r.Proposal.Status == model.CorrectionStatusRejected {
			text = "你的纠错建议已被驳回"
		}
		items = append(items, MessageItem{
			Type: MessageTypeCorrectionReviewed, ID: r.Proposal.ID,
			Text: &text, RecipeVersionID: &versionID, RecipeTitle: r.RecipeTitle, CreatedAt: r.Proposal.UpdatedAt,
		})
	}

	pending, err := s.correctionRepo.ListPendingByCreator(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, p := range pending {
		if p.Proposal == nil {
			continue
		}
		versionID := p.Proposal.RecipeVersionID
		text := p.Proposal.Reason
		items = append(items, MessageItem{
			Type: MessageTypeCorrectionPending, ID: p.Proposal.ID, ActorNickname: p.ProposerNickname,
			Text: &text, RecipeVersionID: &versionID, RecipeTitle: p.RecipeTitle, CreatedAt: p.Proposal.CreatedAt,
		})
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > messageListLimit {
		items = items[:messageListLimit]
	}
	return items, nil
}
