# 个人厨房 JSONB 保存 500 修复设计

## 问题与证据

Android“我的厨房”页面点击任一调料或厨具后，会立即调用
`PUT /api/v1/me/kitchen`。调料和厨具共用同一条后端保存路径，因此两类点击均返回
500。

临时诊断后端已稳定复现该问题。PostgreSQL 返回 `SQLSTATE 22P02`：
`invalid input syntax for type json`。GORM 当前将 Go `[]string` 写成
`('盐')` 或 `NULL`，而 `kitchen_profiles.owned_ingredients` 和
`owned_tools` 均为非空 `JSONB` 列。根因是模型只声明了 `type:jsonb`，没有声明
GORM JSON 序列化器。

## 方案比较

1. 在两个 `[]string` 字段上增加 GORM `serializer:json`。改动最小，继续使用现有
   Go 类型和 PostgreSQL `JSONB` 列，不改变 API。采用此方案。
2. 定义实现 `database/sql.Scanner` 和 `driver.Valuer` 的自定义字符串列表类型。
   行为明确，但会引入本次缺陷不需要的自定义类型和维护代码。
3. 改用 `gorm.io/datatypes.JSONSlice[string]`。需要新增直接依赖，并在服务层和
   API DTO 之间转换，超出最小修复范围。

## 代码设计

在 `backend-go/internal/model/kitchen.go` 中，为
`KitchenProfile.OwnedIngredients` 和 `KitchenProfile.OwnedTools` 增加
`serializer:json`，保留 `type:jsonb`。

GORM 保存时把非空列表编码为 JSON 数组，例如 `["盐"]`；空但非 `nil` 的列表编码
为 `[]`。读取时由同一序列化器恢复为 `[]string`。现有数据库中的合法 JSONB 数组
与新模型兼容，无需数据迁移。

Handler、Service、Android 请求模型、API 路径和响应结构均不改变。

## 测试设计

新增 `backend-go/internal/repository/kitchen_repository_integration_test.go`，使用项目
现有的 `WANLI_TEST_DB_DSN` PostgreSQL 集成测试约定：

1. 创建唯一的临时用户。
2. 首次保存同时包含中文调料和厨具的厨房配置。
3. 读取并断言两个列表完整往返。
4. 更新为不同列表，包含空数组场景，再次读取并断言。
5. 测试清理只删除该临时用户，依赖外键级联清理厨房配置。

测试必须先在现有模型上以 `SQLSTATE 22P02` 失败，再增加序列化标记使其通过。

完成后运行：

```bash
go fmt ./...
go vet ./...
go test ./...
go build -buildvcs=false ./...
./gradlew testDebugUnitTest assembleRelease lintDebug
```

## 运行服务替换与验收

构建当前源码为新的临时后端二进制。在停止服务前重新确认 8081 的监听进程属于
WanLi；向该精确 PID 发送 `SIGTERM`，然后用新二进制在 8081 启动。

部署后验证：

1. `GET /api/v1/health` 返回 200。
2. 使用专用临时诊断账号调用 `PUT /api/v1/me/kitchen`，同时保存调料和厨具，返回
   200。
3. 随后调用 `GET /api/v1/me/kitchen`，返回的两个数组与写入值一致。
4. 删除临时诊断账号，避免污染开发数据库。

本次仅更新后端，Android 已安装应用继续使用现有 8081 地址，无需重新安装 APK。

## 风险与恢复

- 8081 重启期间会有数秒不可用。
- 序列化标记只改变两个字段的 GORM 映射；数据库列、接口契约和现有合法 JSONB 数据
  均不变。
- 启动新服务失败时，不继续操作其他进程；保留构建和启动日志，并报告失败。
- 工作区的 `.git` 目录为空，无法创建规格提交或使用 Git 回滚；改动将保持为可直接
  检查的工作区文件。
