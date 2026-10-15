package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

type fakeCommunityRepo struct {
	publicVersion      bool
	comment            *model.Comment
	record             *model.CookingRecord
	commentDeleteOK    bool
	recordDeleteOK     bool
	deletedCommentID   uuid.UUID
	deletedRecordID    uuid.UUID
	deletedCommentUser uuid.UUID
	deletedRecordUser  uuid.UUID
	feedLimit          int
	feed               []repository.CookingFeedItem
	myRecordsLimit     int
	myRecords          []repository.MyCookingRecordItem
}

func (r *fakeCommunityRepo) IsVersionPublicReadable(context.Context, uuid.UUID) (bool, error) {
	return r.publicVersion, nil
}

func (r *fakeCommunityRepo) CreateComment(_ context.Context, comment *model.Comment) error {
	comment.ID = uuid.New()
	r.comment = comment
	return nil
}

func (r *fakeCommunityRepo) ListComments(context.Context, uuid.UUID) ([]repository.CommentListItem, error) {
	return nil, nil
}

func (r *fakeCommunityRepo) SoftDeleteComment(
	_ context.Context,
	userID, commentID uuid.UUID,
) (bool, error) {
	r.deletedCommentUser = userID
	r.deletedCommentID = commentID
	return r.commentDeleteOK, nil
}

func (r *fakeCommunityRepo) CreateCookingRecord(_ context.Context, record *model.CookingRecord) error {
	record.ID = uuid.New()
	r.record = record
	return nil
}

func (r *fakeCommunityRepo) ListCookingRecords(
	context.Context,
	uuid.UUID,
) ([]repository.CookingRecordListItem, error) {
	return nil, nil
}

func (r *fakeCommunityRepo) ListRecentCookingFeed(
	_ context.Context,
	limit int,
) ([]repository.CookingFeedItem, error) {
	r.feedLimit = limit
	return r.feed, nil
}

func (r *fakeCommunityRepo) SoftDeleteCookingRecord(
	_ context.Context,
	userID, recordID uuid.UUID,
) (bool, error) {
	r.deletedRecordUser = userID
	r.deletedRecordID = recordID
	return r.recordDeleteOK, nil
}

func (r *fakeCommunityRepo) ListRecentCommentsOnUserVersions(
	context.Context,
	uuid.UUID,
	int,
) ([]repository.UserVersionActivityItem, error) {
	return nil, nil
}

func (r *fakeCommunityRepo) ListRecentCookingOnUserVersions(
	context.Context,
	uuid.UUID,
	int,
) ([]repository.UserVersionActivityItem, error) {
	return nil, nil
}

func TestCommunity_CreateCommentRequiresPublicVersion(t *testing.T) {
	repo := &fakeCommunityRepo{publicVersion: false}
	svc := NewCommunityService(repo)

	_, err := svc.CreateComment(context.Background(), uuid.New(), uuid.New(), "很好吃")
	assert.ErrorIs(t, err, ErrCommunityVersionNotFound)
	assert.Nil(t, repo.comment)
}

func TestCommunity_CreateCommentTrimsAndSavesActive(t *testing.T) {
	repo := &fakeCommunityRepo{publicVersion: true}
	svc := NewCommunityService(repo)
	userID := uuid.New()
	versionID := uuid.New()

	comment, err := svc.CreateComment(context.Background(), userID, versionID, "  很好吃  ")
	require.NoError(t, err)
	assert.Equal(t, "很好吃", comment.Content)
	assert.Equal(t, model.CommunityStatusActive, comment.Status)
	assert.Equal(t, userID, *comment.UserID)
	assert.Equal(t, versionID, comment.RecipeVersionID)
}

func TestCommunity_DeleteCommentUsesOwnerScopedSoftDelete(t *testing.T) {
	repo := &fakeCommunityRepo{commentDeleteOK: true}
	svc := NewCommunityService(repo)
	userID := uuid.New()
	commentID := uuid.New()

	require.NoError(t, svc.DeleteComment(context.Background(), userID, commentID))
	assert.Equal(t, userID, repo.deletedCommentUser)
	assert.Equal(t, commentID, repo.deletedCommentID)

	repo.commentDeleteOK = false
	assert.ErrorIs(t, svc.DeleteComment(context.Background(), userID, uuid.New()), ErrCommunityItemNotOwned)
}

