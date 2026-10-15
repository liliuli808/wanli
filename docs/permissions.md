# “碗里” (WanLi) - 访问控制权限矩阵 (permissions.md)

为了保证“碗里 (WanLi)”系统的安全性与数据完整性，Android 客户端与 Go 后端必须严格执行以下权限访问控制。后端 API 必须作为权限控制的绝对拦截层，不能仅依赖 Android 端的界面置灰或按钮隐藏。

---

## 一、 角色定义

在“碗里”系统中，用户主要分为以下四类角色：
1. **游客 (Guest)**：未登录、未注册的用户。
2. **登录用户 (User)**：已通过「邮箱 + 密码 + Turnstile 人机验证」注册并登录的普通注册用户。
3. **所有者/发布者 (Owner/Publisher)**：
   - **来源发布者**：提交某个视频并完成首次 AI 解析校对发布的用户（对应 `recipes.creator_id`）。该用户负责审批他人对自己来源解析版提交的纠错建议。
   - **版本作者**：某份社区改编版（`ADAPTED`）或个人适配版（`KITCHEN_FIT`）的创建者（对应 `recipe_versions.author_id`）。
4. **系统管理员 (Admin)**：具有最高权限的系统运营人员。

> **归属判定速查**
> - 来源纠错审核：校验 `recipes.creator_id == CurrentUser.ID`
> - 改编版/个人适配版修改、删除、公开/取消公开：校验 `recipe_versions.author_id == CurrentUser.ID`
> - 评论删除：校验 `comments.user_id == CurrentUser.ID`
> - “做过”记录删除：校验 `cooking_records.user_id == CurrentUser.ID`

---

## 二、 核心业务权限矩阵

| 功能模块 | 业务操作 | 游客 | 登录用户 | 版本作者/发布者 | 系统管理员 |
| :--- | :--- | :---: | :---: | :---: | :---: |
| **基础浏览** | 查看公开的菜谱、改编版、公开的适配版 | 🟢 允许 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 查看公开的评论、做过记录与成品图 | 🟢 允许 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| **个人厨房** | 查看、修改自己的厨房调料及忌口配置 | ❌ 拒绝 | 🟢 仅限自己 | 🟢 仅限自己 | 🟢 允许（监管）|
| **视频提交** | 粘贴 B 站链接并提交解析任务 | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| **校对与发布**| 修改自己提交的待校对版本并正式公开发布 | ❌ 拒绝 | ❌ 拒绝 | 🟢 仅限自己 | 🟢 允许（代操作）|
| **菜谱改编** | 在公开菜谱基础上创建自己的“改编版” | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 修改自己发布的“改编版”内容 | ❌ 拒绝 | ❌ 拒绝 | 🟢 仅限自己 | 🟢 允许 |
| **来源纠错** | 对他人发布的“来源解析版”提交纠错提案 | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 审批/合入他人对自己来源解析版提交的纠错提案 | ❌ 拒绝 | ❌ 拒绝 | 🟢 仅限来源发布者 | 🟢 允许 |
| **厨房适配** | 手动发起“厨房适配”并生成“个人适配版” | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 将自己的“个人适配版”设置为“公开分享” | ❌ 拒绝 | ❌ 拒绝 | 🟢 仅限版本作者 | 🟢 允许 |
| **互动模块** | 发表评论 | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 删除自己的评论 | ❌ 拒绝 | ❌ 拒绝 | 🟢 仅限评论作者 | 🟢 允许（强制）|
| | 记录“做过”并上传成品图 | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 删除自己的“做过”记录与成品图 | ❌ 拒绝 | ❌ 拒绝 | 🟢 仅限记录作者 | 🟢 允许 |
| **内容治理** | 举报菜谱、评论或图片 | ❌ 拒绝 | 🟢 允许 | 🟢 允许 | 🟢 允许 |
| | 下架违规菜谱/删除违规评论/屏蔽图片 | ❌ 拒绝 | ❌ 拒绝 | ❌ 拒绝 | 🟢 允许 |

**MVP 边界说明**：内容治理中的“举报”与“管理员下架”在 MVP 阶段仅保留接口与权限定义，具体 UI 与管理后台实现放到后续迭代。

---

## 三、 Go 后端权限校验技术规范 (Implementation Guideline)

### 3.1 基于 JWT 的认证中间件
后端必须实现一个 `AuthMiddleware` 提取 HTTP Request Header 中的 `Authorization: Bearer <token>`：
- **未携带 Token 或 Token 已过期**：
  - 如果请求的路由标记为“允许游客”（如 `GET /api/v1/recipe-versions/:id`），则将上下文（Context）中的 `CurrentUser` 设为 `nil`，继续放行。
  - 如果请求的路由属于敏感操作（如 `POST /api/v1/recipe-versions/:id/forks`），立即拦截并返回 `401 Unauthorized` 错误。

### 3.2 级联所属权校验中间件 (Ownership Checks)
对于所有具有 `Owner/Publisher` 专属权限的接口，后端必须在路由层挂载所属权校验拦截器。
- **验证逻辑**：
  1. 从 URL 或 Body 中提取目标资源 ID。
  2. 根据资源类型查询对应所有者字段：
     - 来源纠错审核：`recipes.creator_id`
     - 改编版/个人适配版修改、公开/取消公开：`recipe_versions.author_id`
     - 评论删除：`comments.user_id`
     - “做过”记录删除：`cooking_records.user_id`
  3. 对比 `Context.CurrentUser.ID` 与资源的所有者 ID。若不一致且非管理员，立即拦截并返回 `403 Forbidden`。
- **示例伪代码 (Go Gin)**:
```go
func OnlyOwner(getOwner func(string) (uuid.UUID, error)) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.MustGet("currentUser").(uuid.UUID)
        resourceID := c.Param("id")
        
        ownerID, err := getOwner(resourceID)
        if err != nil || ownerID != userID {
            c.JSON(http.StatusForbidden, gin.H{
                "success": false,
                "error": gin.H{"code": "FORBIDDEN", "message": "无权操作此资源"}
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```
