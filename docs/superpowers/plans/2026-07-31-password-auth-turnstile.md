# 密码登录 + Cloudflare Turnstile 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用「邮箱 + 密码 + Cloudflare Turnstile 人机验证」替换现有「邮箱验证码」认证，注册/登录分离，附带登录失败限流。

**Architecture:** 后端删除验证码链路（`MemoryCodeStore`、`SendEmailCode`、SMTP 配置），新增 `internal/infrastructure/turnstile`（siteverify 校验器）与 `service.LoginLimiter`（Redis/内存双实现），AuthService 暴露 `Register`/`Login`；Android 用 WebView 加载后端托管的 Turnstile 组件页（Cloudflare 官方对原生 App 的标准做法，无官方 SDK），拿 token 后随请求提交。

**Tech Stack:** Go 1.22（`net/http`、`go-redis/v9`、`bcrypt`、`miniredis` 测试）、Gin、PostgreSQL（golang-migrate）、Android Jetpack Compose + WebView、Retrofit、Kotlin 2.0。

**重要说明：**
1. **SDK 变更**：原 spec 假设存在 `com.cloudflare:turnstile-sdk-android`。经核实该 artifact 不在 Maven Central，Cloudflare 官方文档明确原生 App 只能走 **WebView 加载托管页面**，故按 WebView 方案实施（无需新增 Android 依赖）。
2. **无 git 仓库**：当前项目没有 `.git`，各任务的 commit 步骤仅在你先 `git init` 后生效；未初始化前跳过即可。
3. **Android 构建环境**：本机（WSL 与 Windows）均无 JDK，Android 单测/编译需在装有 JDK 17 + Android SDK 的机器上执行；后端 Go 命令在本机 WSL（Ubuntu-Preview）可直接运行。

**所有 Go 命令在 WSL 内执行**（本会话 PowerShell 中需加前缀，例如：`wsl -d Ubuntu-Preview -- bash -lc "cd /home/jichi/wanli/backend-go && go test ./..."`）。

---

### Task 1: Turnstile 校验器（后端，TDD）

**Files:**
- Create: `backend-go/internal/infrastructure/turnstile/turnstile.go`
- Test: `backend-go/internal/infrastructure/turnstile/turnstile_test.go`

- [ ] **Step 1: 写失败测试**

`backend-go/internal/infrastructure/turnstile/turnstile_test.go`：

```go
package turnstile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient 起一个假 siteverify 服务，断言请求带 secret/response，返回固定 success。
func newTestClient(t *testing.T, success bool) *CloudflareTurnstile {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "test-secret", r.Form.Get("secret"))
		assert.Equal(t, "test-token", r.Form.Get("response"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": success})
	}))
	t.Cleanup(srv.Close)
	return NewWithEndpoint("test-secret", srv.URL, srv.Client())
}

func TestVerify_Success(t *testing.T) {
	c := newTestClient(t, true)
	ok, err := c.Verify(context.Background(), "test-token", "")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerify_Rejected(t *testing.T) {
	c := newTestClient(t, false)
	ok, err := c.Verify(context.Background(), "test-token", "")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerify_EmptyToken_ReturnsFalseWithoutRequest(t *testing.T) {
	c := &CloudflareTurnstile{} // 无 endpoint，若发请求会 panic，可证明未走 HTTP
	ok, err := c.Verify(context.Background(), "", "")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := NewWithEndpoint("test-secret", srv.URL, srv.Client())

	_, err := c.Verify(context.Background(), "test-token", "")
	require.Error(t, err)
}

func TestVerify_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	addr := srv.URL
	srv.Close() // 立刻关掉，制造连接错误

	c := NewWithEndpoint("test-secret", addr, srv.Client())
	_, err := c.Verify(context.Background(), "test-token", "")
	require.Error(t, err)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/infrastructure/turnstile/ -v`
Expected: 编译失败（`turnstile` 包不存在）。

- [ ] **Step 3: 实现**

`backend-go/internal/infrastructure/turnstile/turnstile.go`：

```go
package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Verifier 校验前端提交的 Turnstile token。
type Verifier interface {
	Verify(ctx context.Context, token, remoteIP string) (bool, error)
}

// CloudflareTurnstile 调用 challenges.cloudflare.com siteverify 的客户端。
type CloudflareTurnstile struct {
	secretKey string
	endpoint  string
	client    *http.Client
}

// New 创建指向 Cloudflare 官方 siteverify 端点的客户端。
func New(secretKey string) *CloudflareTurnstile {
	return &CloudflareTurnstile{
		secretKey: secretKey,
		endpoint:  "https://challenges.cloudflare.com/turnstile/v0/siteverify",
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// NewWithEndpoint 指定 endpoint（测试用）。
func NewWithEndpoint(secretKey, endpoint string, client *http.Client) *CloudflareTurnstile {
	return &CloudflareTurnstile{secretKey: secretKey, endpoint: endpoint, client: client}
}

type siteVerifyResponse struct {
	Success bool `json:"success"`
}

// Verify 返回 token 是否通过人机验证。token 为空直接返回 (false, nil)。
func (c *CloudflareTurnstile) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if token == "" {
		return false, nil
	}
	form := url.Values{}
	form.Set("secret", c.secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, fmt.Errorf("build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("siteverify request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("siteverify unexpected status: %d", resp.StatusCode)
	}
	var out siteVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("decode siteverify response: %w", err)
	}
	return out.Success, nil
}

// Stub 是未配置 secret key 时的闭锁实现，恒返回未通过，防止误放行。
type Stub struct{}

// Verify 恒返回 false。
func (Stub) Verify(context.Context, string, string) (bool, error) {
	return false, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/infrastructure/turnstile/ -v`
Expected: 全部 PASS。

- [ ] **Step 5: Commit**（仅当 `git init` 后）

```bash
git add backend-go/internal/infrastructure/turnstile/
git commit -m "feat: add cloudflare turnstile siteverify verifier"
```

---

### Task 2: 登录限流器（后端，TDD）

**Files:**
- Create: `backend-go/internal/service/login_limiter.go`
- Test: `backend-go/internal/service/login_limiter_test.go`

- [ ] **Step 1: 写失败测试**

`backend-go/internal/service/login_limiter_test.go`：

```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRedisLimiter(t *testing.T) (*RedisLoginLimiter, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisLoginLimiter(rdb, 5, 15*time.Minute), s
}

func TestRedisLoginLimiter_LocksAfterMaxFailures(t *testing.T) {
	limiter, _ := newRedisLimiter(t)
	ctx := context.Background()

	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}

	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked)
}

func TestRedisLoginLimiter_ResetUnlocks(t *testing.T) {
	limiter, _ := newRedisLimiter(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}
	require.NoError(t, limiter.Reset(ctx, "a@b.c"))

	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)
}

func TestMemoryLoginLimiter_BehavesSame(t *testing.T) {
	limiter := NewMemoryLoginLimiter(5, 15*time.Minute)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "a@b.c"))
	}
	locked, err := limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.True(t, locked)

	require.NoError(t, limiter.Reset(ctx, "a@b.c"))
	locked, err = limiter.IsLocked(ctx, "a@b.c")
	require.NoError(t, err)
	assert.False(t, locked)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/service/ -run LoginLimiter -v`
