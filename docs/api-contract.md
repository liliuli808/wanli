# “碗里” (WanLi) - API 接口契约规范 (api-contract.md)

为了保证 Android 原生端与 Go 后端解耦并行开发，两端必须严格遵守以下 RESTful API 契约规范。所有请求与返回数据均采用 JSON 格式，时间字段采用 ISO-8601 UTC 格式。

---

## 一、 统一返回格式与错误处理

### 1.1 成功返回模板 (HTTP 200/201)
```json
{
  "success": true,
  "data": {} // 或 []
}
```

### 1.2 异常返回模板 (HTTP 4xx/5xx)
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "用户可读的友好提示信息",
    "details": null // 调试详情，非生产环境返回
  }
}
```

### 1.3 核心错误码
| 错误码 | 含义 | 典型场景 |
| :--- | :--- | :--- |
| `UNAUTHORIZED` | 未登录或 Token 无效 | 游客访问需登录接口 |
| `FORBIDDEN` | 无权操作 | 非所有者修改/审核资源 |
| `RESOURCE_NOT_FOUND` | 资源不存在 | 版本、任务、用户不存在 |
| `BVID_ALREADY_PARSED` | 该 BVID 已有公开来源解析版 | 提交重复视频 |
| `VIDEO_FETCH_FAILED` | 获取视频信息失败 | B 站接口异常或视频不存在 |
| `SUBTITLE_NOT_AVAILABLE` | 无可用字幕 | 未找到 CC/AI 字幕或字幕拉取失败 |
| `AI_PARSE_FAILED` | AI 解析失败 | 字幕无法结构化 |
| `KITCHEN_FIT_CANNOT_FORK` | 个人适配版不可被改编 | 对个人适配版点击改编 |
| `ALLERGY_BLOCKED` | 过敏源无法安全替代 | 厨房适配被拦截 |
| `DIETARY_BLOCKED` | 忌口/医疗限制无法安全满足 | 厨房适配被拦截 |
| `AI_ADAPTATION_FAILED` | AI 厨房适配调用失败 | 模型超时、异常或返回无效结构 |
| `VALIDATION_ERROR` | 请求参数校验失败 | 字段缺失/格式错误 |
| `AUTH_EMAIL_TAKEN` | 邮箱已被注册 | 注册时邮箱已存在 |
| `AUTH_INVALID_CREDENTIALS` | 邮箱或密码错误 | 登录校验失败（不区分账号是否存在） |
| `CAPTCHA_FAILED` | 人机验证未通过 | 注册/登录时 Turnstile 校验失败 |
| `AUTH_TOO_MANY_ATTEMPTS` | 登录失败次数超限，账号临时锁定 | 连续失败达限流阈值 |

---

## 二、 核心 API 接口列表

### 2.1 用户认证 (Auth)

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

> **人机验证（默认关闭）**：`turnstile_token` 为可选字段。`TURNSTILE_ENABLED=false`（默认）时服务端跳过校验；设为 `true` 后，Android 端需在 `GET /captcha` 加载内嵌 Turnstile 组件的页面，通过 WebView 的 `AndroidBridge` 拿到 token 提交。token 为一次性，校验失败后需重新加载页面获取新 token。

---

### 2.2 视频导入与解析 (Video Imports)

#### 1. 提交视频解析任务
- **请求**：`POST /api/v1/video-imports`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  { "url": "【标题-哔哩哔哩】 https://b23.tv/osM7T47" }
  ```
- **说明**：`url` 支持纯链接、短链（`b23.tv` / `bili.im`）、纯 BV 号，以及 B 站 App 的整段分享文案；服务端会自动提取其中的链接。
- **返回**：`201 Created`
  ```json
  {
    "success": true,
    "data": {
      "task_id": "f51a8d11-b011-4a5c-9c76-58c1dfcbb982",
      "status": "PENDING", // PENDING, FETCHING, PARSING, COMPLETED, FAILED
      "bvid": "BV1xx411c7xx"
    }
  }
  ```
- **去重冲突**：`409 BVID_ALREADY_PARSED` 时 `error.details` 携带 `recipe_id` 与 `recipe_version_id`（已公开的 SOURCE 版本；若来源版尚未发布则仅返回 `recipe_id`），客户端可直接跳转已有菜谱详情页。
- **需要登录 B 站**：任务失败时 `error_code = BILIBILI_LOGIN_REQUIRED`，表示该视频字幕仅登录可见，提示用户到「我的 - B 站账号」授权后重试。

