# 碗里 UI 重设计 · 后端小补充 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Android 重设计提供 4 组数据支撑：用户资料编辑、首页「做过」聚合 Feed、消息聚合、我的做过照片墙 + 发现页做过计数。

**Architecture:** 沿用现有分层（handler → service → repository 接口 + Gorm 实现），所有响应走 `pkg/response` 信封，错误走 `pkg/apperror`。测试按现有惯例写在 service 层（手写 fake 实现 repository 接口 + testify）。

**Tech Stack:** Go 1.22, gin, GORM(postgres), golang-migrate, testify。

**注意：** `/home/jichi/wanli` 当前不是 git 仓库。各任务的「提交」步骤为可选检查点（若已 `git init` 则执行，否则跳过）。

**工作目录：** 所有命令在 `/home/jichi/wanli/backend-go` 下执行。

**背景知识（已勘察确认的现有代码事实）：**
- 用户表 `users` **已有** `nickname`/`avatar_url` 列（migration 000001），`model.User` 已有对应字段——不需要新迁移。
- 路由全部注册在 `internal/server/server.go` 的 `registerRoutes`；handler 在 `server.New` 中装配。
- `currentUserID(c)` 辅助函数定义在 `internal/handler/kitchen_handler.go:40`，同 package 可直接用。
- 软删除状态常量：`model.CommunityStatusActive` / `model.ContentStatusActive`（值为 `"ACTIVE"`）。
- 公开可读判定模式见 `GormCommunityRepository.IsVersionPublicReadable`（join recipes + 双侧 status/is_public 过滤）。
- service 测试惯例：fake 实现 repository 接口，`assert.ErrorIs` 断言哨兵错误。

---

### Task 1: 用户资料编辑（PUT /me/profile）+ Me 返回完整资料

**Files:**
- Modify: `internal/repository/user_repository.go`（接口 + Gorm 实现各加一个方法）
- Modify: `internal/service/auth_service.go`（加 `GetProfile` / `UpdateProfile`，加哨兵错误）
- Modify: `internal/handler/auth_handler.go`（改 `Me`，加 `UpdateProfile`）
- Modify: `internal/server/server.go`（注册新路由）
- Test: `internal/service/auth_service_test.go`

- [ ] **Step 1: 在 UserRepository 接口与实现中加 UpdateProfile**

`internal/repository/user_repository.go` 的 `UserRepository` 接口追加：

```go
// UpdateProfile 更新用户昵称与头像，返回更新后的用户；用户不存在时返回 (nil, nil)。
UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error)
```

`GormUserRepository` 追加实现（文件已 import `errors`/`fmt`/`gorm`，若没有则补上）：

```go
func (r *GormUserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	user.Nickname = nickname
	user.AvatarURL = avatarURL
	if err := r.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return &user, nil
}
```

- [ ] **Step 2: 写失败测试**

`internal/service/auth_service_test.go` 中找到现有的 fake user repo（实现 `UserRepository` 接口的测试替身；接口加了方法后它必须补实现才能编译）。给 fake 补：

```go
updateProfileCalls int
updatedNickname  string
updatedAvatar    *string
user             *model.User // GetByID/UpdateProfile 返回值

func (r *fakeUserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	r.updateProfileCalls++
	r.updatedNickname = nickname
	r.updatedAvatar = avatarURL
	if r.user == nil {
		return nil, nil
	}
	r.user.Nickname = nickname
	r.user.AvatarURL = avatarURL
	return r.user, nil
}
```

（fake 的实际类型名以文件里现有声明为准；若现有 fake 是内联结构体就照抄其风格。）

追加测试：