Expected: 编译失败（类型未定义）。

- [ ] **Step 3: 实现**

`backend-go/internal/service/login_limiter.go`：

```go
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginLimiter 记录登录失败并判定账号是否被临时锁定。
type LoginLimiter interface {
	IsLocked(ctx context.Context, email string) (bool, error)
	RecordFailure(ctx context.Context, email string) error
	Reset(ctx context.Context, email string) error
}

// RedisLoginLimiter 基于 Redis 的实现（多实例共享）。
type RedisLoginLimiter struct {
	rdb            *redis.Client
	maxFailures    int64
	lockoutTimeout time.Duration
}

// NewRedisLoginLimiter 创建 Redis 限流器。
func NewRedisLoginLimiter(rdb *redis.Client, maxFailures int, lockout time.Duration) *RedisLoginLimiter {
	return &RedisLoginLimiter{
		rdb:            rdb,
		maxFailures:    int64(maxFailures),
		lockoutTimeout: lockout,
	}
}

func (l *RedisLoginLimiter) key(email string) string {
	return "auth:fail:" + email
}

// IsLocked 返回账号当前是否处于锁定状态。
func (l *RedisLoginLimiter) IsLocked(ctx context.Context, email string) (bool, error) {
	n, err := l.rdb.Get(ctx, l.key(email)).Int64()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get login failures: %w", err)
	}
	return n >= l.maxFailures, nil
}

// RecordFailure 累加一次失败并刷新过期时间。
func (l *RedisLoginLimiter) RecordFailure(ctx context.Context, email string) error {
	key := l.key(email)
	pipe := l.rdb.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, l.lockoutTimeout)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	return nil
}

// Reset 清除失败计数（登录成功后调用）。
func (l *RedisLoginLimiter) Reset(ctx context.Context, email string) error {
	if err := l.rdb.Del(ctx, l.key(email)).Err(); err != nil {
		return fmt.Errorf("reset login failures: %w", err)
	}
	return nil
}

type memEntry struct {
	count int
	until time.Time
}

// MemoryLoginLimiter 进程内实现（开发环境无 Redis 时兜底，重启即重置）。
type MemoryLoginLimiter struct {
	mu             sync.Mutex
	failures       map[string]memEntry
	maxFailures    int
	lockoutTimeout time.Duration
}

// NewMemoryLoginLimiter 创建内存限流器。
func NewMemoryLoginLimiter(maxFailures int, lockout time.Duration) *MemoryLoginLimiter {
	return &MemoryLoginLimiter{
		failures:       make(map[string]memEntry),
		maxFailures:    maxFailures,
		lockoutTimeout: lockout,
	}
}

// IsLocked 返回账号是否处于锁定状态。
func (l *MemoryLoginLimiter) IsLocked(ctx context.Context, email string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.failures[email]
	if !ok {
		return false, nil
	}
	if time.Now().After(e.until) {
		delete(l.failures, email)
		return false, nil
	}
	return e.count >= l.maxFailures, nil
}

// RecordFailure 累加一次失败。
func (l *MemoryLoginLimiter) RecordFailure(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.failures[email]
	if !ok || now.After(e.until) {
		l.failures[email] = memEntry{count: 1, until: now.Add(l.lockoutTimeout)}
		return nil
	}
	e.count++
	l.failures[email] = e
	return nil
}

// Reset 清除失败计数。
func (l *MemoryLoginLimiter) Reset(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, email)
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/service/ -run LoginLimiter -v`
Expected: 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/service/login_limiter.go backend-go/internal/service/login_limiter_test.go
git commit -m "feat: add login failure limiter (redis + memory)"
```

---

### Task 3: 配置改造（后端）

**Files:**
- Modify: `backend-go/internal/config/config.go`
- Modify: `backend-go/internal/config/config_test.go`
- Modify: `.env.example`

- [ ] **Step 1: 改 config.go：新增 Turnstile/Auth，删除 SMTP**

在 `backend-go/internal/config/config.go` 中：

1. `Config` 结构体删除 `SMTP SMTPConfig`，新增两行：

```go
	Turnstile TurnstileConfig
	Auth      AuthConfig
```

2. 删除 `SMTPConfig` 结构体（第 166-172 行）。

3. 新增两个结构体：

```go
type TurnstileConfig struct {
	SiteKey   string
	SecretKey string
}

type AuthConfig struct {
	MaxFailures    int
	LockoutMinutes int
}
```

4. `Load()` 中删除 `v.SetDefault("SMTP_MOCK", true)` 与 `v.SetDefault("SMTP_PORT", 587)`，新增：

```go
	v.SetDefault("AUTH_MAX_FAILURES", 5)
	v.SetDefault("AUTH_LOCKOUT_MINUTES", 15)
```

5. `Load()` 返回值中删除 `SMTP: SMTPConfig{...}` 块，新增：

```go
		Turnstile: TurnstileConfig{
			SiteKey:   v.GetString("TURNSTILE_SITE_KEY"),
			SecretKey: v.GetString("TURNSTILE_SECRET_KEY"),
		},
		Auth: AuthConfig{
			MaxFailures:    v.GetInt("AUTH_MAX_FAILURES"),
			LockoutMinutes: v.GetInt("AUTH_LOCKOUT_MINUTES"),
		},
```

- [ ] **Step 2: 改 config_test.go**

`backend-go/internal/config/config_test.go`：
- `TestLoad_Defaults` 中 `assert.True(t, cfg.SMTP.MockMode)` 替换为：

```go
	assert.Empty(t, cfg.Turnstile.SiteKey)
	assert.Empty(t, cfg.Turnstile.SecretKey)
	assert.Equal(t, 5, cfg.Auth.MaxFailures)
	assert.Equal(t, 15, cfg.Auth.LockoutMinutes)
```

- `TestLoad_OSEnv` 中 `t.Setenv("SMTP_MOCK", "false")` 与 `assert.False(t, cfg.SMTP.MockMode)` 替换为：

```go
	t.Setenv("TURNSTILE_SECRET_KEY", "test-secret")
```

与

```go
	assert.Equal(t, "test-secret", cfg.Turnstile.SecretKey)
```

- [ ] **Step 3: 运行配置测试**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/config/ -v`
Expected: 全部 PASS。