#### 2. 查询解析任务状态
- **请求**：`GET /api/v1/video-imports/:task_id`
- **返回**：`200 OK` (若状态为 `COMPLETED`，返回预览版 `recipe_version_id`)

---

### 2.3 菜谱阅读与编辑 (Recipes & Versions)

#### 1. 获取公开菜谱详情 (含步骤、食材)
- **请求**：`GET /api/v1/recipe-versions/:id`
- **Headers**: (可选 `Authorization: Bearer <token>`，允许游客浏览)
- **说明**：公开版本所有人可浏览；未公开的 SOURCE 草稿仅作者可见，其他人访问返回 `RESOURCE_NOT_FOUND`（不泄露草稿存在性）。视频失效时 `video_status = EXPIRED` 且 `video_warning` 返回警示文案（菜谱仍可正常浏览）。
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "id": "7831f28b-b8c2-48df-b59a-09dcfcb82d1c",
      "recipe_id": "e0a29b31-48cf-4cc1-9cb8-bdfcfcb9928c",
      "parent_id": null,
      "title": "家常红烧肉",
      "up_nickname": "美食作家王刚",
      "video_url": "https://www.bilibili.com/video/BV1xx411c7xx/",
      "video_status": "ACTIVE",
      "video_warning": null,
      "cover_url": "https://example.com/cover.jpg",
      "version_type": "SOURCE",
      "is_public": true,
      "author_nickname": "王刚粉丝甲",
      "change_log": null,
      "servings": "2-3人份",
      "estimated_minutes": 45,
      "instructions": "注意火候",
      "ingredients": [
        { "name": "五花肉", "amount": "500克", "category": "MAIN", "is_explicit": true, "substitute_for": null }
      ],
      "steps": [
        { "step_order": 1, "description": "将五花肉切成2厘米见方的块", "video_timestamp": 45, "duration_seconds": 60, "heat": "无" }
      ]
    }
  }
  ```

#### 1.1 校对保存来源解析草稿
- **请求**：`PUT /api/v1/recipe-versions/:id/draft`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：仅 `version_type = 'SOURCE'` 且 `is_public = FALSE` 的草稿版本作者调用；全量替换标题、食材与步骤（草稿允许改写；已公开版本禁止静默覆盖，返回 `VALIDATION_ERROR`）。
- **Body**:
  ```json
  {
    "title": "秘制红烧肉",
    "servings": "3人份",
    "estimated_minutes": 50,
    "instructions": "焯水不可省略",
    "ingredients": [
      { "name": "五花肉", "amount": "600克", "category": "MAIN", "is_explicit": true },
      { "name": "冰糖", "amount": null, "category": "SEASONING", "is_explicit": false }
    ],
    "steps": [
      { "step_order": 1, "description": "焯水去腥", "video_timestamp": 30, "duration_seconds": null, "heat": null }
    ]
  }
  ```
- **返回**：`200 OK`（返回更新后的完整版本详情，结构同上）

#### 2. 创建改编版 (派生)
- **请求**：`POST /api/v1/recipe-versions/:id/forks`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：父版本必须公开且类型为 `SOURCE` 或 `ADAPTED`；`KITCHEN_FIT` 一律返回 `KITCHEN_FIT_CANNOT_FORK`。请求成功即创建公开的 `ADAPTED` 版本。
- **Body**:
  ```json
  {
    "title": "空气炸锅少油红烧肉",
    "change_log": "减少用油并改用空气炸锅",
    "servings": "2人份",
    "estimated_minutes": 35,
    "instructions": "中途翻面一次",
    "ingredients": [
      { "name": "五花肉", "amount": "500克", "category": "MAIN", "is_explicit": true }
    ],
    "steps": [
      { "step_order": 1, "description": "空气炸锅 180℃ 烤制", "video_timestamp": null, "duration_seconds": 1200, "heat": null }
    ]
  }
  ```
- **返回**：`201 Created`（返回新生成改编版的完整详情）

#### 3. 发布来源解析版（校对完成）
- **请求**：`POST /api/v1/recipe-versions/:id/publish`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：仅允许 `version_type = 'SOURCE'` 且当前 `is_public = FALSE` 的草稿版本作者调用。Body 与“校对保存来源解析草稿”一致；后端在同一事务中写入最终校对内容并设置 `is_public = TRUE`，不存在“保存成功但发布失败”的中间状态。
- **Body**：完整 `title`、`servings`、`estimated_minutes`、`instructions`、`ingredients` 与 `steps`，结构同 `PUT /draft`。
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "recipe_version_id": "7831f28b-b8c2-48df-b59a-09dcfcb82d1c",
      "is_public": true
    }
  }
  ```