```go
func TestAuth_UpdateProfileTrimsAndRejectsEmptyNickname(t *testing.T) {
	repo := &fakeUserRepo{user: &model.User{ID: uuid.New(), Email: "a@b.c", Nickname: "旧昵称"}}
	svc := NewAuthService(repo /* …沿用文件中现有构造参数… */)

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), "   ", nil)
	assert.ErrorIs(t, err, ErrProfileInvalid)
	assert.Equal(t, 0, repo.updateProfileCalls)
}

func TestAuth_UpdateProfileTrimsNicknameAndEmptyAvatar(t *testing.T) {
	repo := &fakeUserRepo{user: &model.User{ID: uuid.New(), Email: "a@b.c", Nickname: "旧昵称"}}
	svc := NewAuthService(repo /* …同上… */)

	empty := "  "
	user, err := svc.UpdateProfile(context.Background(), uuid.New(), "  新昵称  ", &empty)
	require.NoError(t, err)
	assert.Equal(t, "新昵称", user.Nickname)
	assert.Nil(t, user.AvatarURL)
}

func TestAuth_UpdateProfileRejectsTooLongNickname(t *testing.T) {
	repo := &fakeUserRepo{user: &model.User{ID: uuid.New()}}
	svc := NewAuthService(repo /* …同上… */)

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), strings.Repeat("很", 51), nil)
	assert.ErrorIs(t, err, ErrProfileInvalid)
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/service/ -run TestAuth_UpdateProfile -v`
Expected: FAIL（`ErrProfileInvalid` / `UpdateProfile` 未定义，编译错误即视为红灯）

- [ ] **Step 4: 实现 service 方法**

`internal/service/auth_service.go` 顶部 `var (` 块追加：

```go
ErrProfileInvalid = errors.New("昵称不能为空且不超过 50 字")
```

追加方法（`s.userRepo` 的字段名以该文件实际声明为准；若 AuthService 构造参数里没有 user repo 字段则按现有字段名引用）：

```go
// GetProfile 返回用户完整资料；用户不存在返回 ErrUserNotFound。
func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile 校验并更新昵称/头像。昵称去首尾空白后需为 1~50 字；头像空白串视为清除。
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" || len([]rune(nickname)) > 50 {
		return nil, ErrProfileInvalid
	}
	if avatarURL != nil {
		trimmed := strings.TrimSpace(*avatarURL)
		if trimmed == "" {
			avatarURL = nil
		} else {
			avatarURL = &trimmed
		}
	}
	user, err := s.userRepo.UpdateProfile(ctx, userID, nickname, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
```

若文件中没有 `ErrUserNotFound`，在 `var (` 块加 `ErrUserNotFound = errors.New("用户不存在")`。若缺 `strings` import 则补上。

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/service/ -run TestAuth_ -v`
Expected: PASS（含既有 auth 测试全部通过）

- [ ] **Step 6: 改 handler——Me 返回完整资料 + 新增 UpdateProfile**

`internal/handler/auth_handler.go` 把 `Me` 整段替换为：

```go
// Me 获取当前登录用户信息。GET /api/v1/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	user, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询用户信息失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"nickname":   user.Nickname,
		"avatar_url": user.AvatarURL,
	})
}
```

追加：

```go
type updateProfileRequest struct {
	Nickname  string  `json:"nickname" binding:"required"`
	AvatarURL *string `json:"avatar_url"`
}

// UpdateProfile 更新昵称与头像。PUT /api/v1/me/profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}
	user, err := h.authService.UpdateProfile(c.Request.Context(), userID, req.Nickname, req.AvatarURL)
	if err != nil {
		if errors.Is(err, service.ErrProfileInvalid) {
			response.FailWithDetails(c, apperror.ErrValidation, err.Error())
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "更新资料失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"nickname":   user.Nickname,
		"avatar_url": user.AvatarURL,
	})
}
```

文件头 import 补 `"errors"`（若尚无）。`middleware` import 若不再被引用则删除。

- [ ] **Step 7: 注册路由并全量验证**

`internal/server/server.go` 的 `registerRoutes` 中 `authorized.GET("/me", authHandler.Me)` 下一行加：

```go
authorized.PUT("/me/profile", authHandler.UpdateProfile)
```

Run: `go build ./... && go test ./...`
Expected: 编译通过，全部测试 PASS

- [ ] **Step 8: 提交检查点（可选）**

```bash
git add internal/repository/user_repository.go internal/service/auth_service.go internal/service/auth_service_test.go internal/handler/auth_handler.go internal/server/server.go
git commit -m "feat(auth): add PUT /me/profile and enrich GET /me with nickname/avatar"
```

---

### Task 2: 首页做过聚合 Feed（GET /feed/cooking-records，公开）

**Files:**
- Modify: `internal/repository/community_repository.go`
- Modify: `internal/service/community_service.go`
- Modify: `internal/handler/community_handler.go`
- Modify: `internal/server/server.go`
- Test: `internal/service/community_service_test.go`

- [ ] **Step 1: 写失败测试**

`internal/service/community_service_test.go` 的 `fakeCommunityRepo` 补接口方法（先加让编译过的空实现，测试断言 limit 传递）：

```go
feedLimit int
feed      []repository.CookingFeedItem