- [ ] **Step 4: 改 .env.example**

`.env.example` 中删除邮件块：

```
# 邮件：默认 mock（验证码打到后端日志）；配置真实 SMTP 时改为 false
SMTP_MOCK=true
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
```

替换为：

```
# === Turnstile 人机验证（注册/登录）===
# 开发可用官方测试 key：TURNSTILE_SITE_KEY=1x00000000000000000000AA（恒通过）
# TURNSTILE_SECRET_KEY 非空即可（测试 key 无匹配要求）；上线前到 Cloudflare 后台创建真实 widget
TURNSTILE_SITE_KEY=
TURNSTILE_SECRET_KEY=
# 登录失败限流：连续失败 AUTH_MAX_FAILURES 次锁 AUTH_LOCKOUT_MINUTES 分钟
AUTH_MAX_FAILURES=5
AUTH_LOCKOUT_MINUTES=15
```

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/config/config.go backend-go/internal/config/config_test.go .env.example
git commit -m "chore: replace smtp config with turnstile and auth limit config"
```

---

### Task 4: AuthService 注册/登录（后端，TDD）

**Files:**
- Modify: `backend-go/internal/service/auth_service.go`
- Delete: `backend-go/internal/service/verification_code.go`
- Test: `backend-go/internal/service/auth_service_test.go`（重写）

- [ ] **Step 1: 重写 auth_service_test.go（先写失败测试）**

`backend-go/internal/service/auth_service_test.go` 整体替换为：

```go
package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/wanli/backend-go/internal/auth"
	"github.com/wanli/backend-go/internal/infrastructure/turnstile"
	"github.com/wanli/backend-go/internal/model"
)

// inMemoryUserRepo 是测试用的内存用户仓库。
type inMemoryUserRepo struct {
	users              map[string]*model.User
	updateProfileCalls int
	updatedNickname    string
	updatedAvatar      *string
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{users: make(map[string]*model.User)}
}

func (r *inMemoryUserRepo) Create(ctx context.Context, user *model.User) error {
	r.users[user.Email] = user
	return nil
}

func (r *inMemoryUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.users[email], nil
}

func (r *inMemoryUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (r *inMemoryUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	_, ok := r.users[email]
	return ok, nil
}

func (r *inMemoryUserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	r.updateProfileCalls++
	r.updatedNickname = nickname
	r.updatedAvatar = avatarURL
	for _, u := range r.users {
		if u.ID == id {
			u.Nickname = nickname
			u.AvatarURL = avatarURL
			return u, nil
		}
	}
	return nil, nil
}

// stubVerifier 固定返回 ok/err 的 Turnstile 校验器。
type stubVerifier struct {
	ok  bool
	err error
}

func (s stubVerifier) Verify(context.Context, string, string) (bool, error) {
	return s.ok, s.err
}

// stubLimiter 记录调用但不实际限制。
type stubLimiter struct {
	locked        bool
	failures      int
	resetCalls    int
}

func (s *stubLimiter) IsLocked(ctx context.Context, email string) (bool, error) { return s.locked, nil }
func (s *stubLimiter) RecordFailure(ctx context.Context, email string) error   { s.failures++; return nil }
func (s *stubLimiter) Reset(ctx context.Context, email string) error           { s.resetCalls++; return nil }

func newAuthTestSvc(repo *inMemoryUserRepo, verifier turnstile.Verifier, limiter LoginLimiter) *AuthService {
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	return NewAuthService(repo, jwtManager, verifier, limiter)
}

func TestAuthService_Register_CreatesUser(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	user, token, err := svc.Register(context.Background(), "a@b.c", "password123", "cf-token")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "a@b.c", user.Email)
	assert.NotEqual(t, "password123", user.HashedPassword)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte("password123")))
	assert.NotEmpty(t, token)
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	repo := newInMemoryUserRepo()
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c"}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, _, err := svc.Register(context.Background(), "a@b.c", "password123", "cf-token")
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestAuthService_Register_CaptchaFailed(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: false}, &stubLimiter{})

	_, _, err := svc.Register(context.Background(), "a@b.c", "password123", "bad-token")
	require.ErrorIs(t, err, ErrCaptchaFailed)
	assert.Len(t, repo.users, 0)
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newInMemoryUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c", HashedPassword: string(hashed)}
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	user, token, err := svc.Login(context.Background(), "a@b.c", "password123", "cf-token")
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", user.Email)
	assert.NotEmpty(t, token)
	assert.Equal(t, 1, limiter.resetCalls)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newInMemoryUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c", HashedPassword: string(hashed)}
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	_, _, err := svc.Login(context.Background(), "a@b.c", "wrong-password", "cf-token")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Equal(t, 1, limiter.failures)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := newInMemoryUserRepo()
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	_, _, err := svc.Login(context.Background(), "ghost@b.c", "password123", "cf-token")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Equal(t, 1, limiter.failures)
}

func TestAuthService_Login_Locked(t *testing.T) {
	repo := newInMemoryUserRepo()
	limiter := &stubLimiter{locked: true}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	_, _, err := svc.Login(context.Background(), "a@b.c", "password123", "cf-token")
	require.ErrorIs(t, err, ErrTooManyAttempts)
	assert.Equal(t, 0, limiter.failures)
}

func TestAuthService_UpdateProfileTrimsAndRejectsEmptyNickname(t *testing.T) {
	repo := newInMemoryUserRepo()
	user := &model.User{ID: uuid.New(), Email: "a@b.c", Nickname: "旧昵称"}
	repo.users[user.Email] = user
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), "   ", nil)
	assert.ErrorIs(t, err, ErrProfileInvalid)
	assert.Equal(t, 0, repo.updateProfileCalls)
}

func TestAuthService_UpdateProfileTrimsNicknameAndEmptyAvatar(t *testing.T) {
	repo := newInMemoryUserRepo()
	user := &model.User{ID: uuid.New(), Email: "a@b.c", Nickname: "旧昵称"}
	repo.users[user.Email] = user
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	empty := "  "
	updated, err := svc.UpdateProfile(context.Background(), user.ID, "  新昵称  ", &empty)
	require.NoError(t, err)
	assert.Equal(t, "新昵称", updated.Nickname)
	assert.Nil(t, updated.AvatarURL)
}

func TestAuthService_UpdateProfileRejectsTooLongNickname(t *testing.T) {
	repo := newInMemoryUserRepo()
	user := &model.User{ID: uuid.New(), Email: "a@b.c"}
	repo.users[user.Email] = user
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), strings.Repeat("很", 51), nil)
	assert.ErrorIs(t, err, ErrProfileInvalid)
}

