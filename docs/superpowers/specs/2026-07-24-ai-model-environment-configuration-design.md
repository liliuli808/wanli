# AI 模型环境变量配置设计

## 背景

后端已经读取 `AI_API_KEY`、`AI_BASE_URL` 和 `AI_MODEL`，并在 AI 请求中使用
`AI_MODEL`，默认值为 `deepseek-chat`。当前缺口是 `.env.example` 和项目文档没有
展示 `AI_MODEL`，导致部署者无法直观看到模型配置入口。

## 方案

仅补齐服务端环境变量示例和文档，不增加 Android 设置页面，不允许客户端传入
模型名，也不通过任何接口返回密钥。

- `backend-go/.env.example`：在 AI 配置段增加
  `AI_MODEL=deepseek-chat`。
- `README.md`：在完整环境变量示例中增加 `AI_MODEL`，并说明修改 AI 配置后需要
  重启后端。
- `docs/ai-parsing.md`：说明三个 AI 环境变量的职责、默认模型，以及
  `AI_BASE_URL` 必须兼容 OpenAI Chat Completions 协议。

## 安全边界

- `AI_API_KEY` 只保存在服务端环境中，不写入 Android、响应体或受版本控制的
  `.env` 文件。
- 模型由部署者统一配置，普通用户不能任意指定模型或服务地址。
- 本次不修改 AI Prompt、解析 Schema、数据库或 API 契约。

## 验证

使用文本检查确认示例和两处文档均包含 `AI_MODEL`，然后运行现有 Go 配置包测试，
确认文档改动没有伴随配置回归。

## 风险

环境变量在进程启动时读取，因此修改 `.env` 后必须重启后端才能生效。模型名必须
与 `AI_BASE_URL` 对应的服务商实际支持的模型标识一致，否则调用时会返回服务商
错误。
