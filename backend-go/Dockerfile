# 碗里后端生产镜像：多阶段构建，最终镜像仅含静态二进制（~30MB）
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
# 离线构建：存在 vendor/（go mod vendor 产物）时跳过网络下载，
# 适配无外网/代理受限环境（如手机经 gnirehtet 构建）
COPY . .
RUN if [ ! -d vendor ]; then go mod download; fi
RUN CGO_ENABLED=0 go build -mod=$(if [ -d vendor ]; then echo vendor; else echo mod; fi) -trimpath -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -mod=$(if [ -d vendor ]; then echo vendor; else echo mod; fi) -trimpath -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/api /out/migrate /app/
COPY migrations ./migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