func TestAuthService_UpdateProfileUserNotFound(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), "新昵称", nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/service/ -run TestAuthService -v`
Expected: 编译失败（`NewAuthService` 签名不匹配、`Register`/`Login` 不存在）。

- [ ] **Step 3: 重写 auth_service.go**

`backend-go/internal/service/auth_service.go` 整体替换为：

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/wanli/backend-go/internal/auth"
	"github.com/wanli/backend-go/internal/infrastructure/turnstile"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

var (
	ErrProfileInvalid     = errors.New("昵称不能为空且不超过 50 字")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrEmailTaken         = errors.New("该邮箱已被注册")
	ErrInvalidCredentials = errors.New("邮箱或密码错误")
	ErrCaptchaFailed      = errors.New("人机验证未通过，请重试")
	ErrTooManyAttempts    = errors.New("失败次数过多，请稍后再试")
)

// AuthService 处理认证相关业务。
type AuthService struct {
	userRepo     repository.UserRepository
	jwtManager   *auth.JWTManager
	turnstile    turnstile.Verifier
	loginLimiter LoginLimiter
}

// NewAuthService 创建 AuthService。
func NewAuthService(
	userRepo repository.UserRepository,
	jwtManager *auth.JWTManager,
	turnstile turnstile.Verifier,
	loginLimiter LoginLimiter,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtManager:   jwtManager,
		turnstile:    turnstile,
		loginLimiter: loginLimiter,
	}
}

// Register 创建新账号并签发 token。
func (s *AuthService) Register(ctx context.Context, email, password, turnstileToken string) (*model.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	ok, err := s.turnstile.Verify(ctx, turnstileToken, "")
	if err != nil || !ok {
		return nil, "", ErrCaptchaFailed
	}

	exists, err := s.userRepo.EmailExists(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("check email exists: %w", err)
	}
	if exists {
		return nil, "", ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:             uuid.New(),
		Email:          email,
		HashedPassword: string(hashed),
		Nickname:       generateNickname(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.jwtManager.Generate(user.ID.String(), user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

// Login 校验人机验证、限流与密码，签发 token。
func (s *AuthService) Login(ctx context.Context, email, password, turnstileToken string) (*model.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	ok, err := s.turnstile.Verify(ctx, turnstileToken, "")
	if err != nil || !ok {
		return nil, "", ErrCaptchaFailed
	}

	locked, err := s.loginLimiter.IsLocked(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("check login limit: %w", err)
	}
	if locked {
		return nil, "", ErrTooManyAttempts
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		if err := s.loginLimiter.RecordFailure(ctx, email); err != nil {
			return nil, "", fmt.Errorf("record login failure: %w", err)
		}
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		if err := s.loginLimiter.RecordFailure(ctx, email); err != nil {
			return nil, "", fmt.Errorf("record login failure: %w", err)
		}
		return nil, "", ErrInvalidCredentials
	}

	if err := s.loginLimiter.Reset(ctx, email); err != nil {
		return nil, "", fmt.Errorf("reset login failures: %w", err)
	}

	token, err := s.jwtManager.Generate(user.ID.String(), user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

func generateNickname() string {
	return "碗友_" + uuid.New().String()[:8]
}

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

- [ ] **Step 4: 删除 verification_code.go**

```bash
rm backend-go/internal/service/verification_code.go
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/jichi/wanli/backend-go && go test ./internal/service/ -v`
Expected: 全部 PASS。

- [ ] **Step 6: Commit**

```bash
git add backend-go/internal/service/auth_service.go backend-go/internal/service/auth_service_test.go
git commit -m "feat: replace email-code auth with password register/login"
```

---

### Task 5: Handler 与路由（后端）

**Files:**
- Modify: `backend-go/internal/handler/auth_handler.go`
- Modify: `backend-go/internal/server/server.go`

- [ ] **Step 1: 改 auth_handler.go**

`backend-go/internal/handler/auth_handler.go` 整体替换为：

```go
package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/service"
	"github.com/wanli/backend-go/pkg/apperror"
	"github.com/wanli/backend-go/pkg/response"
)

// userResponse 返回用户信息的统一响应结构。
func userResponse(u *model.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"email":      u.Email,
		"nickname":   u.Nickname,
		"avatar_url": u.AvatarURL,
	}
}

// AuthHandler 认证接口。
type AuthHandler struct {
	authService *service.AuthService
	siteKey     string
}

// NewAuthHandler 创建 AuthHandler。
func NewAuthHandler(authService *service.AuthService, siteKey string) *AuthHandler {
	return &AuthHandler{authService: authService, siteKey: siteKey}
}

type registerRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8,max=72"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

type loginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

// Register 邮箱+密码注册。
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.TurnstileToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			response.Fail(c, apperror.New("AUTH_EMAIL_TAKEN", err.Error(), http.StatusConflict))
		case errors.Is(err, service.ErrCaptchaFailed):
			response.Fail(c, apperror.New("CAPTCHA_FAILED", err.Error(), http.StatusBadRequest))
		default:
			response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "注册失败", http.StatusInternalServerError, err))
		}
		return
	}

	response.Ok(c, gin.H{"token": token, "user": userResponse(user)})
}

// Login 邮箱+密码登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithDetails(c, apperror.ErrValidation, err.Error())
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, req.TurnstileToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCaptchaFailed):
			response.Fail(c, apperror.New("CAPTCHA_FAILED", err.Error(), http.StatusBadRequest))
		case errors.Is(err, service.ErrTooManyAttempts):
			response.Fail(c, apperror.New("AUTH_TOO_MANY_ATTEMPTS", err.Error(), http.StatusTooManyRequests))
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Fail(c, apperror.New("AUTH_INVALID_CREDENTIALS", err.Error(), http.StatusUnauthorized))
		default:
			response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "登录失败", http.StatusInternalServerError, err))
		}
		return
	}

	response.Ok(c, gin.H{"token": token, "user": userResponse(user)})
}

// Me 获取当前登录用户信息。GET /api/v1/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, appErr := currentUserID(c)
	if appErr != nil {
		response.Fail(c, appErr)
		return
	}
	user, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(c, apperror.ErrResourceNotFound)
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "查询用户信息失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, userResponse(user))
}

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
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(c, apperror.ErrResourceNotFound)
			return
		}
		response.Fail(c, apperror.Wrap("INTERNAL_ERROR", "更新资料失败", http.StatusInternalServerError, err))
		return
	}
	response.Ok(c, userResponse(user))
}

