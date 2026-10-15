#!/bin/sh
# 启动前先执行数据库迁移（ErrNoChange 时 migrate 正常退出），再启动 API 服务。
set -e

echo "[entrypoint] running database migrations..."
/app/migrate up

echo "[entrypoint] starting api server..."
exec /app/api
