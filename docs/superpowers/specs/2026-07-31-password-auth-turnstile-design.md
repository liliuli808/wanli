# 密码登录 + Cloudflare Turnstile 设计文档

日期：2026-07-31
状态：已确认，待实施

---

## 一、背景与决策

原认证方案为「邮箱验证码登录/注册」（一键式，mock/SMTP 发送验证码）。现决定整体放弃邮箱验证码，改为「邮箱 + 密码」认证，并接入 Cloudflare Turnstile 做注册与登录两侧的人机验证。

已与用户确认的决策：

1. **注册与登录分离**：`POST /auth/register` 与 `POST /auth/login` 两个独立接口，防止邮箱被抢注。
2. **注册 + 登录都校验 Turnstile**：两端都提交 turnstile_token，后端统一走 siteverify。
3. **作废存量开发用户**：现有通过验证码创建的账号（`hashed_password` 是验证码的 bcrypt）直接作废，用户需重新注册。
4. **密码强度**：最少 8 位（bcrypt 上限 72 字节以内）。

**MVP 边界**：
- 不做「忘记密码 / 重置密码」流程（无邮件/短信通道）。
- Turnstile 走 Cloudflare 全球节点，国内网络偶发可达性问题为已知风险；后续如体验差可替换为国内验证码服务。

---

## 二、后端接口契约

### 2.1 注册

- **请求**：`POST /api/v1/auth/register`
- **Body**：
  ```json
  {
    "email": "user@example.com",
    "password": "supersecret123",
    "turnstile_token": "0.XXXXXXX..."
  }
  ```
- **校验**：`email` 合法邮箱；`password` 长度 8~72；`turnstile_token` 非空且 siteverify 通过；邮箱已存在返回 409。
- **返回**：`200 OK`，`data` 为 `{ token, user }`（结构同现有登录响应）。

### 2.2 登录

- **请求**：`POST /api/v1/auth/login`
- **Body**：
  ```json
  {
    "email": "user@example.com",
    "password": "supersecret123",
    "turnstile_token": "0.XXXXXXX..."
  }
  ```
- **校验**：邮箱或密码错误统一返回 401 `AUTH_INVALID_CREDENTIALS`（不区分账号是否存在）；登录失败计数达上限返回 429。
- **返回**：`200 OK`，`data` 为 `{ token, user }`。

### 2.3 删除的接口

- `POST /api/v1/auth/email-code` 删除。

### 2.4 新增错误码

| 错误码 | 含义 | HTTP |
| :--- | :--- | :---: |
| `AUTH_EMAIL_TAKEN` | 邮箱已被注册 | 409 |
| `AUTH_INVALID_CREDENTIALS` | 邮箱或密码错误 | 401 |
| `CAPTCHA_FAILED` | 人机验证未通过 | 400 |
| `AUTH_TOO_MANY_ATTEMPTS` | 登录失败次数超限，账号临时锁定 | 429 |

---

## 三、后端实现

### 3.1 Turnstile 校验器

`internal/infrastructure/turnstile/turnstile.go`（新建）：

```go
type Verifier interface {
    Verify(ctx context.Context, token, remoteIP string) (bool, error)
}

type Config struct {
    SecretKey string
    SiteKey   string // 备用校验字段，可留空
    Timeout   time.Duration
}

// SiteVerifyRequest/Response 对应 challenges.cloudflare.com/turnstile/v0/siteverify
```

- 请求：`POST https://challenges.cloudflare.com/turnstile/v0/siteverify`，表单字段 `secret`、`response`、可选 `remoteip`。
- 返回 `{"success": true}` 表示通过；HTTP 超时 5 秒。
- 测试：`httptest` 起假 siteverify 服务，覆盖 success / failure / HTTP 错误。

### 3.2 AuthService 改造

`internal/service/auth_service.go`：

- 删除 `SendEmailCode`、`GenerateCode`、`MemoryCodeStore` 相关引用；删除 `verification_code.go`。
- `Register(ctx, email, password, token)`：
  1. `turnstile.Verify` 失败 → `CAPTCHA_FAILED`。
  2. `userRepo.GetByEmail` 已存在 → `AUTH_EMAIL_TAKEN`。
  3. `bcrypt.GenerateFromPassword(password, DefaultCost)` 入库，返回 `{ user, token }`。
- `Login(ctx, email, password, token)`：
  1. `turnstile.Verify` 失败 → `CAPTCHA_FAILED`。
  2. 校验限流：`loginLimiter.IsLocked(email)` → 429。
  3. `GetByEmail` 不存在 或 `bcrypt.CompareHashAndPassword` 失败 → 记录失败并返回 401。
  4. 成功 → 清除失败计数，返回 `{ user, token }`。

`internal/handler/auth_handler.go`：新增 `Register` / `Login` handler，删除 `SendEmailCode`。

### 3.3 登录限流

`internal/service/login_limiter.go`（新建）：

- Redis 实现：key `auth:fail:{email}`，`INCR` + `EXPIRE 15min`；失败次数 ≥ 5 返回 `IsLocked = true`。
- 成功登录时 `DEL` 该 key。
- 参数化：`MaxFailures`（默认 5）、`LockoutDuration`（默认 15 分钟）。
- 接口化以便单测用 `miniredis`（已在依赖中）或内存实现。
- Redis 客户端为 nil 时（开发环境无 Redis）回退到内存实现，行为等价，进程重启即重置计数。