// CaptchaWidget 返回内嵌 Turnstile 组件的页面（供 Android WebView 加载）。
func (h *AuthHandler) CaptchaWidget(c *gin.Context) {
	page := fmt.Sprintf(captchaPageTemplate, h.siteKey)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

const captchaPageTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>人机验证</title>
<style>body{margin:0;display:flex;justify-content:center;padding-top:16px;background:transparent;}</style>
</head>
<body>
<div id="turnstile-widget"></div>
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onTurnstileLoad" async defer></script>
<script>
function onTurnstileLoad() {
  turnstile.render('#turnstile-widget', {
    sitekey: '%s',
    callback: function(token) {
      if (window.AndroidBridge) { window.AndroidBridge.onTokenReceived(token); }
    },
    'error-callback': function(code) {
      if (window.AndroidBridge) { window.AndroidBridge.onError(code); }
    }
  });
}
</script>
</body>
</html>`
```

- [ ] **Step 2: 改 server.go**

`backend-go/internal/server/server.go`：

1. import 增加 `"github.com/wanli/backend-go/internal/infrastructure/turnstile"`。

2. 删除第 67-68 行：

```go
	codeStore := service.NewMemoryCodeStore(5 * time.Minute)
	authService := service.NewAuthService(userRepo, codeStore, jwtManager, 5*time.Minute)
```

替换为：

```go
	var turnstileVerifier turnstile.Verifier
	if cfg.Turnstile.SecretKey == "" {
		logger.Warn("TURNSTILE_SECRET_KEY 未配置，人机验证将恒失败（闭锁）")
		turnstileVerifier = turnstile.Stub{}
	} else {
		turnstileVerifier = turnstile.New(cfg.Turnstile.SecretKey)
	}

	var loginLimiter service.LoginLimiter
	lockout := time.Duration(cfg.Auth.LockoutMinutes) * time.Minute
	if rdb != nil {
		loginLimiter = service.NewRedisLoginLimiter(rdb, cfg.Auth.MaxFailures, lockout)
	} else {
		logger.Warn("redis not available, using in-memory login limiter")
		loginLimiter = service.NewMemoryLoginLimiter(cfg.Auth.MaxFailures, lockout)
	}

	authService := service.NewAuthService(userRepo, jwtManager, turnstileVerifier, loginLimiter)
```

3. `authHandler := handler.NewAuthHandler(authService)` 改为：

```go
	authHandler := handler.NewAuthHandler(authService, cfg.Turnstile.SiteKey)
```

4. `registerRoutes` 中删除 `v1.POST("/auth/email-code", authHandler.SendEmailCode)`，`v1.POST("/auth/login", authHandler.Login)` 前加一行：

```go
		v1.POST("/auth/register", authHandler.Register)
```

5. `registerRoutes` 函数开头（`v1 := r.Group("/api/v1")` 之前）注册验证码页面路由：

```go
	r.GET("/captcha", authHandler.CaptchaWidget)
```

- [ ] **Step 3: 全量校验后端编译与测试**

Run: `cd /home/jichi/wanli/backend-go && gofmt -l . && go vet ./... && go build ./... && go test ./...`
Expected: `gofmt` 无输出、`go vet` 无告警、build 成功、全部测试 PASS。

- [ ] **Step 4: Commit**

```bash
git add backend-go/internal/handler/auth_handler.go backend-go/internal/server/server.go
git commit -m "feat: add register/login handlers and captcha widget route"
```

---

### Task 6: 数据库迁移 000005

**Files:**
- Create: `backend-go/migrations/000005_obsolete_email_code_users.up.sql`
- Create: `backend-go/migrations/000005_obsolete_email_code_users.down.sql`

- [ ] **Step 1: 写 up 迁移**

`backend-go/migrations/000005_obsolete_email_code_users.up.sql`：

```sql
-- 作废通过邮箱验证码创建的存量开发账号：将 hashed_password 置为不可验证哨兵，
-- bcrypt 比较恒失败，账号无法再登录，但保留用户行以维持外键完整性。
-- 用户需重新注册（预发布阶段无真实用户）。
UPDATE users
SET hashed_password = '!code-era-obsolete',
    updated_at = CURRENT_TIMESTAMP;
```

- [ ] **Step 2: 写 down 迁移**

`backend-go/migrations/000005_obsolete_email_code_users.down.sql`：

```sql
-- 不可逆：原 hashed_password（验证码的 bcrypt）已被覆盖，无法恢复。
-- 如需还原请在执行 up 前备份 users 表。此迁移不做任何数据恢复操作。
```

- [ ] **Step 3: 对迁移到新库的空库执行验证（可执行性检查）**

Run: `cd /home/jichi/wanli/backend-go && go run ./cmd/migrate up`
Expected: 成功且无报错（若本地无数据库服务可跳过，仅确认 SQL 语法）。

- [ ] **Step 4: Commit**

```bash
git add backend-go/migrations/000005_obsolete_email_code_users.up.sql backend-go/migrations/000005_obsolete_email_code_users.down.sql
git commit -m "feat: migration to obsolete email-code era users"
```

---

### Task 7: docker-compose 配置

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: 更新 backend 服务环境变量**

`docker-compose.yml` 中删除：

```yaml
      SMTP_MOCK: ${SMTP_MOCK:-true}
      SMTP_HOST: ${SMTP_HOST:-}
      SMTP_PORT: ${SMTP_PORT:-587}
      SMTP_USER: ${SMTP_USER:-}
      SMTP_PASS: ${SMTP_PASS:-}
```

新增：

```yaml
      TURNSTILE_SITE_KEY: ${TURNSTILE_SITE_KEY:-}
      TURNSTILE_SECRET_KEY: ${TURNSTILE_SECRET_KEY:-}
      AUTH_MAX_FAILURES: ${AUTH_MAX_FAILURES:-5}
      AUTH_LOCKOUT_MINUTES: ${AUTH_LOCKOUT_MINUTES:-15}
```

- [ ] **Step 2: 校验配置语法**

Run: `cd /home/jichi/wanli && docker compose config --quiet`（无 Docker 时跳过）
Expected: 无报错。

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "chore: replace smtp envs with turnstile and auth limit envs"
```

---

### Task 8: api-contract 文档

**Files:**
- Modify: `docs/api-contract.md`

- [ ] **Step 1: 重写 2.1 用户认证节**

`docs/api-contract.md` 中从 `#### 1. 发送邮箱验证码` 到 2.2 节之前整段替换为：

```markdown
#### 1. 注册（邮箱 + 密码 + Turnstile）
- **请求**：`POST /api/v1/auth/register`
- **Body**:
  ```json
  {
    "email": "user@example.com",
    "password": "supersecret123",
    "turnstile_token": "0.XXXXXXX..."
  }
  ```
- **校验**：`email` 合法邮箱；`password` 长度 8~72；`turnstile_token` 非空且通过 Cloudflare siteverify；邮箱已存在返回 `409 AUTH_EMAIL_TAKEN`。
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "token": "JWT_BEARER_TOKEN_HERE",
      "user": {
        "id": "e93817f5-2f84-482f-b05b-86d1dfcbf81c",
        "email": "user@example.com",
        "nickname": "碗友_5821",
        "avatar_url": null
      }
    }
  }
  ```

#### 2. 登录
- **请求**：`POST /api/v1/auth/login`
- **Body**:
  ```json
  {
    "email": "user@example.com",
    "password": "supersecret123",
    "turnstile_token": "0.XXXXXXX..."
  }
  ```
- **校验**：邮箱或密码错误统一返回 `401 AUTH_INVALID_CREDENTIALS`（不区分账号是否存在）；连续失败达 `AUTH_MAX_FAILURES`（默认 5）次锁定 `AUTH_LOCKOUT_MINUTES`（默认 15）分钟，返回 `429 AUTH_TOO_MANY_ATTEMPTS`。
- **返回**：`200 OK`，结构同注册。

> **人机验证页面**：Android 端在 `GET /captcha` 加载内嵌 Turnstile 组件的页面，成功后通过 WebView 的 `AndroidBridge` 拿到 token。token 为一次性，校验失败后需重新加载页面获取新 token。
```

- [ ] **Step 2: 错误码表新增 4 行**

`docs/api-contract.md` 核心错误码表末尾追加：

```markdown
| `AUTH_EMAIL_TAKEN` | 邮箱已被注册 | 注册时邮箱已存在 |
| `AUTH_INVALID_CREDENTIALS` | 邮箱或密码错误 | 登录校验失败（不区分账号是否存在） |
| `CAPTCHA_FAILED` | 人机验证未通过 | 注册/登录时 Turnstile 校验失败 |
| `AUTH_TOO_MANY_ATTEMPTS` | 登录失败次数超限，账号临时锁定 | 连续失败达限流阈值 |
```

- [ ] **Step 3: Commit**

```bash
git add docs/api-contract.md
git commit -m "docs: update auth contract to password + turnstile"
```

---

### Task 9: Android 数据层（TDD）

**Files:**
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/data/remote/ApiService.kt`
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/data/repository/AuthRepository.kt`
- Test: `frontend-android/app/src/test/kotlin/com/wanli/android/data/repository/AuthRepositoryTest.kt`

- [ ] **Step 1: 改 ApiService.kt**

1. 删除 `sendEmailCode` 方法：

```kotlin
    @POST("auth/email-code")
    suspend fun sendEmailCode(@Body request: SendEmailCodeRequest): BaseResponse
```

2. 删除 `SendEmailCodeRequest` 数据类，并把 `LoginRequest` 替换为：

```kotlin
@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
    @SerialName("turnstile_token") val turnstileToken: String
)

