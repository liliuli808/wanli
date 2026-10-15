# Android 后端端口修正设计

## 背景

Android 客户端当前请求 `http://10.0.2.2:8080/api/v1/`。宿主机的
`8080` 端口由 Uvicorn 服务占用并对 WanLi 健康检查返回 404；WanLi Go
后端实际监听 `8081`，其 `/api/v1/health` 返回 200。

## 方案比较

1. 将 Android `BASE_URL` 端口改为 `8081`。这是最小、低风险且与当前运行环境一致的方案。
2. 停止占用 `8080` 的服务并让 WanLi 后端改回 `8080`。这会影响项目范围外的进程，不采用。
3. 引入 Build Variant 管理不同环境地址。当前只有一个本地开发地址，超出本次修复范围，不采用。

## 设计

仅修改 `frontend-android/app/src/main/assets/config.properties`，保留
`10.0.2.2`、API 前缀和结尾斜杠，只将端口从 `8080` 改为 `8081`。
网络权限、明文 HTTP 策略和后端路由均不改变。

## 验证

修改前用精确配置断言复现失败，修改后再次运行同一断言；随后执行 Android
Debug APK 构建，并检查 APK 内打包的 `assets/config.properties`。

## 风险

配置会内嵌进 APK，MuMu 中已安装的旧版本不会自动更新，必须重新安装新 APK。
本次修复不验证 MuMu 是否支持 `10.0.2.2`；如果新 APK 仍无法连接，需要再单独
检查 MuMu 的宿主机地址映射。
