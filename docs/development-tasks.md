# “碗里” (WanLi) - 分阶段垂直切片开发任务书 (development-tasks.md)

本文档将 MVP 开发拆分为可独立交付、可验证的垂直切片。每个切片均包含后端（Go）与前端（Android）工作，完成后应可端到端验证。

---

## 阶段 0：项目骨架与基础设施

**目标**：建立可运行的 Go 后端与 Android 项目骨架，完成数据库迁移与统一错误处理。

### 后端
- [x] 创建 Go 项目结构（`cmd/api`、`cmd/migrate`、`internal/` 分层）。
- [x] 配置 `golang-migrate` 与 PostgreSQL 连接。
- [x] 实现统一 JSON 响应包装（`success/data/error`）。
- [x] 实现 `AuthMiddleware`（JWT 提取、游客放行、登录拦截）。
- [x] 实现 `OwnershipMiddleware` 工厂（支持不同资源类型的 owner 校验）。
- [x] 配置 Redis 连接（任务状态机、限流）。
- [x] 配置 S3/MinIO 对象存储客户端。

### Android
- [x] 创建 Android Studio 项目（Kotlin + Jetpack Compose）。
- [x] 配置 MVVM + Clean Architecture 模块（`data`、`domain`、`presentation`）。
- [x] 集成 Retrofit、OkHttp、Room、Coil、Hilt/Koin。
- [x] 实现统一错误处理与 Loading/Empty 状态组件。

### 验收
- [ ] `go build` 通过，`go test ./...` 通过（需本地安装 Go 后执行）。
- [ ] Android `./gradlew assembleDebug` 通过（需本地安装 JDK 17 + Android SDK 后执行）。

---

## 阶段 1：账号、个人厨房与忌口

**目标**：用户可以注册/登录，并配置自己的厨房与忌口。

### 后端
- [x] ~~实现邮箱验证码发送（SMTP/mock）~~ → 废弃，改为「邮箱+密码+Turnstile」
- [x] ~~实现邮箱验证码登录/注册，返回 JWT~~ → 改为密码注册/登录（注册/登录分离）
- [x] 实现 `GET /api/v1/me/kitchen` 与 `PUT /api/v1/me/kitchen`。
- [x] 实现 `GET /api/v1/me/dietary-restrictions` 与 `PUT /api/v1/me/dietary-restrictions`。
- [ ] 编写用户、厨房配置、饮食限制的 Repository 单元测试。

### Android
- [x] 登录/注册页面（邮箱 + 密码 + Turnstile）。
- [x] “我的厨房”页面（勾选已有调料、厨具）。
- [x] “忌口设置”页面（过敏源、不吃、医疗忌口）。
- [x] 本地缓存用户 Token 与厨房配置（Room）。

### 验收
- [x] 游客无法访问“我的厨房”。
- [x] 登录后可完整配置并持久化。

---

## 阶段 2：视频提交、字幕拉取与 AI 解析

**目标**：用户可以粘贴 B 站链接，后端完成去重、字幕拉取、AI 结构化解析。

### 后端
- [x] 实现 B 站 BVID 提取与校验。
- [x] 实现 B 站 CC/AI 字幕获取服务；无可用字幕或字幕拉取失败时立即终止，简介/评论仅作补充。
- [x] 实现 `POST /api/v1/video-imports` 与 `GET /api/v1/video-imports/:task_id`。
- [x] 实现基于 Redis 的解析任务状态机（PENDING → FETCHING → PARSING → COMPLETED/FAILED）。
- [x] 集成 AI 解析（按 `ai-parsing.md` Schema），生成待校对草稿版本。
- [x] 编写字幕获取、BVID 校验、任务状态机的单元测试。

### Android
- [x] “粘贴链接”页面与提交解析任务。
- [x] 解析状态轮询页面（加载中/失败/成功）。
- [x] 失败时展示明确文案（如无字幕、AI 解析失败）。

### 验收
- [x] AC1：重复 BVID 直接跳转已有菜谱。（后端返回 409 `BVID_ALREADY_PARSED` + `recipe_id` + `recipe_version_id`；App 展示去重提示卡并可一键跳转菜谱详情页——阶段 3 已落地）
- [x] AC2：无字幕的视频 15 秒内进入 FAILED 状态并提示，不调用 AI。

---

## 阶段 3：校对发布与菜谱阅读

**目标**：提交者可以校对并发布来源解析版，所有用户可以浏览公开菜谱。

### 后端
- [x] 实现 `POST /api/v1/recipe-versions/:id/publish`（仅 SOURCE 草稿作者，完整校对内容与公开状态原子写入）。
- [x] 实现 `GET /api/v1/recipe-versions/:id`（支持游客）。
- [x] 实现 `GET /api/v1/recipes?sort=latest&page=...` 公开列表。
- [x] 实现 `GET /api/v1/recipes/:id/version-tree` 版本派生树。
- [x] 视频失效时正确返回 `video_status = EXPIRED` 与警示信息。
- [x] 实现 `PUT /api/v1/recipe-versions/:id/draft` 校对保存（草稿作者全量改写食材与步骤；已公开版本拒绝静默覆盖）。