#### 4. 提交纠错建议
- **请求**：`POST /api/v1/recipe-versions/:id/corrections`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  {
    "suggested_changes": {
      "title": "家常红烧肉",
      "servings": "2-3人份",
      "estimated_minutes": 45,
      "instructions": "焯水不可省略",
      "ingredients": [
        { "name": "五花肉", "amount": "600克", "category": "MAIN", "is_explicit": true }
      ],
      "steps": [
        { "step_order": 1, "description": "焯水去腥", "video_timestamp": 30, "duration_seconds": null, "heat": null }
      ]
    },
    "reason": "视频第 2 步明确说了要焯水，AI 漏掉了"
  }
  ```
- **返回**：`201 Created`

#### 5. 审核纠错建议
- **请求**：`POST /api/v1/correction-proposals/:id/review`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  {
    "action": "approve", // approve | reject
    "rejection_reason": "视频中没有该步骤"
  }
  ```
- **说明**：`reject` 时 `rejection_reason` 必填。并发审核只允许第一次成功；`approve` 会生成新的公开 `SOURCE` 版本，原版本保持不变。
- **返回**：`200 OK` (若 `approve`，返回生成的新版本 `recipe_version_id`)

#### 5.1 获取当前发布者待审核纠错
- **请求**：`GET /api/v1/me/correction-proposals`
- **Headers**: `Authorization: Bearer <token>`
- **返回**：`200 OK`，`data.items` 仅包含 `recipes.creator_id` 为当前用户且状态为 `PENDING` 的建议；每项包含 `id`、`recipe_version_id`、`recipe_title`、`proposer_nickname`、`reason`、完整 `suggested_changes` 与 `created_at`。

#### 6. 获取菜谱版本派生树
- **请求**：`GET /api/v1/recipes/:id/version-tree`
- **Headers**: (可选 `Authorization: Bearer <token>`，允许游客浏览)
- **返回**：`200 OK` (返回该菜谱下所有公开版本的树状结构，包含 `id`, `version_type`, `parent_id`, `author_nickname`, `created_at`)
  ```json
  {
    "success": true,
    "data": {
      "recipe_id": "e0a29b31-48cf-4cc1-9cb8-bdfcfcb9928c",
      "nodes": [
        { "id": "7831f28b-...", "title": "家常红烧肉", "version_type": "SOURCE", "parent_id": null, "author_nickname": "王刚粉丝甲", "change_log": null, "created_at": "2026-07-24T10:00:00Z" },
        { "id": "9a42bb70-...", "title": "空气炸锅红烧肉", "version_type": "ADAPTED", "parent_id": "7831f28b-...", "author_nickname": "减脂达人乙", "change_log": "减少用油", "created_at": "2026-07-24T11:00:00Z" }
      ]
    }
  }
  ```

#### 7. 获取公开菜谱列表
- **请求**：`GET /api/v1/recipes?sort=latest&page=1&page_size=20`
- **Headers**: (可选 `Authorization: Bearer <token>`，允许游客浏览)
- **Query 参数**：
  - `sort`: `latest`（按最新公开来源版本时间）, `hottest`（按公开版本总数降序，同分按最新来源版本时间降序）
  - `category`: 可选，如 `家常菜`、`减脂`（MVP 暂未实现分类筛选）
  - `page`, `page_size`: 分页（`page_size` 默认 20，最大 50）
