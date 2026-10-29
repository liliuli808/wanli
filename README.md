# 碗里 (WanLi) - 项目说明与使用指南 (README.md)

欢迎来到 **“碗里 (WanLi)”** 移动端应用及后端系统的官方开源仓库！

“碗里” 是一款专门为烹饪爱好者打造的独立 Android 客户端与 Go 后端系统。它旨在打破“看视频学做菜”时频繁暂停、倒退、记录不便的痛点，通过 AI 提取 B 站等视频中的字幕，将其转化为结构化、可分步操作的精美菜谱。同时，系统支持根据用户的个人厨房现有调味料和忌口情况，一键手动适配专属于当前厨房的“完美菜谱”。

---

## 一、 项目背景与功能特色

1. **视频解析，一键装碗**：只需粘贴带有字幕的 B 站视频地址，AI 便能自动抽取出所需的食材、调味料、用量和分步烹饪逻辑，并自动与视频时间戳绑定。
2. **公开编辑，共创精品**：任何用户都可以对 AI 解析出来的菜谱进行“纠错建议”（需经发布者审核），或“派生改编”出自己的专属公开版本（如：低卡减脂版、空气炸锅版）。
3. **我的厨房，智能适配**：在 App 中勾选你厨房里已有的调料（只记录“有没有”，无需管理复杂的库存数量），设定全家人的避讳忌口与过敏源。在做任何菜前，点击“厨房适配”，AI 会立即计算替换方案，并重写一份适合你当下条件的菜谱步骤！
4. **大字烹饪模式**：进入烹饪模式，大字卡片显示，屏幕常亮，支持单步计时，提供下一步备料预告，让厨房手忙脚乱成为过去。
5. **美味生活，随手记录**：首版即支持在每个版本菜谱下进行评论互动，并在制作完成后通过“做过”记录，上传你的精美成品图。
6. **应用内自更新**：新版本发布后 App 启动即提示，点一下即可下载并安装，无需再跑去 GitHub 找安装包。

---

## 二、 目录结构规范

本项目采用典型的移动端客户端 + 服务端 Monorepos 或双目录结构进行组织：

```text
wanli-project/
├── AGENTS.md                  # AI 编程助手专用的“入职工作手册”
├── README.md                  # 本文件（项目主引导说明）
├── docs/                      # 产品及技术详细设计文档（AI 核心参考）
│   ├── product.md             # 产品定义与核心价值体系
│   ├── mvp.md                 # 最小可行性产品（MVP）功能边界与验收标准
│   ├── user-flows.md          # 核心用户使用旅程状态转换图
│   ├── data-model.md          # 数据库实体、关联关系与版本衍生模型 (Go & SQL)
│   ├── api-contract.md        # API 接口契约规范 (HTTP REST JSON)
│   ├── permissions.md         # 访问控制权限矩阵 (游客 vs 登录用户 vs 发布者)
│   ├── ai-parsing.md          # 视频解析与厨房适配的 AI Prompts & JSON Schema
│   └── development-tasks.md   # 分阶段垂直切片开发任务书
├── backend-go/                # Go 后端源码
│   ├── cmd/                   # 应用程序入口
│   ├── internal/              # 内部业务逻辑 (Router, Handler, Service, Repository, Model)
│   ├── pkg/                   # 公共可复用包 (database, logger, ai, parser)
│   └── migrations/            # SQL 数据库版本迁移脚本
└── frontend-android/          # Android 原生客户端源码
    ├── app/                   # 主程序 Module
    │   ├── src/main/kotlin/   # 源码目录 (采用 MVVM + Jetpack Compose 架构，Kotlin)
    │   └── src/main/res/      # 资源目录
    └── build.gradle.kts       # 客户端编译依赖
```

---

## 阶段 0：工程基线启动指南

阶段 0 已完成：项目包含可启动的 Go 后端骨架、Android 客户端骨架、PostgreSQL/Redis/MinIO 本地开发环境，以及完整的数据库迁移脚本。

### 1. 启动本地基础设施

确保已安装 Docker，然后在项目根目录执行：

```bash
docker compose up -d
```

这会启动：
- PostgreSQL 15：`127.0.0.1:5432`，数据库 `wanli_db`，用户名/密码 `postgres/postgres`
- Redis 7：`127.0.0.1:6380`（若本机 6379 已被占用，默认映射到 6380）
- MinIO：`127.0.0.1:9000`（API）与 `127.0.0.1:9001`（Console），账号 `minioadmin/minioadmin`

