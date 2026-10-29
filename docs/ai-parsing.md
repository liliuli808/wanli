# “碗里” (WanLi) - AI 解析与厨房适配 Schema 规范 (ai-parsing.md)

本文档定义“碗里”后端与 AI 模型交互时的输入输出 JSON Schema、字段约束与 Prompt 原则。所有 AI 调用必须使用 **Structured Outputs（结构化 JSON）** 模式，禁止让模型自由输出非 JSON 内容。

---

## B 站字幕登录态

后端可通过 `BILIBILI_COOKIE` 携带 B 站登录态，以读取仅登录用户可见的字幕轨。
该值是完整的浏览器 `Cookie` 请求头，只能保存在服务端环境变量或未纳入版本控制的
本地 `.env` 中，禁止写入日志、API 响应或 Android 客户端。未配置时使用游客模式；
Cookie 过期或被撤销后需要人工更新并重启后端。

```env
BILIBILI_COOKIE=
```

### 用户级授权（优先于全局 Cookie）

用户在 App「我的 → B 站账号」中扫码登录或粘贴 `SESSDATA` 后，后端会先用
`x/web-interface/nav` 校验，再用 `BILIBILI_CREDENTIAL_ENC_KEY`（留空则由
`JWT_SECRET` 派生）加密保存到 `users.bilibili_sessdata_enc`。解析任务创建时优先使用
该用户的登录态，没有授权时才回落到全局 `BILIBILI_COOKIE`。

- 凭证永不出现在 API 响应和日志里；用户可随时 `DELETE /api/v1/me/bilibili-credential`。
- 轮换加密密钥会使已授权凭证失效，需要用户重新授权。
- 字幕接口 `x/player/wbi/v2` 会带 wbi 签名（`wts` + `w_rid`），密钥来自 `nav` 的
  `wbi_img`，缓存 1 小时，`-403` 时自动刷新重试。

登录态只影响字幕可见性，不改变严格字幕拦截：携带 Cookie 后仍没有可用字幕时，
任务返回 `SUBTITLE_NOT_AVAILABLE`；明确返回 `need_login_subtitle` 且当前没有可用
登录态时，任务返回 `BILIBILI_LOGIN_REQUIRED` 提示用户授权。两种情况都不会调用 AI。

---

## AI 服务配置

后端通过以下环境变量配置 AI 服务：

- `AI_API_KEY`：服务商访问凭据，仅保存在服务端，禁止写入 Android 客户端或
  API 响应。
- `AI_BASE_URL`：兼容 OpenAI Chat Completions 协议的服务地址，默认值为
  `https://api.deepseek.com/v1`。
- `AI_MODEL`：发送给服务商的模型标识，默认值为 `deepseek-chat`。

本地开发配置示例：

```env
AI_API_KEY=
AI_BASE_URL=https://api.deepseek.com/v1
AI_MODEL=deepseek-chat
```

这些配置在后端进程启动时读取。修改后必须重启后端服务，并确保 `AI_MODEL` 是
`AI_BASE_URL` 对应服务商实际支持的模型标识。

---

## 一、 通用要求

1. **字幕是强制来源**：AI 解析必须取得 B 站 CC/AI 字幕（原始文本 + 时间戳）。没有字幕轨、字幕内容为空或字幕拉取失败时，任务直接失败（`SUBTITLE_NOT_AVAILABLE`）。**禁止使用简介/评论替代字幕，也禁止回退到语音转写（ASR）**。
2. **补充文本来源**：已取得字幕时，可附加视频简介、置顶/高赞菜谱型评论作为补充；与字幕冲突时以字幕为准。
3. **真实来源至上**：
   - 不得凭空捏造未在字幕或视频中明确出现的食材、用量、火候、步骤。
   - 无法确认的量必须输出为 `null`，不可估算为具体数字。
   - “少许、适量、一点点、一些”等模糊表述必须**原样保留**在 `amount` 字段中。
   - 若 AI 基于常识补充了字幕未提及但做菜必需的食材/步骤，必须设置 `is_explicit = false`。
4. **时间戳精度**：`video_timestamp` 以秒为单位，取该步骤在视频中首次出现的字幕时间。
5. **模型选择**：优先使用支持 Structured Outputs 的模型（如 OpenAI GPT-4o / Claude / DeepSeek）。

---

## 二、 视频字幕 → 结构化菜谱（AI 解析）

### 2.1 输入格式

后端将字幕（可选含简介与评论补充）整理为如下格式提交给 AI：

```json
{
  "bvid": "BV1xx411c7xx",
  "title": "视频标题（可选，用于校验菜名）",
  "subtitles": [
    { "start": 0, "end": 5, "text": "今天做一道家常红烧肉" },
    { "start": 6, "end": 12, "text": "准备五花肉500克" },
    { "start": 15, "end": 20, "text": "冷水下锅焯水" }
  ],
  "description": "视频简介（可选，仅在已有字幕时作为补充）",
  "comments": ["置顶/高赞评论（可选，仅在已有字幕时作为补充）"]
}
```