func (r *fakeCommunityRepo) ListRecentCookingFeed(ctx context.Context, limit int) ([]repository.CookingFeedItem, error) {
	r.feedLimit = limit
	return r.feed, nil
}
```

追加测试：

```go
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
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/service/ -run TestCommunity_ListCookingFeed -v`
Expected: FAIL（`CookingFeedItem` / `ListCookingFeed` 未定义）

- [ ] **Step 3: repository 加 feed 查询**

`internal/repository/community_repository.go` 中 `CookingRecordListItem` 之后追加：

```go
// CookingFeedItem 是全站「做过」动态流的列表项（首页 Feed 用）。
type CookingFeedItem struct {
	ID             uuid.UUID
	Rating         *int
	PhotoURL       *string
	Feedback       *string
	AuthorNickname string
	CreatedAt      time.Time
	RecipeVersionID uuid.UUID
	RecipeID       uuid.UUID
	RecipeTitle    string
	CoverURL       *string
}
```

`CommunityRepository` 接口追加：

```go
ListRecentCookingFeed(ctx context.Context, limit int) ([]CookingFeedItem, error)
```

`GormCommunityRepository` 追加实现：

```go
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
```

- [ ] **Step 4: service 加方法**

`internal/service/community_service.go` 追加：

```go
// ListCookingFeed 返回全站最近的「做过」动态，limit 超出 1~50 时回退为 20。
func (s *CommunityService) ListCookingFeed(ctx context.Context, limit int) ([]repository.CookingFeedItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.ListRecentCookingFeed(ctx, limit)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/service/ -run TestCommunity_ -v`
Expected: PASS

- [ ] **Step 6: handler + 路由**

`internal/handler/community_handler.go` 追加（`strconv` 若无则 import）：

```go
// ListCookingFeed 全站「做过」动态流。GET /api/v1/feed/cooking-records?limit=20
func (h *CommunityHandler) ListCookingFeed(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.communityService.ListCookingFeed(c.Request.Context(), limit)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询动态失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":                item.ID.String(),
			"rating":            item.Rating,
			"photo_url":         item.PhotoURL,
			"feedback":          item.Feedback,
			"author_nickname":   item.AuthorNickname,
			"created_at":        item.CreatedAt,
			"recipe_version_id": item.RecipeVersionID.String(),
			"recipe_id":         item.RecipeID.String(),
			"recipe_title":      item.RecipeTitle,
			"cover_url":         item.CoverURL,
		})
	}
	response.Ok(c, gin.H{"items": list})
}
```

（handler 结构体字段名以现有为准，本文件其它方法怎么调 service 就怎么写。）

`internal/server/server.go` 的公开路由区（`v1.GET("/recipe-versions/:id/cooking-records", ...)` 附近）加：

```go
if communityHandler != nil {
	v1.GET("/feed/cooking-records", communityHandler.ListCookingFeed)
}
```

若公开区没有 nil 判断块而 authorized 区有，按该文件现有风格放置。

- [ ] **Step 7: 全量验证 + 手动冒烟**

Run: `go build ./... && go test ./...`
Expected: 全部 PASS

冒烟（本地起了服务时）：`curl http://localhost:8081/api/v1/feed/cooking-records?limit=5`，期望 `{"success":true,"data":{"items":[...]}}`。