- **返回**：`200 OK` (返回公开菜谱列表及首个公开版本摘要)
  ```json
  {
    "success": true,
    "data": {
      "items": [
        {
          "recipe_id": "e0a29b31-...",
          "title": "家常红烧肉",
          "cover_url": "https://example.com/cover.jpg",
          "up_nickname": "美食作家王刚",
          "video_status": "ACTIVE",
          "recipe_version_id": "7831f28b-...",
          "servings": "2-3人份",
          "estimated_minutes": 45,
          "author_nickname": "王刚粉丝甲",
          "public_version_count": 3,
          "cooking_record_count": 12,
          "created_at": "2026-07-24T10:00:00Z"
        }
      ],
      "total": 1,
      "page": 1,
      "page_size": 20
    }
  }
  ```

---

### 2.4 个人厨房与忌口 (Kitchen & Dietary)

#### 1. 获取当前登录用户信息
- **请求**：`GET /api/v1/me`
- **Headers**: `Authorization: Bearer <token>`
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "id": "e93817f5-2f84-482f-b05b-86d1dfcbf81c",
      "email": "user@example.com",
      "nickname": "小碗",
      "avatar_url": null
    }
  }
  ```

#### 2. 更新当前用户资料（昵称/头像）
- **请求**：`PUT /api/v1/me/profile`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  { "nickname": "小碗", "avatar_url": "https://wanli-assets.s3.amazonaws.com/uploads/avatars/e93b-xxxx.jpg" }
  ```
- **说明**：`nickname` 必填（1-50 个字符）；`avatar_url` 可选，传 `null` 清除头像。
- **返回**：`200 OK`，`data` 为更新后的用户信息（`id`, `email`, `nickname`, `avatar_url`）。

#### 3. 获取/更新个人厨房配置
- **请求**：`GET /api/v1/me/kitchen` | `PUT /api/v1/me/kitchen`
- **Headers**: `Authorization: Bearer <token>`
- **Body (PUT)**:
  ```json
  {
    "owned_ingredients": ["盐", "生抽", "老抽", "糖", "大蒜"],
    "owned_tools": ["电饭煲", "炒锅", "空气炸锅"]
  }
  ```

#### 4. 获取/更新个人忌口
- **请求**：`GET /api/v1/me/dietary-restrictions` | `PUT /api/v1/me/dietary-restrictions`
- **Headers**: `Authorization: Bearer <token>`
- **Body (PUT)**:
  ```json
  {
    "restrictions": [
      { "restriction_type": "ALLERGY", "ingredient_name": "花生" },
      { "restriction_type": "DISLIKE", "ingredient_name": "香菜" },
      { "restriction_type": "MEDICAL", "ingredient_name": "糖" }
    ]
  }
  ```
- **说明**：`restriction_type` 限定为 `ALLERGY`（过敏）、`DISLIKE`（不吃）、`MEDICAL`（医疗控糖/控盐）；更新为全量替换。

#### 5. B 站账号授权（读取仅登录可见的字幕）
- **请求**：`GET /api/v1/me/bilibili-credential` | `PUT /api/v1/me/bilibili-credential` | `DELETE /api/v1/me/bilibili-credential`
- **Headers**: `Authorization: Bearer <token>`
- **Body (PUT)**:
  ```json
  { "sessdata": "abc123%2C1793262567%2Cxyz*11" }
  ```
- **返回**：`200 OK`
  ```json
  { "success": true, "data": { "bound": true, "mid": "123456", "uname": "小碗" } }
  ```
- **说明**：
  - 只接受 `SESSDATA` 值（可从 `SESSDATA=` 前缀或引号中自动清洗），不接受完整 Cookie 串。
  - 服务端先用 B 站 `nav` 接口校验，再用 AES-256-GCM 加密落库；**任何接口都不会回传凭证内容**。
  - `DELETE` 清除授权，返回 `{ "bound": false }`。
  - 错误码：`BILIBILI_CREDENTIAL_INVALID`（400，无效或已过期）、`BILIBILI_CREDENTIAL_DISABLED`（503，服务端未配置加密密钥）。

---

### 2.5 个人厨房手动适配 (Adaptation)