### 3.4 配置

`internal/config/config.go`：

- 新增 `TurnstileConfig { SiteKey, SecretKey string }`，读取 `TURNSTILE_SITE_KEY` / `TURNSTILE_SECRET_KEY`。
- 删除 `SMTPConfig`。
- 新增 `AUTH_MAX_FAILURES`（默认 5）、`AUTH_LOCKOUT_MINUTES`（默认 15）。

### 3.5 服务组装

`internal/server/server.go`：

- 删除 `service.NewMemoryCodeStore` 组装。
- 按 `cfg.Turnstile.SecretKey` 构造 `turnstile.Verifier`（secret 为空时构造一个恒返回 false 的 stub，阻止误用）。
- 按 Redis 客户端可用性构造 `LoginLimiter`。

### 3.6 数据库迁移

`migrations/000005_obsolete_email_code_users.up.sql`：

- 将存量用户的 `hashed_password` 置为不可验证哨兵 `'!code-era-obsolete'`，使 bcrypt 比较恒失败，账号作废但保留历史数据（评论/记录外键不悬空）。
- Down：无法恢复原 hash，`down.sql` 为注明不可逆的空操作。

---

## 四、Android 客户端

### 4.1 依赖与配置

- 新增 `com.cloudflare:turnstile-sdk-android`（Maven Central）。
- `build.gradle` 通过 `buildConfigField` 注入 `TURNSTILE_SITE_KEY`（公钥，可内嵌；来源 `gradle.properties` / local 配置，绝不含 secret）。

### 4.2 UI 与状态

`LoginScreen`：
- 邮箱 + 密码（`PasswordVisualTransformation`）两个输入框。
- 「登录 / 注册」模式切换（SegmentedButton 或 Tab）。
- Turnstile widget（managed 模式）嵌入卡片内；`onTokenReceived` 回调保存 token，token 为空时提交按钮禁用。
- 移除「发送验证码」行与 `CodeSent` 提示。

`LoginViewModel`：
- 删除 `sendCode(email)` 与 `LoginUiState.CodeSent`。
- 新增 `register(email, password, turnstileToken)`；`login` 改为 `login(email, password, turnstileToken)`。
- 状态：`Idle / Loading / LoggedIn / Error`。

### 4.3 数据层

- `ApiService`：删除 `sendEmailCode`；`LoginRequest` 改为 `{ email, password, turnstileToken }`；新增 `register(RegisterRequest)`。
- `AuthRepository`：删除 `sendEmailCode`；新增 `register`；`login` 改签名。
- `TokenDataStore` 不变。

### 4.4 测试

- 更新 `LoginViewModelTest`：登录/注册成功、失败、Turnstile token 缺失时的状态。
- 更新 `AuthRepositoryTest`：新请求体与响应解析。
- Turnstile SDK 不进入 JVM 单测（仅编译依赖）。

---

## 五、文档同步

| 文件 | 改动 |
| :--- | :--- |
| `docs/api-contract.md` | 2.1 重写；错误码表加 4 个新码、删验证码相关 |
| `docs/permissions.md` | 「已通过邮箱验证码登录」改为「已通过邮箱+密码登录」 |
| `docs/mvp.md` | 注册登录描述更新 |
| `docs/user-flows.md` | 认证流程更新 |
| `docs/data-model.md` | 如涉及认证措辞同步更新 |
| `docs/deploy-home-tunnel.md` | 删除邮箱验证码 mock 相关说明 |
| `.env.example` / `docker-compose.yml` | 删 SMTP_*，加 TURNSTILE_* 与 AUTH_* |

---

## 六、安全与合规

- `TURNSTILE_SECRET_KEY` 仅存后端环境变量，禁止入库/入日志；site key 属公开值，可入 Android BuildConfig。
- 登录失败响应统一 401，不泄露账号是否存在。
- 密码 bcrypt 哈希存储，不记录明文。
- 限流在 Turnstile 通过之后、密码校验之前执行。

---

## 七、测试与验收（DoD）

- Go：`go fmt ./...`、`go vet ./...`、`go build ./...`、`go test ./...` 全绿。
  - turnstile verifier 单测（httptest）。
  - auth service register/login 单测（成功、重复邮箱、错密码、锁定）。
  - login limiter 单测（miniredis）。
- Android：`gradlew assembleDebug` 与相关单测通过；Kotlin 无 Warning。
- 手工验证：注册→登出→登录→连续错密码 5 次触发 429。

---

## 八、风险

| 风险 | 影响 | 缓解 |
| :--- | :--- | :--- |
| Turnstile 国内可达性 | 部分用户无法登录/注册 | 已知风险，接受；可后续切换国内服务商 |
| Turnstile SDK 体积/WebView 兼容性 | 包体积与低端机兼容 | managed 模式省去自绘；SDK 版本锁定 |
| 存量用户作废 | 旧开发账号需重注册 | 预发布阶段无真实用户，接受 |
| 无找回密码 | 用户忘记密码无法自助恢复 | MVP 边界，明确不做 |