- [ ] **Step 8: 提交检查点（可选）**

```bash
git add internal/repository/community_repository.go internal/service/community_service.go internal/service/community_service_test.go internal/handler/community_handler.go internal/server/server.go
git commit -m "feat(community): add public recent cooking-records feed endpoint"
```

---

### Task 3: 消息聚合（GET /me/messages，需登录）

聚合四类事件（v1 不做已读状态）：
1. `comment` — 有人评论了我发布的版本
2. `cooking_record` — 有人在我的版本上标记了做过
3. `correction_reviewed` — 我提交的纠错被通过/驳回
4. `correction_pending` — 待我审核的纠错（复用现有 `ListPendingByCreator`）

**Files:**
- Modify: `internal/repository/community_repository.go`
- Modify: `internal/repository/correction_repository.go`
- Create: `internal/service/message_service.go`
- Create: `internal/handler/message_handler.go`
- Modify: `internal/server/server.go`
- Test: `internal/service/message_service_test.go`

- [ ] **Step 1: repository 加查询方法**

`internal/repository/community_repository.go` 追加类型 + 两个接口方法 + 实现：

```go
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
```

接口追加：

```go
ListRecentCommentsOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]UserVersionActivityItem, error)
ListRecentCookingOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]UserVersionActivityItem, error)
```

实现（两个方法只差表名与 select 字段，照现有 join+Scan 风格）：

```go
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
```

（注意：`recipe_versions.author_id` 字段名以 `internal/model/recipe.go` 实际声明为准，若叫 `AuthorID *uuid.UUID` 则 join 条件写法不变。）

`internal/repository/correction_repository.go` 追加类型 + 接口方法 + 实现：

```go
// ReviewedProposalItem 是我提交的、已被审核的纠错建议。
type ReviewedProposalItem struct {
	Proposal        model.CorrectionProposal
	RecipeTitle     string
}

// ListReviewedByProposer 查询提议人提交的、已出审核结果（APPROVED/REJECTED）的建议。
ListReviewedByProposer(ctx context.Context, proposerID uuid.UUID, limit int) ([]ReviewedProposalItem, error)
```

```go
func (r *GormCorrectionRepository) ListReviewedByProposer(ctx context.Context, proposerID uuid.UUID, limit int) ([]ReviewedProposalItem, error) {
	var rows []struct {
		model.CorrectionProposal
		RecipeTitle string `gorm:"column:recipe_title"`
	}
	err := r.db.WithContext(ctx).
		Table("correction_proposals AS cp").
		Select("cp.*, v.title AS recipe_title").
		Joins("JOIN recipe_versions v ON v.id = cp.recipe_version_id").
		Where("cp.proposer_id = ? AND cp.status IN ?", proposerID, []string{model.ProposalStatusApproved, model.ProposalStatusRejected}).
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
```

（状态常量名以 `internal/model/correction.go` 实际为准，可能是 `ProposalStatusApproved` 或裸字符串 `"APPROVED"`/`"REJECTED"`。）

- [ ] **Step 2: 写失败测试**

Create `internal/service/message_service_test.go`：

```go
package service

import (
	"context"
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
	comments []repository.UserVersionActivityItem
	cookings []repository.UserVersionActivityItem
}

func (r *fakeMessageCommunityRepo) ListRecentCommentsOnUserVersions(ctx context.Context, authorID uuid.UUID, limit int) ([]repository.UserVersionActivityItem, error) {
	return r.comments, nil
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
		pending:  []repository.PendingProposalItem{{Proposal: model.CorrectionProposal{ID: uuid.New()}, RecipeTitle: "糖醋排骨"}},
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
			assert.Contains(t, *it.Text, "通过")
		}
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/service/ -run TestMessage_ -v`
Expected: FAIL（类型/函数未定义）

- [ ] **Step 4: 实现 message_service.go**