func TestCommunity_CreateCookingRecordValidatesAndSaves(t *testing.T) {
	repo := &fakeCommunityRepo{publicVersion: true}
	svc := NewCommunityService(
		repo,
		"http://10.0.2.2:9000/wanli-assets/uploads/records/",
	)
	userID := uuid.New()
	versionID := uuid.New()
	rating := 5
	photoURL := "http://10.0.2.2:9000/wanli-assets/uploads/records/photo.jpg"
	feedback := "  外酥里嫩  "

	record, err := svc.CreateCookingRecord(
		context.Background(), userID, versionID, &rating, &photoURL, &feedback,
	)
	require.NoError(t, err)
	assert.Equal(t, model.CommunityStatusActive, record.Status)
	assert.Equal(t, "外酥里嫩", *record.Feedback)
	assert.Equal(t, 5, *record.Rating)

	invalidRating := 6
	_, err = svc.CreateCookingRecord(
		context.Background(), userID, versionID, &invalidRating, nil, nil,
	)
	assert.ErrorIs(t, err, ErrCommunityContentInvalid)

	externalURL := "https://tracker.example/photo.jpg"
	_, err = svc.CreateCookingRecord(
		context.Background(), userID, versionID, nil, &externalURL, nil,
	)
	assert.ErrorIs(t, err, ErrCommunityContentInvalid)
}

func (r *fakeCommunityRepo) ListCookingRecordsByUser(
	_ context.Context,
	_ uuid.UUID,
	limit int,
) ([]repository.MyCookingRecordItem, error) {
	r.myRecordsLimit = limit
	return r.myRecords, nil
}

func TestCommunity_ListCookingFeedClampsLimit(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewCommunityService(repo)

	_, err := svc.ListCookingFeed(context.Background(), 0)
	require.NoError(t, err)
	assert.Equal(t, 20, repo.feedLimit)

	_, err = svc.ListCookingFeed(context.Background(), 999)
	require.NoError(t, err)
	assert.Equal(t, 20, repo.feedLimit)

	_, err = svc.ListCookingFeed(context.Background(), 30)
	require.NoError(t, err)
	assert.Equal(t, 30, repo.feedLimit)

	_, err = svc.ListCookingFeed(context.Background(), -5)
	require.NoError(t, err)
	assert.Equal(t, 20, repo.feedLimit)
}

func TestCommunity_ListMyCookingRecordsClampsLimit(t *testing.T) {
	repo := &fakeCommunityRepo{}
	svc := NewCommunityService(repo)
	userID := uuid.New()

	_, err := svc.ListMyCookingRecords(context.Background(), userID, 0)
	require.NoError(t, err)
	assert.Equal(t, 60, repo.myRecordsLimit)

	_, err = svc.ListMyCookingRecords(context.Background(), userID, 999)
	require.NoError(t, err)
	assert.Equal(t, 60, repo.myRecordsLimit)

	_, err = svc.ListMyCookingRecords(context.Background(), userID, 30)
	require.NoError(t, err)
	assert.Equal(t, 30, repo.myRecordsLimit)

	_, err = svc.ListMyCookingRecords(context.Background(), userID, -5)
	require.NoError(t, err)
	assert.Equal(t, 60, repo.myRecordsLimit)
}

func TestCommunity_DeleteCookingRecordUsesOwnerScopedSoftDelete(t *testing.T) {
	repo := &fakeCommunityRepo{recordDeleteOK: true}
	svc := NewCommunityService(repo)
	userID := uuid.New()
	recordID := uuid.New()

	require.NoError(t, svc.DeleteCookingRecord(context.Background(), userID, recordID))
	assert.Equal(t, userID, repo.deletedRecordUser)
	assert.Equal(t, recordID, repo.deletedRecordID)

	repo.recordDeleteOK = false
	assert.ErrorIs(
		t,
		svc.DeleteCookingRecord(context.Background(), userID, uuid.New()),
		ErrCommunityItemNotOwned,
	)
}
