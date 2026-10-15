# “碗里” (WanLi) - 数据库模型与版本衍生设计 (data-model.md)

为了支持“来源解析版”、“社区改编版”、“个人适配版”三层树状派生结构，以及首版新增的评论和“做过”上传图片功能，“碗里”后端采用 **PostgreSQL** 作为主数据库。以下是核心表结构与关联设计。

---

## 一、 关系型数据库模型 (SQL DDL/Go Struct 规范)

### 1. 用户表 (users)
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    hashed_password VARCHAR(255) NOT NULL,
    nickname VARCHAR(50) NOT NULL,
    avatar_url VARCHAR(512),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 2. 视频来源表 (video_sources)
```sql
CREATE TABLE video_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bvid VARCHAR(50) UNIQUE NOT NULL, -- B站唯一视频标识
    title VARCHAR(255) NOT NULL,
    up_nickname VARCHAR(100) NOT NULL,
    video_url VARCHAR(512) NOT NULL,
    cover_url VARCHAR(512),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE (正常), EXPIRED (来源失效/视频下架)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3. 菜谱根实体表 (recipes)
每一道被解析的视频菜谱即为一个根节点。后续的社区改编版、个人适配版均挂在同一 `recipe_id` 下，通过 `recipe_versions.parent_id` 构建版本衍生树；`recipes.creator_id` 始终为最初提交并发布该视频来源解析版的用户（即来源发布者）。
```sql
CREATE TABLE recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(100) NOT NULL, -- 菜名
    video_source_id UUID REFERENCES video_sources(id) ON DELETE SET NULL, -- 视频来源
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL, -- 最初提交并发布来源解析版的用户
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE (正常), BLOCKED (被管理员下架)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 4. 菜谱正文版本表 (recipe_versions)
菜谱的实质内容，每次编辑、改编或适配都会在此表产生一个唯一的版本记录。已公开的版本不允许静默覆盖，任何内容变更都必须生成新版本。
```sql
CREATE TABLE recipe_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES recipe_versions(id) ON DELETE SET NULL, -- 父版本ID，建立衍生树
    author_id UUID REFERENCES users(id) ON DELETE SET NULL, -- 当前版本的作者/改编者
    version_type VARCHAR(20) NOT NULL, -- SOURCE (来源解析版), ADAPTED (社区改编版), KITCHEN_FIT (个人适配版)
    title VARCHAR(100) NOT NULL, -- 当前版本菜名；改编版可以拥有独立标题
    is_public BOOLEAN NOT NULL DEFAULT FALSE, -- 是否公开；SOURCE/ADAPTED 创建时通常为 TRUE，KITCHEN_FIT 创建时为 FALSE
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE (正常), BLOCKED (被管理员下架)
    servings VARCHAR(50), -- 适合分量（如“2人份”）
    estimated_minutes INT, -- 预计耗时
    instructions TEXT, -- 提示与注意事项
    change_log VARCHAR(255), -- 改动日志说明
    adaptation_notes JSONB, -- 仅 KITCHEN_FIT：缺少项、替换原因与口味变化
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

**各类型创建时的公开约定**
- `SOURCE`：提交者校对发布时 `is_public = TRUE`；草稿/待校对状态为 `FALSE`。
- `ADAPTED`：发布时 `is_public = TRUE`。
- `KITCHEN_FIT`：生成时 `is_public = FALSE`；用户手动选择“公开分享”后可设为 `TRUE`。
- `adaptation_notes` 由结构化 AI 适配输出写入，正文仍保存在食材/步骤表中；非 `KITCHEN_FIT` 版本为 `NULL`。

### 5. 食材与配料清单表 (recipe_ingredients)
```sql
CREATE TABLE recipe_ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL, -- 配料/食材名称
    amount VARCHAR(50), -- 数量，如“200g”、“少许”；AI 不确定时允许为 NULL
    category VARCHAR(20) NOT NULL, -- MAIN (主料), AUXILIARY (辅料), SEASONING (调味品)
    is_explicit BOOLEAN NOT NULL DEFAULT TRUE, -- TRUE: 字幕明确提到; FALSE: AI推测/补充
    substitute_for VARCHAR(100) -- 如果是厨房适配产生的替换料，记录原食材名称
);
```

### 6. 烹饪步骤表 (recipe_steps)
```sql
CREATE TABLE recipe_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id) ON DELETE CASCADE,
    step_order INT NOT NULL, -- 步骤序号
    description TEXT NOT NULL, -- 步骤描述
    video_timestamp INT, -- 对应原视频的时间戳（秒数）
    duration_seconds INT, -- 步骤预计用时（秒）
    heat VARCHAR(50) -- 火候控制，如“大火”、“中小火”
);
```

### 7. 个人厨房配置表 (kitchen_profiles)
只记录“有没有”调料与厨具，Boolean 标志。
```sql
CREATE TABLE kitchen_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    owned_ingredients JSONB NOT NULL DEFAULT '[]'::jsonb, -- 拥有的调味品/常备食材名数组，如 ["盐", "生抽", "老抽"]
    owned_tools JSONB NOT NULL DEFAULT '[]'::jsonb, -- 拥有的厨具，如 ["空气炸锅", "电压力锅"]
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 8. 饮食限制表 (dietary_restrictions)
```sql
CREATE TABLE dietary_restrictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    restriction_type VARCHAR(20) NOT NULL, -- ALLERGY (过敏), DISLIKE (偏好不吃), MEDICAL (医疗控糖/控盐)
    ingredient_name VARCHAR(100) NOT NULL, -- 限制的食材/品类名
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 9. 评论表 (comments) (MVP新增)
```sql
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE (正常), DELETED (用户或管理员删除)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 10. 做过记录表 (cooking_records) (MVP新增)
```sql
CREATE TABLE cooking_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INT CHECK (rating BETWEEN 1 AND 5), -- 评分
    photo_url VARCHAR(512), -- 上传的成品图 URL
    feedback TEXT, -- 制作体会
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE (正常), DELETED (作者软删除)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 11. 纠错建议表 (correction_proposals)
```sql
CREATE TABLE correction_proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_version_id UUID NOT NULL REFERENCES recipe_versions(id) ON DELETE CASCADE, -- 被纠错的版本
    proposer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    suggested_changes JSONB NOT NULL, -- 修改后的完整菜谱副本；后端据此确定性生成新版本
    reason VARCHAR(255) NOT NULL, -- 纠错理由
    rejection_reason VARCHAR(255), -- 拒绝时由来源发布者填写
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING (待处理), APPROVED (已通过), REJECTED (拒绝)
    approved_version_id UUID REFERENCES recipe_versions(id) ON DELETE SET NULL, -- 审核通过后生成的新版本ID
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

