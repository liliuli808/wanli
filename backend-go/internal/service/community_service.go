package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

var (
	ErrCommunityVersionNotFound = errors.New("公开菜谱版本不存在")
	ErrCommunityContentInvalid  = errors.New("社区内容校验失败")
	ErrCommunityItemNotOwned    = errors.New("社区内容不存在或不属于当前用户")
)

type CommunityService struct {
	repo           repository.CommunityRepository
	photoURLPrefix string
}

func NewCommunityService(
	repo repository.CommunityRepository,
	photoURLPrefix ...string,
) *CommunityService {
	service := &CommunityService{repo: repo}
	if len(photoURLPrefix) > 0 {
		service.photoURLPrefix = photoURLPrefix[0]
	}
	return service
}

func (s *CommunityService) CreateComment(
	ctx context.Context,
	userID, versionID uuid.UUID,
	content string,
) (*model.Comment, error) {
	if err := s.requirePublicVersion(ctx, versionID); err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 1000 {
		return nil, fmt.Errorf("%w: 评论须为 1-1000 个字符", ErrCommunityContentInvalid)
	}
	comment := &model.Comment{
		RecipeVersionID: versionID,
		UserID:          &userID,
		Content:         content,
		Status:          model.CommunityStatusActive,
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *CommunityService) ListComments(
	ctx context.Context,
	versionID uuid.UUID,
) ([]repository.CommentListItem, error) {
	if err := s.requirePublicVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.repo.ListComments(ctx, versionID)
}

func (s *CommunityService) DeleteComment(ctx context.Context, userID, commentID uuid.UUID) error {
	deleted, err := s.repo.SoftDeleteComment(ctx, userID, commentID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrCommunityItemNotOwned
	}
	return nil
}

func (s *CommunityService) CreateCookingRecord(
	ctx context.Context,
	userID, versionID uuid.UUID,
	rating *int,
	photoURL, feedback *string,
) (*model.CookingRecord, error) {
	if err := s.requirePublicVersion(ctx, versionID); err != nil {
		return nil, err
	}
	if rating != nil && (*rating < 1 || *rating > 5) {
		return nil, fmt.Errorf("%w: rating 只能为 1-5", ErrCommunityContentInvalid)
	}
	photoURL = trimOptional(photoURL)
	feedback = trimOptional(feedback)
	if photoURL == nil && feedback == nil && rating == nil {
		return nil, fmt.Errorf("%w: 评分、反馈和成品图至少填写一项", ErrCommunityContentInvalid)
	}
	if feedback != nil && len([]rune(*feedback)) > 2000 {
		return nil, fmt.Errorf("%w: 反馈不能超过 2000 个字符", ErrCommunityContentInvalid)
	}
	if photoURL != nil && !validHTTPURL(*photoURL) {
		return nil, fmt.Errorf("%w: photo_url 非法", ErrCommunityContentInvalid)
	}
	if photoURL != nil && s.photoURLPrefix != "" && !strings.HasPrefix(*photoURL, s.photoURLPrefix) {
		return nil, fmt.Errorf("%w: photo_url 不是本站成品图地址", ErrCommunityContentInvalid)
	}
	record := &model.CookingRecord{
		RecipeVersionID: versionID,
		UserID:          userID,
		Rating:          rating,
		PhotoURL:        photoURL,
		Feedback:        feedback,
		Status:          model.CommunityStatusActive,
	}
	if err := s.repo.CreateCookingRecord(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *CommunityService) ListCookingRecords(
	ctx context.Context,
	versionID uuid.UUID,
) ([]repository.CookingRecordListItem, error) {
	if err := s.requirePublicVersion(ctx, versionID); err != nil {
		return nil, err
	}
	return s.repo.ListCookingRecords(ctx, versionID)
}

// ListCookingFeed 返回全站最近的「做过」动态，limit 超出 1~50 时回退为 20。
func (s *CommunityService) ListCookingFeed(ctx context.Context, limit int) ([]repository.CookingFeedItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.ListRecentCookingFeed(ctx, limit)
}

// ListMyCookingRecords 返回用户自己的「做过」记录（照片墙用），封顶 100 条。
func (s *CommunityService) ListMyCookingRecords(ctx context.Context, userID uuid.UUID, limit int) ([]repository.MyCookingRecordItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 60
	}
	return s.repo.ListCookingRecordsByUser(ctx, userID, limit)
}

func (s *CommunityService) DeleteCookingRecord(ctx context.Context, userID, recordID uuid.UUID) error {
	deleted, err := s.repo.SoftDeleteCookingRecord(ctx, userID, recordID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrCommunityItemNotOwned
	}
	return nil
}

func (s *CommunityService) requirePublicVersion(ctx context.Context, versionID uuid.UUID) error {
	ok, err := s.repo.IsVersionPublicReadable(ctx, versionID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCommunityVersionNotFound
	}
	return nil
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
