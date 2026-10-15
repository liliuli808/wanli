package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

type fakeMessageCommunityRepo struct {
	repository.CommunityRepository // 嵌入接口，未实现的方法 panic；本测试只用下面两个
	comments                       []repository.UserVersionActivityItem
	cookings                       []repository.UserVersionActivityItem
	err                            error
}

func (r *fakeMessageCommunityRepo) ListRecentCommentsOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]repository.UserVersionActivityItem, error) {
	return r.comments, r.err
}
func (r *fakeMessageCommunityRepo) ListRecentCookingOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]repository.UserVersionActivityItem, error) {
	return r.cookings, nil
}

type fakeMessageCorrectionRepo struct {
	repository.CorrectionRepository
	reviewed []repository.ReviewedProposalItem
	pending  []repository.PendingProposalItem
}

func (r *fakeMessageCorrectionRepo) ListReviewedByProposer(ctx context.Context, proposerID uuid.UUID, limit int) ([]repository.ReviewedProposalItem, error) {
	return r.reviewed, nil
}
func (r *fakeMessageCorrectionRepo) ListPendingByCreator(ctx context.Context, creatorID uuid.UUID) ([]repository.PendingProposalItem, error) {
	return r.pending, nil
}

func TestMessage_ListMergesAndSortsDesc(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Hour)
	community := &fakeMessageCommunityRepo{
		comments: []repository.UserVersionActivityItem{{ID: uuid.New(), CreatedAt: old, RecipeTitle: "红烧肉"}},
		cookings: []repository.UserVersionActivityItem{{ID: uuid.New(), CreatedAt: now, RecipeTitle: "红烧肉"}},
	}
	correction := &fakeMessageCorrectionRepo{}
	svc := NewMessageService(community, correction)

	items, err := svc.ListMessages(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, MessageTypeCookingRecord, items[0].Type) // 新的在前
	assert.Equal(t, MessageTypeComment, items[1].Type)
}

func TestMessage_ListMapsCorrectionTypes(t *testing.T) {
	community := &fakeMessageCommunityRepo{}
	correction := &fakeMessageCorrectionRepo{
		reviewed: []repository.ReviewedProposalItem{{Proposal: model.CorrectionProposal{ID: uuid.New(), Status: "APPROVED"}, RecipeTitle: "红烧肉"}},
		pending: []repository.PendingProposalItem{{
			Proposal:    &model.CorrectionProposal{ID: uuid.New()},
			RecipeTitle: "糖醋排骨",
		}},
	}
	svc := NewMessageService(community, correction)

	items, err := svc.ListMessages(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Len(t, items, 2)
	types := map[MessageType]bool{}
	for _, it := range items {
		types[it.Type] = true
	}
	assert.True(t, types[MessageTypeCorrectionReviewed])
	assert.True(t, types[MessageTypeCorrectionPending])
	// correction_reviewed 的 Text 应包含审核结果
	for _, it := range items {
		if it.Type == MessageTypeCorrectionReviewed {
			require.NotNil(t, it.Text)
			assert.Contains(t, *it.Text, "通过")
		}
	}
}

func TestMessage_ListTruncatesTo50(t *testing.T) {
	now := time.Now()
	cookings := make([]repository.UserVersionActivityItem, 60)
	for i := range cookings {
		cookings[i] = repository.UserVersionActivityItem{ID: uuid.New(), CreatedAt: now.Add(-time.Duration(i) * time.Minute), RecipeTitle: "红烧肉"}
	}
	community := &fakeMessageCommunityRepo{cookings: cookings}
	correction := &fakeMessageCorrectionRepo{}
	svc := NewMessageService(community, correction)

	items, err := svc.ListMessages(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Len(t, items, 50)
}

func TestMessage_ListPropagatesRepoError(t *testing.T) {
	sentinel := errors.New("db down")
	community := &fakeMessageCommunityRepo{err: sentinel}
	correction := &fakeMessageCorrectionRepo{}
	svc := NewMessageService(community, correction)

	_, err := svc.ListMessages(context.Background(), uuid.New())
	assert.ErrorIs(t, err, sentinel)
}
