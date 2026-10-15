# 发现页重复键闪退修复设计

## 问题与证据

`GET /api/v1/recipes` 当前返回了两条 `recipe_id` 相同、`recipe_version_id`
不同的记录。Android 发现页使用 `recipe_id` 作为 `LazyColumn` 唯一键，
因此列表布局时会触发 Compose 重复键异常。

`8081` 上的 `/tmp/wanli-api` 进程启动时间早于当前后端列表去重查询源码，
而且其可执行文件已经被删除，说明运行服务未包含当前实现。

## 方案比较

1. 只重启后端：能纠正当前响应，但客户端以后收到异常重复数据仍会闪退。
2. 只修改 Android 键：能避免闪退，但旧后端仍会展示重复菜谱。
3. 同时更新运行后端并为 Android 使用版本唯一键：修复当前根因，并防止异常响应
   再次导致客户端崩溃。采用此方案。

## Android 设计

新增 `discoverItemKey(item: RecipeSummaryDto): String`，返回非空且按契约唯一的
`recipe_version_id`。发现页 `LazyColumn` 通过该函数生成键。

回归测试构造两个 `recipe_id` 相同、`recipe_version_id` 不同的条目，验证得到
不同列表键。测试先失败，再实现最小改动使其通过。

## 后端部署设计

不修改后端查询源码。先执行 Go 测试、静态检查和构建，再向已确认占用 `8081`
的旧 WanLi 进程发送 `SIGTERM`，使用当前源码构建的新二进制在同一端口启动。
服务从 `backend-go/.env` 读取现有基础设施配置，并显式使用 `PORT=8081`。

启动后验证：

- `/api/v1/health` 返回 200。
- `/api/v1/recipes` 中不再存在重复 `recipe_id`。
- 监听 `8081` 的进程是新 `/tmp/wanli-api`。

## 风险与恢复

重启期间会有数秒不可用。服务使用现有优雅退出和 Redis 卡死任务恢复机制。
如果新进程启动失败，保留构建日志并立即报告，不继续修改其他组件。

API 请求、响应结构与数据库结构均不改变，因此无需修改 `docs/api-contract.md`
或迁移文件。