- `subtitles` 必须包含至少一条可用字幕；没有字幕轨、字幕内容为空或字幕拉取失败时，任务直接失败。
- 简介与评论仅作字幕的补充：与字幕冲突时以字幕为准；缺乏字幕支撑的补充内容应体现不确定性（如将相关 `amount` 置为 `null`）。

### 2.2 输出 JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["title", "servings", "estimated_minutes", "ingredients", "steps", "instructions"],
  "properties": {
    "title": { "type": "string", "description": "菜名" },
    "servings": { "type": ["string", "null"], "description": "适合分量，如“2-3人份”" },
    "estimated_minutes": { "type": ["integer", "null"], "description": "预计烹饪耗时（分钟）" },
    "ingredients": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "category", "is_explicit"],
        "properties": {
          "name": { "type": "string", "description": "食材/调料名称" },
          "amount": { "type": ["string", "null"], "description": "用量；不确定时为 null；保留“少许/适量”" },
          "category": { "type": "string", "enum": ["MAIN", "AUXILIARY", "SEASONING"], "description": "主料/辅料/调味品" },
          "is_explicit": { "type": "boolean", "description": "true=字幕明确提到；false=AI补充" }
        }
      }
    },
    "steps": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["step_order", "description"],
        "properties": {
          "step_order": { "type": "integer", "description": "步骤序号，从1开始" },
          "description": { "type": "string", "description": "步骤描述" },
          "video_timestamp": { "type": ["integer", "null"], "description": "对应视频时间戳（秒）" },
          "duration_seconds": { "type": ["integer", "null"], "description": "该步骤预计用时（秒）" },
          "heat": { "type": ["string", "null"], "description": "火候，如“大火”“中小火”" }
        }
      }
    },
    "instructions": { "type": ["string", "null"], "description": "提示与注意事项" }
  }
}
```

### 2.3 Prompt 核心约束（必须写入 system prompt）

```text
你是一位严谨的家庭菜谱结构化助手。请根据用户提供的视频字幕（可选含视频简介与评论），生成一份结构化菜谱 JSON。

来源优先级：
1. 字幕内容最可信，包含操作步骤和顺序。
2. 视频简介中的材料和配方可信，常包含文字版菜谱。
3. 置顶/高赞评论只作补充参考；若只是用户评价，不要当成菜谱步骤。
4. 不同来源冲突时，以字幕为准。

规则：
1. 必须完全忠实于来源文本，不得编造未提及的用量、火候或步骤。
2. 当来源没有给出具体用量时，amount 字段必须设为 null；若来源使用“少许、适量、一点点”等模糊词，请原样保留。
3. 若你补充了来源未明确提及但菜品必需的食材/步骤，必须设置 is_explicit = false。
4. 每个步骤的 video_timestamp 必须是该步骤在字幕中首次出现的时间（秒）；无字幕来源或无法对应时设为 null。
5. 仅来源于简介/评论、缺乏来源支撑的内容，相关 amount 置为 null，并在 instructions 中说明不确定性。
6. 如果无法从来源中提取有效菜谱，返回 ingredients 和 steps 为空数组的 JSON。
7. 不要输出任何解释性文字，只输出符合 schema 的 JSON。
```

### 2.4 失败处理

- 若 AI 返回的 JSON 无法通过 schema 校验，或关键字段（title/ingredients/steps）缺失，任务状态流转为 `FAILED`，错误码 `AI_PARSE_FAILED`。
- 允许提交者在 App 内重试一次。

---

## 三、 厨房适配（手动发起）

### 3.1 输入格式

```json
{
  "recipe_version_id": "7831f28b-b8c2-48df-b59a-09dcfcb82d1c",
  "original_recipe": {
    "title": "家常红烧肉",
    "servings": "2-3人份",
    "estimated_minutes": 45,
    "ingredients": [
      { "name": "五花肉", "amount": "500克", "category": "MAIN", "is_explicit": true },
      { "name": "冰糖", "amount": "30克", "category": "SEASONING", "is_explicit": true }
    ],
    "steps": [
      { "step_order": 1, "description": "五花肉切块焯水", "video_timestamp": 45, "duration_seconds": 300, "heat": null }
    ],
    "instructions": "红烧酱油可替换为老抽+生抽混合"
  },
  "kitchen_profile": {
    "owned_ingredients": ["盐", "生抽", "老抽", "料酒"],
    "owned_tools": ["炒锅", "电饭煲"]
  },
  "dietary_restrictions": [
    { "restriction_type": "ALLERGY", "ingredient_name": "花生" },
    { "restriction_type": "DISLIKE", "ingredient_name": "香菜" },
    { "restriction_type": "MEDICAL", "ingredient_name": "糖" }
  ]
}
```

### 3.2 输出 JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["safe", "title", "servings", "estimated_minutes", "ingredients", "steps", "instructions", "adaptation_notes"],
  "properties": {
    "safe": { "type": "boolean", "description": "true=适配成功；false=存在无法安全处理的过敏/忌口" },
    "block_reason": { "type": ["string", "null"], "description": "safe=false 时必填，说明拦截原因" },
    "title": { "type": "string", "description": "适配后的菜名，通常原菜名+适配说明" },
    "servings": { "type": ["string", "null"] },
    "estimated_minutes": { "type": ["integer", "null"] },
    "ingredients": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "category", "is_explicit", "substitute_for"],
        "properties": {
          "name": { "type": "string" },
          "amount": { "type": ["string", "null"] },
          "category": { "type": "string", "enum": ["MAIN", "AUXILIARY", "SEASONING"] },
          "is_explicit": { "type": "boolean" },
          "substitute_for": { "type": ["string", "null"], "description": "被替换的原食材名；非替换料为 null" }
        }
      }
    },
    "steps": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["step_order", "description"],
        "properties": {
          "step_order": { "type": "integer" },
          "description": { "type": "string" },
          "video_timestamp": { "type": ["integer", "null"] },
          "duration_seconds": { "type": ["integer", "null"] },
          "heat": { "type": ["string", "null"] }
        }
      }
    },
    "instructions": { "type": ["string", "null"] },
    "adaptation_notes": {
      "type": "object",
      "required": ["missing_ingredients", "missing_tools", "substitutions", "flavor_changes"],
      "properties": {
        "missing_ingredients": { "type": "array", "items": { "type": "string" }, "description": "用户厨房缺少且无法替代的必要调料/食材" },
        "missing_tools": { "type": "array", "items": { "type": "string" }, "description": "用户厨房缺少的厨具" },
        "substitutions": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["original", "replacement", "reason"],
            "properties": {
              "original": { "type": "string" },
              "replacement": { "type": "string" },
              "reason": { "type": "string" }
            }
          }
        },
        "flavor_changes": { "type": "array", "items": { "type": "string" }, "description": "因替换导致的口味变化说明" }
      }
    }
  }
}
```