Create `internal/service/message_service.go`：

```go
package service

import (
	"context"
	"sort"

	"github.com/google/uuid"

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
		result := "通过"
		if r.Proposal.Status == "REJECTED" {
			result = "驳回"
		}
		text := "你的纠错建议已被" + result
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
		versionID := p.Proposal.RecipeVersionID
		text := p.Proposal.Reason
		items = append(items, MessageItem{
			Type: MessageTypeCorrectionPending, ID: p.Proposal.ID, ActorNickname: p.ProposerNickname,
			Text: &text, RecipeVersionID: &versionID, RecipeTitle: p.RecipeTitle, CreatedAt: p.Proposal.CreatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > messageListLimit {
		items = items[:messageListLimit]
	}
	return items, nil
}
```

注意：`PendingProposalItem` 的字段名（`Proposal`/`RecipeTitle`/`ProposerNickname`）以 `correction_repository.go:17` 实际声明为准；`MessageItem` 需要 import `"time"`。

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/service/ -run TestMessage_ -v`
Expected: PASS

- [ ] **Step 6: handler + 装配 + 路由**

Create `internal/handler/message_handler.go`：

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// MessageHandler 消息聚合接口。
type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

// List 当前用户的消息列表。GET /api/v1/me/messages
func (h *MessageHandler) List(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	items, err := h.messageService.ListMessages(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询消息失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		var versionID any
		if item.RecipeVersionID != nil {
			versionID = item.RecipeVersionID.String()
		}
		list = append(list, gin.H{
			"type":              string(item.Type),
			"id":                item.ID.String(),
			"actor_nickname":    item.ActorNickname,
			"text":              item.Text,
			"rating":            item.Rating,
			"recipe_version_id": versionID,
			"recipe_title":      item.RecipeTitle,
			"created_at":        item.CreatedAt,
		})
	}
	response.Ok(c, gin.H{"items": list})
}
```

`internal/server/server.go` 的 `New` 中，在 community/correction handler 装配之后加（repo 变量名以该文件实际为准）：

```go
messageHandler := handler.NewMessageHandler(service.NewMessageService(communityRepo, correctionRepo))
```

`registerRoutes` 签名加 `messageHandler *handler.MessageHandler` 参数，`New` 中调用处同步传参；`authorized` 组内加：

```go
authorized.GET("/me/messages", messageHandler.List)
```

- [ ] **Step 7: 全量验证 + 提交检查点（可选）**

Run: `go build ./... && go test ./...`
Expected: 全部 PASS

```bash
git add internal/repository/community_repository.go internal/repository/correction_repository.go internal/service/message_service.go internal/service/message_service_test.go internal/handler/message_handler.go internal/server/server.go
git commit -m "feat(messages): add aggregated GET /me/messages endpoint"
```

---

### Task 4: 我的做过（GET /me/cooking-records）+ 发现页做过计数

**Files:**
- Modify: `internal/repository/community_repository.go`
- Modify: `internal/service/community_service.go`
- Modify: `internal/handler/community_handler.go`
- Modify: `internal/repository/recipe_repository.go`（列表 SQL 加计数列）
- Modify: `internal/handler/recipe_handler.go`（ListRecipes 返回新字段）
- Modify: `internal/server/server.go`

- [ ] **Step 1: repository 加「我的做过」查询**

`internal/repository/community_repository.go` 追加：

```go
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
```

接口追加 `ListCookingRecordsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]MyCookingRecordItem, error)`，实现：

```go
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
```

- [ ] **Step 2: service + handler + 路由**

`community_service.go` 追加：

```go
// ListMyCookingRecords 返回用户自己的「做过」记录（照片墙用），封顶 100 条。
func (s *CommunityService) ListMyCookingRecords(ctx context.Context, userID uuid.UUID, limit int) ([]repository.MyCookingRecordItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 60
	}
	return s.repo.ListCookingRecordsByUser(ctx, userID, limit)
}
```

`community_handler.go` 追加：