---

## 二、 三层版本在 SQL 中的级联与衍生设计约束

### 1. 修改保护约束
- `version_type = 'SOURCE'`（来源解析版）的版本，其 `author_id` 始终是提交解析的人。除该作者通过“纠错建议”审批直接生成新版本外，其他用户调用任何更新接口一律报错 `403 Forbidden`。
- 未公开的 `SOURCE` 草稿允许作者保存校对内容；最终发布必须在一个事务中同时写入完整校对内容并设置 `is_public = TRUE`。
- **禁止静默覆盖**：任何已公开的版本（`is_public = TRUE`）的内容变更，都必须生成新的 `recipe_versions` 记录，并填写 `change_log`，不允许直接 UPDATE 原记录。

### 2. 独占改编与叶子节点约束
- 当用户想要“改编”某个版本时：
  - 后端检查父版本的 `version_type`。如果是 `KITCHEN_FIT`（个人适配版），**一律拒绝并报错“个人适配版属于定制节点，不支持派生改编”**。
  - 对于 `SOURCE` 和 `ADAPTED`，均允许继续派生。
  - 创建新的 `recipe_versions`，设置 `parent_id = 原版本ID`，`recipe_id = 原版本所属 recipe_id`，`version_type = 'ADAPTED'`，`is_public = TRUE`。

### 3. 来源失效级联删除约束
- 原视频 `video_sources` 被物理删除或标记下架（`status = 'EXPIRED'`）时：
  - `recipes.video_source_id` 设为 `NULL` 或保持原引用，但 **绝不级联删除 `recipes` 和 `recipe_versions`**。
  - 这确保了用户的改编历史和适配历史依然可用，仅在界面显示“原视频下架”警示。

### 4. 内容治理字段约定
- `recipes.status` 与 `recipe_versions.status` 默认均为 `ACTIVE`。管理员下架违规内容时，将其设为 `BLOCKED`；前端对 `BLOCKED` 内容隐藏或展示“内容不可见”提示。
- 评论删除通过 `comments.status = 'DELETED'` 实现，保留记录以支持治理追溯，前端不展示已删除评论内容。
- “做过”记录删除通过 `cooking_records.status = 'DELETED'` 实现；成品图对象不在删除记录时同步物理删除，便于审计与后续异步清理。