### 2. 配置并启动 Go 后端

```bash
cd backend-go
cp .env.example .env
# 按需编辑 .env，默认已匹配 docker compose 配置

# 执行数据库迁移
go run ./cmd/migrate up

# 启动 API 服务
go run ./cmd/api
```

如需解析仅登录可见的字幕，请在本地 `.env` 中配置 `BILIBILI_COOKIE` 后再启动服务。

服务默认监听 `http://127.0.0.1:8080`。可访问健康检查：

```bash
curl http://127.0.0.1:8080/api/v1/health
```

期望返回：

```json
{ "success": true, "data": { "status": "ok" } }
```

### 3. 编译运行 Android 客户端

1. 使用 **Android Studio Koala (或更高版本)** 打开 `frontend-android/` 文件夹。
2. 确保已安装 JDK 17 并将其配置为 Gradle JDK。
3. 修改 `app/src/main/assets/config.properties`，将 API 地址指向你的 Go 后端服务：
   ```properties
   BASE_URL=http://10.0.2.2:8080/api/v1/
   ```
4. 连接 Android 真机或启动模拟器，点击 **Run 'app'** 即可在设备上运行。

首次打开时，首页会自动请求后端 `/api/v1/health` 并显示连接状态。

**App 自更新**：CI 用 `GITHUB_RUN_NUMBER` 注入单调递增的 `versionCode`，并在 latest release 里发布
`version.json`（含 `version_code` / `sha256` / `size`）。App 启动时请求 `GET /api/v1/app/update`，
发现更高版本后经 `GET /api/v1/app/download` 下载（后端代理 GitHub，避免国内直连失败），
校验 SHA-256 后通过 FileProvider 拉起系统安装器。「我的 → 检查更新」可手动触发。

### 4. 常用检查命令

```bash
# 后端
(cd backend-go && go fmt ./... && go vet ./... && go test ./... && go build ./cmd/api && go build ./cmd/migrate)

# Android
(cd frontend-android && ./gradlew assembleDebug lintDebug testDebugUnitTest)
```

---

## 阶段 1：账号、个人厨房与忌口

阶段 1 已完成：支持邮箱验证码登录/注册、我的厨房配置、忌口设置。

### 1. 邮箱验证码登录流程

当前后端处于 mock 模式，发送验证码后会在服务日志中打印验证码，无需真实 SMTP。

```bash
# 1. 请求发送验证码
curl -X POST http://127.0.0.1:8080/api/v1/auth/email-code \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'

# 2. 后端日志会输出类似：
# [MOCK] verification code for test@example.com: 123456 (expires in 5m0s)

# 3. 使用验证码登录/注册
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}'
```

登录成功返回：

```json
{
  "success": true,
  "data": {
    "token": "eyJ...",
    "user": {
      "id": "...",
      "email": "test@example.com",
      "nickname": "碗友_xxxxxxxx",
      "avatar_url": null
    }
  }
}
```

### 2. 配置个人厨房

```bash
# 更新厨房配置（需携带 Authorization: Bearer <token>）
curl -X PUT http://127.0.0.1:8080/api/v1/me/kitchen \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "owned_ingredients": ["盐", "生抽", "糖", "大蒜", "姜"],
    "owned_tools": ["炒锅", "电饭煲"]
  }'
```

### 3. 配置忌口

```bash
# 更新忌口（全量替换）
curl -X PUT http://127.0.0.1:8080/api/v1/me/dietary-restrictions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "restrictions": [
      {"restriction_type": "ALLERGY", "ingredient_name": "花生"},
      {"restriction_type": "DISLIKE", "ingredient_name": "香菜"},
      {"restriction_type": "MEDICAL", "ingredient_name": "糖"}
    ]
  }'
```

忌口类型说明：
- `ALLERGY`：过敏源
- `DISLIKE`：不吃
- `MEDICAL`：医疗忌口

### 4. Android 端体验

1. 启动 Android 应用后，底部导航包含：首页、厨房、忌口、我的。
2. 未登录时点击“厨房”或“忌口”会提示登录。
3. 进入“我的”点击“登录”，输入邮箱并获取验证码（mock 模式任意 6 位数字均可登录）。
4. 登录后即可在“厨房”勾选调料和厨具，在“忌口”添加避讳食材。
5. Token、用户 ID、邮箱使用 DataStore 本地持久化。

### 5. 切换真实邮箱服务（可选）