### Android
- [x] 提交者校对页面（编辑食材、用量、步骤）。
- [x] 菜谱详情页（食材、步骤、时间戳、版本来源）。
- [x] 社区发现列表页（最新/最热）。
- [x] 视频失效黄色警示条。

### 验收
- [x] AC4：视频失效后菜谱仍可浏览并显示警示。（后端返回 `video_status = EXPIRED` + `video_warning`，详情页顶部展示黄色警示条）
- [x] 游客可浏览公开菜谱。（发现列表与菜谱详情均无需登录；草稿仅作者可见）

---

## 阶段 4：纠错与派生改编

**目标**：社区用户可对来源解析版纠错，可在公开版本基础上派生改编。

### 后端
- [x] 实现 `POST /api/v1/recipe-versions/:id/corrections`。
- [x] 实现 `POST /api/v1/correction-proposals/:id/review`（仅来源发布者）。
- [x] 纠错通过后生成新的 SOURCE 版本，并更新 `correction_proposals.approved_version_id`。
- [x] 实现 `POST /api/v1/recipe-versions/:id/forks`（禁止对 KITCHEN_FIT 派生）。
- [x] 实现改编版发布，设置 `parent_id` 与 `version_type = 'ADAPTED'`。
- [x] 编写纠错、改编权限与版本树单元测试。
- [x] 实现 `GET /api/v1/me/correction-proposals`，仅返回当前来源发布者的待审核建议。

### Android
- [x] “来源纠错”页面（修改副本 + 填写理由）。
- [x] “派生改编”页面（基于当前版本自由编辑食材与步骤）。
- [x] 版本派生树可视化（或列表）。
- [x] 发布者待审核通知/列表。

### 验收
- [x] 非发布者无法直接修改 SOURCE 版本。
- [x] 个人适配版无法被改编。
- [x] 社区改编版可以被再次改编。

---

## 阶段 5：个人厨房智能适配

**目标**：用户可手动发起厨房适配，生成私有个人适配版，并可选择公开。

### 后端
- [x] 实现 `POST /api/v1/recipe-versions/:id/adaptations`。
- [x] 按 `ai-parsing.md` 调用 AI 适配并做安全校验。
- [x] 过敏/忌口无法安全替代时返回 `ALLERGY_BLOCKED`。
- [x] 实现 `PATCH /api/v1/recipe-versions/:id/visibility`（仅 KITCHEN_FIT 作者）。
- [x] 编写过敏拦截、适配结果保存的单元测试。

### Android
- [x] 菜谱详情页“适配我的厨房”按钮。
- [x] 适配结果页（替换说明、缺少项、口味变化）。
- [x] 适配失败提示（过敏拦截）。
- [x] 个人中心“我的适配版”列表与公开开关。

### 验收
- [x] AC3：过敏源无法替代时安全拦截；可替代时输出不含过敏食材。
- [x] 个人适配版默认私有，公开后不可被改编。

---

## 阶段 6：评论与“做过”成品图

**目标**：用户可以评论公开菜谱，并上传成品图。

### 后端
- [x] 实现 `POST /api/v1/recipe-versions/:id/comments` 与 `DELETE /api/v1/comments/:id`。
- [x] 实现 `POST /api/v1/recipe-versions/:id/cooking-records` 与 `DELETE /api/v1/cooking-records/:id`。
- [x] 实现 `POST /api/v1/upload/presigned-url` 预签名上传接口。
- [x] 编写评论、记录、软删除单元测试。

### Android
- [x] 菜谱详情页评论区。
- [x] “做过”弹窗（评分、反馈、成品图选择与上传）。
- [x] 成品图使用预签名 URL 直传对象存储。

### 验收
- [x] 游客不能评论/上传成品图。
- [x] 用户可删除自己的评论与“做过”记录。

---

## 阶段 7：治理、测试与发布准备

**目标**：补齐限流、日志、测试与构建发布。

### 后端
- [ ] 核心接口限流（Redis）。
- [ ] 日志与基础监控。
- [ ] 数据库迁移脚本 Up/Down 双向验证。
- [ ] 集成测试覆盖核心闭环。

### Android
- [ ] Release 构建与混淆配置。
- [ ] 关键 ViewModel 与 Repository 单元测试。
- [ ] Lint 零报错。

### 验收
- [ ] `go fmt ./... && go vet ./...` 通过。
- [ ] `go test ./...` 通过率 100%。
- [ ] `./gradlew assembleDebug` 通过，Lint 无报错。

---

## 当前阶段推荐

建议从 **阶段 0 → 阶段 1 → 阶段 2 → 阶段 3** 顺序推进，优先跑通“视频提交 → AI 解析 → 校对发布 → 浏览菜谱”的核心闭环，再进入社区共创与适配模块。