### 3.3 Prompt 核心约束（必须写入 system prompt）

```text
你是一位注重安全的家庭菜谱适配助手。用户会提供一份原菜谱、他的厨房调料/厨具清单以及忌口/过敏信息。请生成一份适合该用户当前厨房条件的菜谱 JSON。

安全规则（最高优先级）：
1. 如果原菜谱包含用户明确过敏的食材，且你无法用非过敏食材安全替代，必须设置 safe = false，并给出 block_reason。
2. 适配后的菜谱正文和配料表中，绝不能出现用户过敏的食材或含该过敏源的制品。
3. 对于医疗饮食忌口（如控糖、控盐），优先寻找安全替代方案；若无法替代则设置 safe = false。

适配规则：
1. 只能在用户已有调料/厨具或安全替代品范围内调整，禁止发明不存在的烹饪步骤。
2. 如果用户缺少某必要调料/厨具且无法替代，记录到 adaptation_notes.missing_* 中，并相应调整步骤。
3. 对每一次替换，在 adaptation_notes.substitutions 中说明原食材、替换品和原因。
4. 不得凭空估算用量；不确定时 amount 设为 null。
5. 只输出符合 schema 的 JSON。
```

### 3.4 失败处理

- 若 `safe = false`，后端直接返回 `ALLERGY_BLOCKED` 或 `DIETARY_BLOCKED` 错误，不保存任何新版本。
- 即使 AI 返回 `safe = true`，后端仍必须扫描适配后实际会被烹饪的标题、食材名称、正文提示和步骤；命中任一 `ALLERGY` 词条时返回 `ALLERGY_BLOCKED`，不得落库。`substitute_for` 和替换说明可以保留原过敏食材名称用于解释，不视为实际用料。
- 若 `safe = true`，后端将结果保存为 `recipe_versions` 中 `version_type = 'KITCHEN_FIT'`、`is_public = FALSE` 的版本。

---

## 四、 版本对比与变更说明

在纠错建议、改编、适配等场景中，若需要向用户展示“改了什么”，后端应通过对比两个 `recipe_versions` 的 JSON 内容生成增量，而不是依赖 AI 自由描述。AI 仅在 `change_log` 字段提供简短的人类可读说明。

---

## 五、 变更记录

| 日期 | 变更人 | 说明 |
| :--- | :--- | :--- |
| 2026-07-23 | AI 助手 | 初稿：定义解析与适配 Schema、Prompt 约束、失败处理 |
| 2026-07-24 | AI 助手 | 放宽输入来源：字幕优先，允许简介/评论兜底；三者均不可用才判定失败（仍禁止 ASR） |
| 2026-07-24 | AI 助手 | 恢复严格字幕拦截：无可用字幕时立即失败；简介/评论仅在已有字幕时作为补充 |