@Serializable
data class RegisterRequest(
    val email: String,
    val password: String,
    @SerialName("turnstile_token") val turnstileToken: String
)
```

3. 在 `login` 方法上方新增：

```kotlin
    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): LoginResponse
```

- [ ] **Step 2: 改 AuthRepository.kt**

`AuthRepository.kt` 中：

1. 删除 `sendEmailCode` 方法与 `SendEmailCodeRequest` import。
2. 新增 `register`，并改写 `login`：

```kotlin
    suspend fun register(email: String, password: String, turnstileToken: String): ApiResult<UserDto> {
        return try {
            val response = apiService.register(RegisterRequest(email, password, turnstileToken))
            if (response.success && response.data != null) {
                tokenDataStore.saveToken(
                    token = response.data.token,
                    userId = response.data.user.id,
                    email = response.data.user.email
                )
                ApiResult.Success(response.data.user)
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        } catch (e: Exception) {
            ApiResult.Error(
                code = "NETWORK_ERROR",
                message = "网络错误，请稍后重试"
            )
        }
    }

    suspend fun login(email: String, password: String, turnstileToken: String): ApiResult<UserDto> {
        return try {
            val response = apiService.login(LoginRequest(email, password, turnstileToken))
            if (response.success && response.data != null) {
                tokenDataStore.saveToken(
                    token = response.data.token,
                    userId = response.data.user.id,
                    email = response.data.user.email
                )
                ApiResult.Success(response.data.user)
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        } catch (e: Exception) {
            ApiResult.Error(
                code = "NETWORK_ERROR",
                message = "网络错误，请稍后重试"
            )
        }
    }
```

3. import 把 `com.wanli.android.data.remote.SendEmailCodeRequest` 替换为 `com.wanli.android.data.remote.RegisterRequest`。

- [ ] **Step 3: 更新 AuthRepositoryTest.kt**

`AuthRepositoryTest.kt` 整体替换为：

```kotlin
package com.wanli.android.data.repository

import com.wanli.android.data.local.TokenDataStore
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.RegisterRequest
import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import io.mockk.verify
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.IOException

class AuthRepositoryTest {

    private lateinit var apiService: ApiService
    private lateinit var tokenDataStore: TokenDataStore
    private lateinit var repository: AuthRepository

    @Before
    fun setup() {
        apiService = mockk()
        tokenDataStore = mockk()
        every { tokenDataStore.isLoggedIn } returns flowOf(false)
        repository = AuthRepository(apiService, tokenDataStore)
    }

    @Test
    fun `login returns network error instead of throwing when api call fails`() = runTest {
        coEvery { apiService.login(any()) } throws IOException("timeout")

        val result = repository.login("a@b.c", "password123", "cf-token")

        assertTrue(result is ApiResult.Error)
        result as ApiResult.Error
        assertEquals("NETWORK_ERROR", result.code)
        assertEquals("网络错误，请稍后重试", result.message)
    }

    @Test
    fun `register sends email password and turnstile token`() = runTest {
        coEvery { apiService.register(any()) } returns mockLoginResponse()

        repository.register("a@b.c", "password123", "cf-token")

        verify { apiService.register(match<RegisterRequest> {
            it.email == "a@b.c" && it.password == "password123" && it.turnstileToken == "cf-token"
        }) }
    }
}
```

并在文件内补一个 helper（`mockLoginResponse` 需要返回 `LoginResponse`）：

```kotlin
import com.wanli.android.data.remote.LoginData
import com.wanli.android.data.remote.LoginResponse
import com.wanli.android.data.remote.UserDto

private fun mockLoginResponse() = LoginResponse(
    success = true,
    data = LoginData(
        token = "jwt-token",
        user = UserDto(id = "u1", email = "a@b.c")
    )
)
```

- [ ] **Step 4: 运行 Android 单测**

Run（需 JDK 17 + Android SDK）：`cd /home/jichi/wanli/frontend-android && ./gradlew testDebugUnitTest`
Expected: `AuthRepositoryTest` 全部 PASS。（当前环境无 JDK 时，将本步骤留到有 Java 的机器执行。）

- [ ] **Step 5: Commit**

```bash
git add frontend-android/app/src/main/kotlin/com/wanli/android/data/remote/ApiService.kt frontend-android/app/src/main/kotlin/com/wanli/android/data/repository/AuthRepository.kt frontend-android/app/src/test/kotlin/com/wanli/android/data/repository/AuthRepositoryTest.kt
git commit -m "feat(android): switch auth data layer to password + turnstile"
```

---

### Task 10: Android serverRootUrl 注入

**Files:**
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/di/NetworkModule.kt`

- [ ] **Step 1: 新增 provider**

`NetworkModule.kt` import 增加 `javax.inject.Named`，并在类内新增：

```kotlin
    @Provides
    @Singleton
    @Named("serverRootUrl")
    fun provideServerRootUrl(baseUrl: String): String {
        return baseUrl.trimEnd('/').removeSuffix("/api/v1")
    }
```

- [ ] **Step 2: 编译确认**

Run: `cd /home/jichi/wanli/frontend-android && ./gradlew compileDebugKotlin`
Expected: 编译通过（无 JDK 环境时留到有 Java 的机器执行）。

- [ ] **Step 3: Commit**

```bash
git add frontend-android/app/src/main/kotlin/com/wanli/android/di/NetworkModule.kt
git commit -m "feat(android): provide server root url for captcha widget"
```

---

### Task 11: Android TurnstileWidget（WebView）

**Files:**
- Create: `frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/TurnstileWidget.kt`

- [ ] **Step 1: 新建组件**

`frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/TurnstileWidget.kt`：

```kotlin
package com.wanli.android.presentation.auth

import android.annotation.SuppressLint
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView

/**
 * 用 WebView 加载后端托管的 Turnstile 组件页，并把 token/错误回调给 Compose 层。
 * Cloudflare 官方对原生 App 的标准接入方式即 WebView（无原生 SDK）。
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TurnstileWidget(
    widgetUrl: String,
    onTokenReceived: (String) -> Unit,
    onError: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                addJavascriptInterface(
                    object {
                        @JavascriptInterface
                        fun onTokenReceived(token: String) = onTokenReceived(token)

                        @JavascriptInterface
                        fun onError(code: String) = onError(code)
                    },
                    "AndroidBridge"
                )
                webViewClient = WebViewClient()
                loadUrl(widgetUrl)
            }
        }
    )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/TurnstileWidget.kt
git commit -m "feat(android): add turnstile webview widget"
```

---

### Task 12: Android LoginViewModel（TDD）

**Files:**
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/LoginViewModel.kt`
- Test: `frontend-android/app/src/test/kotlin/com/wanli/android/presentation/auth/LoginViewModelTest.kt`

- [ ] **Step 1: 重写 LoginViewModel.kt**

`LoginViewModel.kt` 整体替换为：

```kotlin
package com.wanli.android.presentation.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject
import javax.inject.Named

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    @Named("serverRootUrl") private val serverRootUrl: String
) : ViewModel() {

    private val _uiState = MutableStateFlow<LoginUiState>(LoginUiState.Idle)
    val uiState: StateFlow<LoginUiState> = _uiState

    val captchaUrl: String
        get() = "$serverRootUrl/captcha"

    fun register(email: String, password: String, turnstileToken: String) {
        viewModelScope.launch {
            _uiState.value = LoginUiState.Loading
            _uiState.value = when (val result = authRepository.register(email, password, turnstileToken)) {
                is ApiResult.Success -> LoginUiState.LoggedIn
                is ApiResult.Error -> LoginUiState.Error(result.message)
                is ApiResult.Loading -> LoginUiState.Loading
            }
        }
    }

    fun login(email: String, password: String, turnstileToken: String) {
        viewModelScope.launch {
            _uiState.value = LoginUiState.Loading
            _uiState.value = when (val result = authRepository.login(email, password, turnstileToken)) {
                is ApiResult.Success -> LoginUiState.LoggedIn
                is ApiResult.Error -> LoginUiState.Error(result.message)
                is ApiResult.Loading -> LoginUiState.Loading
            }
        }
    }
}

sealed class LoginUiState {
    data object Idle : LoginUiState()
    data object Loading : LoginUiState()
    data object LoggedIn : LoginUiState()
    data class Error(val message: String) : LoginUiState()
}
```

- [ ] **Step 2: 重写 LoginViewModelTest.kt**

`LoginViewModelTest.kt` 整体替换为：

```kotlin
package com.wanli.android.presentation.auth

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.UserDto
import com.wanli.android.data.repository.AuthRepository
import io.mockk.coEvery
import io.mockk.mockk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var authRepository: AuthRepository

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        authRepository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `login success moves to LoggedIn state`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.login("a@b.c", "password123", "cf-token")

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
    }

    @Test
    fun `login network error surfaces error state with message`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Error("NETWORK_ERROR", "网络错误，请稍后重试")

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.login("a@b.c", "password123", "cf-token")

        val state = viewModel.uiState.value
        assertTrue(state is LoginUiState.Error)
        assertEquals("网络错误，请稍后重试", (state as LoginUiState.Error).message)
    }

    @Test
    fun `register success moves to LoggedIn state`() = runTest {
        coEvery { authRepository.register("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.register("a@b.c", "password123", "cf-token")

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
    }

    @Test
    fun `captcha url derives from server root`() {
        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        assertEquals("http://10.0.2.2:8081/captcha", viewModel.captchaUrl)
    }
}
```

- [ ] **Step 3: 运行 Android 单测**

Run（需 JDK 17）：`cd /home/jichi/wanli/frontend-android && ./gradlew testDebugUnitTest --tests "com.wanli.android.presentation.auth.LoginViewModelTest"`
Expected: 全部 PASS。（无 JDK 环境时留到有 Java 的机器执行。）

- [ ] **Step 4: Commit**

```bash
git add frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/LoginViewModel.kt frontend-android/app/src/test/kotlin/com/wanli/android/presentation/auth/LoginViewModelTest.kt
git commit -m "feat(android): login viewmodel password + turnstile"
```

---

### Task 13: Android LoginScreen UI

**Files:**
- Modify: `frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/LoginScreen.kt`

- [ ] **Step 1: 重写 LoginScreen.kt**

`LoginScreen.kt` 整体替换为：

```kotlin
package com.wanli.android.presentation.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.SoupKitchen
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.SageSoft

private enum class LoginMode { Login, Register }

@Composable
fun LoginScreen(
    viewModel: LoginViewModel = androidx.hilt.navigation.compose.hiltViewModel(),
    onLoggedIn: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var turnstileToken by remember { mutableStateOf<String?>(null) }
    var mode by remember { mutableStateOf(LoginMode.Login) }
    var widgetEpoch by remember { mutableStateOf(0) }

    LaunchedEffect(uiState) {
        if (uiState is LoginUiState.LoggedIn) {
            onLoggedIn()
        }
    }

    // 登录/注册失败后，强制刷新验证码（token 一次性）
    LaunchedEffect(uiState) {
        if (uiState is LoginUiState.Error) {
            turnstileToken = null
            widgetEpoch++
        }
    }

    fun switchMode() {
        mode = if (mode == LoginMode.Login) LoginMode.Register else LoginMode.Login
        turnstileToken = null
        widgetEpoch++
    }

    val submit: () -> Unit = {
        turnstileToken?.let { token ->
            if (mode == LoginMode.Login) {
                viewModel.login(email, password, token)
            } else {
                viewModel.register(email, password, token)
            }
        }
    }

    Scaffold { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .imePadding()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(200.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Box(
                    contentAlignment = Alignment.Center,
                    modifier = Modifier
                        .size(96.dp)
                        .clip(CircleShape)
                        .background(SageSoft)
                ) {
                    Icon(
                        imageVector = Icons.Outlined.SoupKitchen,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(48.dp)
                    )
                }

                Text(
                    text = "碗里",
                    style = MaterialTheme.typography.headlineMedium,
                    modifier = Modifier.padding(top = 16.dp)
                )
                Text(
                    text = "把视频变成能下锅的菜谱",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }

            WanLiCard(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.cardContentPadding()) {
                    OutlinedTextField(
                        value = email,
                        onValueChange = { email = it },
                        label = { Text("邮箱") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it },
                        label = { Text("密码") },
                        singleLine = true,
                        visualTransformation = PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    )

                    Spacer(modifier = Modifier.height(16.dp))

                    key(widgetEpoch) {
                        TurnstileWidget(
                            widgetUrl = viewModel.captchaUrl,
                            onTokenReceived = { turnstileToken = it },
                            onError = { turnstileToken = null },
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(80.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            Button(
                onClick = submit,
                modifier = Modifier.fillMaxWidth(),
                shape = MaterialTheme.shapes.large,
                enabled = email.isNotBlank() &&
                    password.isNotBlank() &&
                    turnstileToken != null &&
                    uiState !is LoginUiState.Loading
            ) {
                Text(if (mode == LoginMode.Login) "登录" else "注册")
            }

            TextButton(onClick = ::switchMode) {
                Text(if (mode == LoginMode.Login) "没有账号？去注册" else "已有账号？去登录")
            }

            if (uiState is LoginUiState.Error) {
                Spacer(modifier = Modifier.height(16.dp))
                StatusBanner(type = StatusBannerType.ERROR) {
                    Text(
                        text = (uiState as LoginUiState.Error).message,
                        style = MaterialTheme.typography.bodyMedium
                    )
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "登录即表示同意用户协议与隐私政策",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth()
            )
        }
    }
}
```

- [ ] **Step 2: 编译确认**

Run: `cd /home/jichi/wanli/frontend-android && ./gradlew compileDebugKotlin`
Expected: 编译通过（无 JDK 环境时留到有 Java 的机器执行）。

- [ ] **Step 3: Commit**

```bash
git add frontend-android/app/src/main/kotlin/com/wanli/android/presentation/auth/LoginScreen.kt
git commit -m "feat(android): login screen with password and turnstile"
```

---

### Task 14: 文档同步与全量验证

**Files:**
- Modify: `docs/permissions.md`
- Modify: `docs/mvp.md`
- Modify: `docs/deploy-home-tunnel.md`
- Modify: `docs/development-tasks.md`

- [ ] **Step 1: permissions.md**

`docs/permissions.md` 第 11 行：

```markdown
2. **登录用户 (User)**：已通过邮箱验证码登录的普通注册用户。
```

改为：

```markdown
2. **登录用户 (User)**：已通过「邮箱 + 密码 + Turnstile 人机验证」注册并登录的普通注册用户。
```

- [ ] **Step 2: mvp.md**

`docs/mvp.md` 第 8 行：

```markdown
- **注册与登录**：使用邮箱验证码方式进行注册与登录（后端验证码使用 mock 或 SMTP 发送）。
```

改为：

```markdown
- **注册与登录**：使用「邮箱 + 密码 + Cloudflare Turnstile 人机验证」进行注册与登录，注册/登录分离，附带登录失败限流（默认 5 次锁 15 分钟）。
```

- [ ] **Step 3: deploy-home-tunnel.md**

`docs/deploy-home-tunnel.md` 第 67 行：

```markdown
- 邮箱验证码默认走 mock（打印在后端日志）；要发真实邮件需配置 SMTP 且 `SMTP_MOCK=false`
```

改为：

```markdown
- 注册/登录需配置 `TURNSTILE_SITE_KEY` 与 `TURNSTILE_SECRET_KEY`（Cloudflare 后台创建 widget）；开发期可用官方测试 key `1x00000000000000000000AA`（恒通过），上线前替换为真实 key 并在 widget 中添加生产域名
```

- [ ] **Step 4: development-tasks.md**

`docs/development-tasks.md` 中以下三行改为 `- [ ]`（验证码方案已废弃）：

```markdown
- [x] 实现邮箱验证码发送（SMTP/mock）。
- [x] 实现邮箱验证码登录/注册，返回 JWT。
- [x] 登录/注册页面（邮箱 + 验证码）。
```

改为：

```markdown
- [x] ~~实现邮箱验证码发送（SMTP/mock）~~ → 废弃，改为「邮箱+密码+Turnstile」
- [x] ~~实现邮箱验证码登录/注册，返回 JWT~~ → 改为密码注册/登录（注册/登录分离）
- [x] 登录/注册页面（邮箱 + 密码 + Turnstile）
```

- [ ] **Step 5: 全量后端校验**

Run: `cd /home/jichi/wanli/backend-go && gofmt -l . && go vet ./... && go build ./... && go test ./...`
Expected: `gofmt` 无输出、vet 无告警、build 成功、全部测试 PASS。

- [ ] **Step 6: 全量 Android 校验**（需 JDK 17 机器）

Run: `cd /home/jichi/wanli/frontend-android && ./gradlew clean testDebugUnitTest assembleDebug`
Expected: BUILD SUCCESSFUL，无 Kotlin Warning。

- [ ] **Step 7: Commit**

```bash
git add docs/permissions.md docs/mvp.md docs/deploy-home-tunnel.md docs/development-tasks.md
git commit -m "docs: sync auth docs to password + turnstile"
```

---

## 验证清单（完成定义）

- [ ] `go fmt ./...`、`go vet ./...`、`go build ./...`、`go test ./...` 全绿
- [ ] Android 单测与 Release 编译通过（需 JDK 17 机器）
- [ ] 手工冒烟：`POST /api/v1/auth/register` → `POST /api/v1/auth/login` → 连续错密码 5 次返回 429
- [ ] 迁移 000005 可执行（up/down）
- [ ] `docs/api-contract.md`、`docs/permissions.md` 等已同步