#### 1. 请求 AI 手动厨房适配
- **请求**：`POST /api/v1/recipe-versions/:id/adaptations`
- **Headers**: `Authorization: Bearer <token>`
- **约束**：仅可适配公开且未下架的 `SOURCE` / `ADAPTED` 版本。公开的 `KITCHEN_FIT` 是终端节点，不允许再次派生。
- **返回**：`200 OK`，`data` 为 2.3.1 的完整版本详情；新版本固定为 `version_type = "KITCHEN_FIT"`、`is_public = false`，并额外返回：
  ```json
  {
    "adaptation_notes": {
      "missing_ingredients": [],
      "missing_tools": [],
      "substitutions": [
        {
          "original": "花生酱",
          "replacement": "芝麻酱",
          "reason": "规避花生过敏源"
        }
      ],
      "flavor_changes": ["坚果香气会有变化"]
    }
  }
  ```
- **安全失败**：过敏源无法安全替代返回 `ALLERGY_BLOCKED`；其他忌口/医疗限制无法满足返回 `DIETARY_BLOCKED`；失败时不创建版本。

#### 2. 将个人适配版设为公开/取消公开
- **请求**：`PATCH /api/v1/recipe-versions/:id/visibility`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  { "is_public": true }
  ```
- **说明**：仅 `version_type = 'KITCHEN_FIT'` 且 `author_id` 为当前用户时允许。一旦 `is_public = TRUE`，该版本即成为终端节点，其他用户无法对其调用 `/forks`。
- **返回**：`200 OK`，`data` 为更新后的完整版本详情。

#### 3. 获取我的个人适配版
- **请求**：`GET /api/v1/me/adaptations`
- **Headers**: `Authorization: Bearer <token>`
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "a60f...",
          "recipe_id": "63ac...",
          "parent_id": "7831...",
          "title": "芝麻酱拌面",
          "is_public": false,
          "created_at": "2026-07-24T06:30:00Z"
        }
      ]
    }
  }
  ```
- **说明**：该接口按创建时间倒序返回当前用户的全部 `KITCHEN_FIT`，包括私有版本。

---

### 2.6 图片上传 (Uploads)

#### 1. 获取成品图预签名上传 URL
- **请求**：`POST /api/v1/upload/presigned-url`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  {
    "filename": "finished-dish.jpg",
    "content_type": "image/jpeg",
    "size_bytes": 2048000
  }
  ```
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "upload_url": "https://wanli-assets.s3.amazonaws.com/...",
      "public_url": "https://wanli-assets.s3.amazonaws.com/uploads/records/e93b-xxxx.jpg",
      "object_name": "uploads/records/e93b-xxxx.jpg",
      "expires_in": 900
    }
  }
  ```
- **说明**：仅接受 JPEG、PNG、WebP，大小为 1 字节至 10MB；预签名 URL 有效期 15 分钟。客户端使用无 API 鉴权头的 `PUT` 直传对象存储，再将 `public_url` 填入“做过”记录。

---

### 2.7 评论与成品展示 (Comments & Records) (MVP新增)

#### 1. 获取评论
- **请求**：`GET /api/v1/recipe-versions/:id/comments`
- **说明**：游客可读取公开且未下架版本的有效评论，按时间倒序返回；已软删除评论不返回。
- **返回**：
  ```json
  {
    "success": true,
    "data": {
      "items": [{
        "id": "comment-id",
        "content": "很好吃",
        "author_nickname": "小碗",
        "is_mine": false,
        "created_at": "2026-07-24T08:00:00Z"
      }]
    }
  }
  ```

