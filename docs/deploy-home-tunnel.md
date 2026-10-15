# 自家电脑 + Cloudflare Tunnel 部署指南

零成本方案：服务跑在自己电脑的 Docker 里，经 Cloudflare Tunnel 出站隧道暴露公网。
无需公网 IP、无需路由器端口转发，自带 HTTPS。

## 架构

```
Android App ──https──> Cloudflare 边缘 ──Tunnel(出站)──> cloudflared 容器
                                                          ├─ api.域名   → backend:8080 (Go API)
                                                          └─ assets.域名 → minio:9000  (成品图)
本机容器：postgres / redis / minio / backend（端口只绑 127.0.0.1，不对公网开放）
```

## 前置条件

1. 一个域名（任意便宜域名即可，~10 元/年），NS 托管到 Cloudflare（免费套餐）
2. 本机 Docker 可用

## 一次性设置（约 15 分钟）

1. **创建 Tunnel**：Cloudflare Zero Trust 面板 → Networks → Tunnels → Create → Cloudflared，
   复制 token（形如 `eyJh...`）
2. **配置两个 Public Hostname**（在 Tunnel 配置页）：
   | Public hostname | Service |
   |---|---|
   | `api.你的域名` | `http://backend:8080` |
   | `assets.你的域名` | `http://minio:9000` |
3. **填写 `wanli/.env`**（参考 `.env.example`）：
   ```bash
   POSTGRES_PASSWORD=<openssl rand -base64 24>
   MINIO_ROOT_PASSWORD=<openssl rand -base64 24>
   JWT_SECRET=<openssl rand -base64 32>
   TUNNEL_TOKEN=<第 1 步的 token>
   S3_PUBLIC_ENDPOINT=assets.你的域名
   S3_PUBLIC_USE_SSL=true
   AI_API_KEY=<DeepSeek key>
   ```
   ⚠️ 已有数据卷的情况下改 POSTGRES_PASSWORD 不会生效（Postgres 只在首次初始化时读取），
   需 `docker exec wanli-postgres psql -U postgres -c "ALTER USER postgres PASSWORD '新密码'"`
4. **启动**：
   ```bash
   docker compose --profile tunnel up -d
   ```
   backend 容器启动时自动执行数据库迁移（见 `backend-go/docker-entrypoint.sh`）。

## Android 端接入

- 开发（模拟器连本机）：`app/src/main/assets/config.properties` 已指向 `http://10.0.2.2:8081/api/v1/`（宿主机 8080 被本机其他服务占用，故映射 8081）
- 真机/正式包：`frontend-android/local.properties` 加一行
  `BASE_URL=https://api.你的域名/api/v1/`

## 日常运维

| 操作 | 命令 |
|---|---|
| 看后端日志 | `docker logs -f wanli-backend` |
| 更新代码后重启 | `docker compose up -d --build backend` |
| 备份数据库 | `docker exec wanli-postgres pg_dump -U postgres wanli_db > backup.sql` |
| 备份图片 | 备份 minio_data 卷（`docker run --rm -v wanli_minio_data:/data -v $PWD:/b alpine tar czf /b/minio.tar.gz -C /data .`） |
| 开机自启 | 所有服务均 `restart: unless-stopped`；WSL2 需设置 Windows 开机启动（`wsl -d <distro> -u root service docker start` 或 Docker Desktop 自启） |

## 注意

- 电脑关机服务即离线；Cloudflare 边缘会返回 502/503，属正常
- BILIBILI_COOKIE 含 `$` 时需在 .env 中写成 `$$`（docker compose 变量插值规则）
- 注册/登录需配置 `TURNSTILE_SITE_KEY` 与 `TURNSTILE_SECRET_KEY`（Cloudflare 后台创建 widget）；开发期可用官方测试 key `1x00000000000000000000AA`（恒通过），上线前替换为真实 key 并在 widget 中添加生产域名
