# 视频解析严格字幕拦截设计

## 目标

视频导入任务只有在取得至少一条可用字幕后才能调用 AI。没有字幕轨、字幕内容为空
或字幕拉取失败时，任务必须立即进入 `FAILED`，错误码为
`SUBTITLE_NOT_AVAILABLE`，且不得使用简介或评论替代字幕。

## 数据流

`VideoImportService.Process` 获取视频信息后调用 `FetchBestSubtitle`：

1. 字幕拉取返回错误：记录内部诊断日志，任务立即失败。
2. `hasSubtitle=false` 或字幕分段为空：任务立即失败。
3. 字幕有效：继续读取视频简介和菜谱型评论，作为字幕的补充来源，再调用 AI。

简介和评论不能单独满足解析前置条件。评论获取失败仍不阻断已有字幕的解析流程。
本次不增加 ASR 或网页抓取兜底。

## 错误行为

客户端继续使用现有错误码 `SUBTITLE_NOT_AVAILABLE`，不改变 API 响应结构。用户提示
统一为“该视频没有可用字幕，无法解析”，不再提及简介或评论。

字幕拉取失败的技术原因只写入服务端日志，不透出第三方响应详情。

## 修改范围

- `backend-go/internal/service/video_import_service.go`
  - 在字幕获取后增加严格拦截。
  - 仅在字幕有效时获取评论并组装 AI 输入。
- `backend-go/pkg/apperror/apperror.go`
  - 更新 `SUBTITLE_NOT_AVAILABLE` 的用户提示。
- `backend-go/internal/service/video_import_service_test.go`
  - 覆盖“无字幕但存在简介/菜谱型评论”的场景。
  - 断言任务失败、错误码正确、AI 调用次数为零、无草稿落库。
- `frontend-android/app/src/main/kotlin/com/wanli/android/presentation/home/ImportViewModel.kt`
  和对应单元测试
  - 同步客户端兜底提示。
- `docs/api-contract.md`、`docs/ai-parsing.md`、`docs/user-flows.md`、
  `docs/development-tasks.md`
  - 删除“无字幕时使用简介/评论兜底”的旧描述。
  - 明确简介和评论仅在已有字幕时作为补充来源。

## 测试

后端测试采用可计数的模拟 AI 服务。回归用例提供空字幕轨，同时保留非空简介和符合
菜谱关键词的评论，验证 AI 服务未收到请求。现有成功用例继续验证“字幕 + 简介”
可以完成解析。

完成后运行：

```bash
go fmt ./...
go vet ./...
go test ./...
go build -buildvcs=false ./...
./gradlew testDebugUnitTest assembleRelease lintDebug
```

## 风险与兼容性

- 过去能够仅凭简介或评论进入 AI 解析的视频将改为失败，这是满足“真实来源至上”和
  “字幕失败即时终止”的预期行为。
- API 错误码和响应结构保持不变，不需要数据库迁移。
- 本任务不处理 AI 网关 502 重试；该改动应作为独立任务实施和验证。