编辑 `backend-go/.env`，将 `SMTP_MOCK=true` 改为 `SMTP_MOCK=false` 并配置 SMTP 参数：

```env
SMTP_MOCK=false
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=no-reply@wanli.com
SMTP_PASS=your_smtp_password
```

同时需要修改 `backend-go/internal/service/auth_service.go` 中的 `SendEmailCode`，将 `fmt.Printf` 替换为真实 SMTP 发送逻辑。

---

## 三、 本地快速启动指南（后续阶段通用）

以下指南适用于完整 MVP 开发阶段，包含所有服务配置。阶段 0 开发者可直接使用上方“阶段 0”指南。

### 3.1 运行 Go 后端服务
#### 1. 准备工作
- 安装 Go 1.22+ 运行环境
- 安装 PostgreSQL 15+ 并创建名为 `wanli_db` 的数据库
- 安装 Redis 7+

#### 2. 配置环境变量
在 `backend-go/` 根目录创建 `.env` 文件（生产环境请使用云端密钥注入，勿提交此文件）：
```env
PORT=8080
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=wanli_db
DB_SSLMODE=disable

REDIS_HOST=127.0.0.1
REDIS_PORT=6379

BILIBILI_COOKIE=
BILIBILI_CREDENTIAL_ENC_KEY=

AI_API_KEY=your_openai_or_deepseek_key
AI_BASE_URL=https://api.deepseek.com/v1
AI_MODEL=deepseek-chat

SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=no-reply@wanli.com
SMTP_PASS=smtp_auth_password

S3_ENDPOINT=your_s3_endpoint
S3_ACCESS_KEY=your_access_key
S3_SECRET_KEY=your_secret_key
S3_BUCKET=wanli-assets

# App 自更新：读取 latest release 的 version.json 并代理下载 APK
APP_UPDATE_BASE_URL=https://github.com/liliuli808/wanli/releases/download/latest
APP_UPDATE_CACHE_SECONDS=600
```

`BILIBILI_COOKIE` 可选，是服务端兜底的 B 站登录态，用于读取仅登录用户可见的字幕轨。
填写浏览器请求中的完整 `Cookie` 请求头，只能保存在后端本地 `.env` 或部署平台密钥中；
禁止提交、写入日志或放入 Android。Cookie 更新后需要重启后端，过期后需重新获取。

更推荐的做法是**由用户在自己的账号里授权**：App 内「我的 → B 站账号」扫码登录或手动粘贴
`SESSDATA`，后端先用 `nav` 接口校验，再用 AES-256-GCM 加密存进 `users` 表，解析时优先使用
该用户的登录态。加密密钥来自 `BILIBILI_CREDENTIAL_ENC_KEY`，留空时由 `JWT_SECRET` 派生；
轮换密钥会让已授权凭证失效，需要用户重新授权。凭证永远不会通过 API 回传，用户可随时解除授权。

`AI_MODEL` 是发送给 `AI_BASE_URL` 所配置服务商的模型标识，默认值为
`deepseek-chat`。修改 AI 环境变量后需要重启后端服务才能生效；模型标识必须是
对应服务商实际支持的值。

#### 3. 运行迁移与启动
```bash
cd backend-go
# 执行迁移
go run cmd/migrate/main.go up
# 启动 API 服务
go run cmd/api/main.go
```

---

### 3.2 编译运行 Android 客户端
1. 使用 **Android Studio Koala (或更高版本)** 打开 `frontend-android/` 文件夹。
2. 确保已安装 JDK 17 并将其配置为 Gradle JDK。
3. 修改 `app/src/main/assets/config.properties`，将 API 地址指向你的 Go 后端服务：
   ```properties
   BASE_URL=http://10.0.2.2:8080/api/v1/
   ```
4. 连接 Android 真机或启动模拟器，点击 **Run 'app'** 即可在设备上运行。

---

## 四、 开发者参与须知

请在开始编写任何业务代码前，在本地命令行运行一次：
- **后端静态检查**：`go fmt ./... && go vet ./...`
- **后端测试**：`go test ./...`
- **Android 构建**：`./gradlew assembleDebug`

所有的需求细化、接口变更均应优先反映在 `docs/` 目录中。如果您正在使用 **Cursor / Claude / Copilot** 等 AI 编程助手，请直接在 prompt 中加上：
> *“请在工作前完整阅读根目录的 `AGENTS.md` 和 `docs/`，并遵循其中的设计、规范与工作流约束。”*