```go
// ListMyCookingRecords 当前用户的「做过」列表。GET /api/v1/me/cooking-records
func (h *CommunityHandler) ListMyCookingRecords(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "60"))
	items, err := h.communityService.ListMyCookingRecords(c.Request.Context(), userID, limit)
	if err != nil {
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询失败，请稍后再试", http.StatusInternalServerError, err))
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, gin.H{
			"id":                item.ID.String(),
			"rating":            item.Rating,
			"photo_url":         item.PhotoURL,
			"feedback":          item.Feedback,
			"created_at":        item.CreatedAt,
			"recipe_version_id": item.RecipeVersionID.String(),
			"recipe_title":      item.RecipeTitle,
			"cover_url":         item.CoverURL,
		})
	}
	response.Ok(c, gin.H{"items": list})
}
```

`server.go` 的 `authorized` 组内（community nil 判断块中）加：

```go
authorized.GET("/me/cooking-records", communityHandler.ListMyCookingRecords)
```

service 测试：仿照 Task 2 Step 1 的 clamp 测试补 `TestCommunity_ListMyCookingRecordsClampsLimit`（断言 0→60、999→60、30→30）。

- [ ] **Step 3: 发现页列表加做过计数**

`internal/repository/recipe_repository.go`：
- `recipeListRow`（346 行附近）加字段 `CookingRecordCount int64`，并确认 `RecipeListItem`（44 行附近）同样加 `CookingRecordCount int64`（`RecipeListItem(row)` 转换依赖两结构字段一致）。
- `publicRecipeListSelect` 常量中 `public_version_count` 子查询之后加一列：

```sql
		(SELECT COUNT(*) FROM cooking_records cr
			JOIN recipe_versions pv2 ON pv2.id = cr.recipe_version_id
			WHERE pv2.recipe_id = r.id AND pv2.is_public = TRUE AND pv2.status = 'ACTIVE'
				AND cr.status = 'ACTIVE') AS cooking_record_count
```

`internal/handler/recipe_handler.go` 的 `ListRecipes` 中 `gin.H{...}` 加一行：

```go
"cooking_record_count": item.CookingRecordCount,
```

- [ ] **Step 4: 全量验证 + 提交检查点（可选）**

Run: `go build ./... && go test ./...`
Expected: 全部 PASS

```bash
git add internal/repository/community_repository.go internal/service/community_service.go internal/service/community_service_test.go internal/handler/community_handler.go internal/repository/recipe_repository.go internal/handler/recipe_handler.go internal/server/server.go
git commit -m "feat(community): add my cooking-records endpoint and per-recipe cooked count"
```

---

## 执行记录（2026-07-29）

4 个任务全部完成（实现 → 规格审查 → 质量审查循环通过）。质量审查遗留的非阻塞后续项：

1. **索引**：`cooking_records(created_at DESC) WHERE status='ACTIVE'`（全站 feed）、`cooking_records(user_id, created_at DESC) WHERE status='ACTIVE'`（我的做过）——量级上来后加迁移。
2. **计数包装查询**：`ListPublicRecipes` 的 COUNT 包装会对全部行求值两个相关子查询；食谱表变大后改为仅 `COUNT(r.id)`。
3. `ListPendingByCreator` 无 limit 参数（消息聚合每次全量取待审）；评论无 parent_id、消息无已读状态（v1 接受）。

## Self-Review 记录

- 规格覆盖：设计文档 §6 后端补充清单 3 项全部对应（Task 1 ↔ 昵称/头像；Task 2 ↔ 做过聚合流；Task 3 ↔ 消息聚合），另加 Task 4 支撑「我的」照片墙与发现页徽章。
- 已知限制（v1 接受，不动数据模型）：评论无回复结构（Comment 无 parent_id）；消息无已读状态；`recipe_versions.author_id` 为 NULL 的历史 SOURCE 版本不产生消息。
- API 契约文档：`docs/api-contract.md` 若存在且维护了接口清单，实现时同步补 4 个新端点说明（apperror.go 注释要求保持同步）。