#### 2. 发表评论
- **请求**：`POST /api/v1/recipe-versions/:id/comments`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  { "content": "这道菜用空气炸锅适配后真的太省油了！" }
  ```
- **说明**：仅登录用户可对公开且未下架版本评论；正文长度 1-1000 个字符。
- **返回**：`201 Created`

#### 3. 删除评论
- **请求**：`DELETE /api/v1/comments/:id`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：仅评论作者可删除；删除使用 `status = 'DELETED'` 软删除。
- **返回**：`204 No Content`

#### 4. 获取“做过”记录
- **请求**：`GET /api/v1/recipe-versions/:id/cooking-records`
- **说明**：游客可读取公开且未下架版本的有效记录，按时间倒序返回；已软删除记录不返回。

#### 5. 提交“做过”记录并上传成品图
- **请求**：`POST /api/v1/recipe-versions/:id/cooking-records`
- **Headers**: `Authorization: Bearer <token>`
- **Body**:
  ```json
  {
    "rating": 5,
    "photo_url": "https://wanli-assets.s3.amazonaws.com/uploads/records/e93b-xxxx.jpg",
    "feedback": "外酥里嫩，家人很喜欢！"
  }
  ```
- **说明**：仅登录用户可提交，`rating` 限 1-5；评分、反馈、成品图至少填写一项。
- **返回**：`201 Created`

#### 6. 删除“做过”记录
- **请求**：`DELETE /api/v1/cooking-records/:id`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：仅记录作者可删除；删除使用 `status = 'DELETED'` 软删除。
- **返回**：`204 No Content`

#### 7. 全站“做过”动态流（首页 Feed）
- **请求**：`GET /api/v1/feed/cooking-records?limit=20`
- **Headers**: (可选 `Authorization: Bearer <token>`，允许游客浏览)
- **说明**：按创建时间倒序返回全站公开且未下架版本的“做过”记录；`limit` 默认 20，超出 1~50 回退为 20。
- **返回**：`200 OK`，`items` 每项含 `id`, `rating`, `photo_url`, `feedback`, `author_nickname`, `created_at`, `recipe_version_id`, `recipe_id`, `recipe_title`, `cover_url`。

#### 8. 我的“做过”列表（照片墙）
- **请求**：`GET /api/v1/me/cooking-records?limit=60`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：按创建时间倒序返回当前用户的“做过”记录（仅关联未下架版本与菜谱）；`limit` 默认 60，超出 1~100 回退为 60。
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "a60f...",
          "rating": 5,
          "photo_url": "https://wanli-assets.s3.amazonaws.com/uploads/records/e93b-xxxx.jpg",
          "feedback": "外酥里嫩，家人很喜欢！",
          "created_at": "2026-07-24T08:00:00Z",
          "recipe_version_id": "7831f28b-...",
          "recipe_title": "家常红烧肉",
          "cover_url": "https://example.com/cover.jpg"
        }
      ]
    }
  }
  ```

---

### 2.8 消息中心 (Messages)

#### 1. 获取我的消息
- **请求**：`GET /api/v1/me/messages`
- **Headers**: `Authorization: Bearer <token>`
- **说明**：聚合四类事件——他人对我版本的评论（`comment`）、他人对我版本的“做过”（`cooking_record`）、我的纠错审核结果（`correction_reviewed`）、他人提交给我的待审核纠错（`correction_pending`），按时间倒序返回，每类取最近 50 条，合并后截断为 50 条。
- **返回**：`200 OK`，`items` 每项含 `type`, `id`, `actor_nickname`, `text`, `rating`, `recipe_version_id`, `recipe_title`, `created_at`。

---

### 2.9 App 自更新 (App Update)

#### 1. 查询最新版本
- **请求**：`GET /api/v1/app/update`
- **Headers**: 无需登录
- **说明**：服务端读取 CI 发布到 GitHub Releases 的 `version.json`（缓存 10 分钟，可在 `APP_UPDATE_CACHE_SECONDS` 调整）。
  客户端用 `version_code` 与本地 `BuildConfig.VERSION_CODE` 比较，大于本地版本才提示更新。
  上游不可用时返回 503 `APP_UPDATE_UNAVAILABLE`，客户端应静默忽略（自动检查）或提示稍后再试（手动检查）。
- **返回**：`200 OK`
  ```json
  {
    "success": true,
    "data": {
      "version_code": 42,
      "version_name": "0.0.42",
      "size": 12345678,
      "sha256": "…",
      "commit": "d830e28",
      "published_at": "2026-10-29T00:00:00Z",
      "download_path": "/api/v1/app/download"
    }
  }
  ```

#### 2. 下载最新安装包
- **请求**：`GET /api/v1/app/download`
- **Headers**: 无需登录
- **说明**：后端代理 GitHub Releases 的 APK，`Content-Type: application/vnd.android.package-archive`。
  走代理而不是让客户端直连 GitHub，是为了规避国内网络访问 Releases 不稳定。
  客户端下载完成后应按 `sha256` 校验，再通过 FileProvider 拉起系统安装器。
- **返回**：`200 OK`（二进制流）

**相关配置**

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `APP_UPDATE_BASE_URL` | `https://github.com/liliuli808/wanli/releases/download/latest` | latest release 资产目录 |
| `APP_UPDATE_CACHE_SECONDS` | `600` | `version.json` 缓存秒数 |
