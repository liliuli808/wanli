# B 站 Cookie 环境变量配置设计

## 目标

允许后端通过 `BILIBILI_COOKIE` 环境变量携带 B 站登录态，从而读取仅登录用户可见
的字幕轨。未配置 Cookie 时继续使用现有游客请求行为。

## 配置与数据流

新增 `BilibiliConfig`，由配置层读取完整的 `BILIBILI_COOKIE` 字符串。服务装配时
把该值注入 B 站客户端；客户端的统一 GET 请求方法在 Cookie 非空时设置 HTTP
`Cookie` 请求头，因此视频信息、播放器字幕、字幕文件和评论请求使用同一登录态。

保留现有无 Cookie 构造方法，避免测试和其他调用方被迫提供认证信息；新增带 Cookie
的构造入口供生产服务装配使用。

## 安全边界

- Cookie 仅通过后端环境变量或未纳入版本控制的 `backend-go/.env` 注入。
- `.env.example` 只包含空值 `BILIBILI_COOKIE=`，不得填写真实 Cookie。
- Cookie 不写入日志、错误响应、Android 客户端、数据库或 Redis。
- 单元测试只使用虚构 Cookie。
- 建议使用专门的低权限 B 站账号；Cookie 过期或被撤销后需要人工更新并重启后端。

## 修改范围

- `backend-go/internal/config/config.go`
  - 增加 `BilibiliConfig.Cookie` 并读取 `BILIBILI_COOKIE`。
- `backend-go/internal/config/config_test.go`
  - 验证 `.env` 与操作系统环境变量可以注入 Cookie。
- `backend-go/internal/infrastructure/bilibili/client.go`
  - 保存 Cookie，并在非空时为所有统一 GET 请求设置请求头。
  - 保留游客构造方法，新增认证构造方法。
- `backend-go/internal/infrastructure/bilibili/client_test.go`
  - 验证配置 Cookie 时请求头完整透传，未配置时不发送 Cookie。
- `backend-go/internal/server/server.go`
  - 使用配置中的 Cookie 创建 B 站客户端。
- `backend-go/.env.example`、`README.md`、`docs/ai-parsing.md`
  - 展示变量名、登录字幕用途、安全注意事项和重启要求。

## 测试与验收

测试服务器记录收到的 `Cookie` 请求头。测试先证明当前客户端不发送登录态，再验证
认证构造入口对视频信息请求发送与配置完全一致的虚构 Cookie；游客构造入口仍不发送
该请求头。

完成后运行：

```bash
go fmt ./...
go vet ./...
go test ./...
go build -buildvcs=false ./...
./gradlew testDebugUnitTest assembleRelease lintDebug
```

## 风险与限制

- B 站 Cookie 会过期，也可能因风控或接口策略变化而失效；配置 Cookie 不保证账号
  本身无权访问的字幕可见。
- 本次只提供后端登录态透传，不实现扫码登录、Cookie 自动刷新或 Android 端 B 站
  登录。
- 不改变已完成的严格字幕拦截：携带 Cookie 后仍没有可用字幕时，任务继续返回
  `SUBTITLE_NOT_AVAILABLE`，不会调用 AI。
