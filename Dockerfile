FROM golang:1.25.0-alpine AS builder

# private repository
# RUN go env -w GOPRIVATE=github.com/sy159

ARG GOPRIVATE

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ -n "$GOPRIVATE" ]; then go env -w GOPRIVATE="$GOPRIVATE"; fi && \
    go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/snowgo ./cmd/http


FROM alpine:3.21 AS runtime
#FROM debian:stable-slim AS runtime

# 安装运行时依赖并设置时区
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/cache/apk/*

# config通过docker-compose volume挂载，不打进镜像
ENV APP_HOME=/app \
    PORT=8000 \
    CONFIG_PATH=/app/config \
    LOG_PATH=/app/logs \
    TZ=Asia/Shanghai

WORKDIR ${APP_HOME}

COPY --from=builder /out/snowgo ${APP_HOME}/

EXPOSE ${PORT}

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:${PORT}/healthz || exit 1

ENTRYPOINT ["/app/snowgo"]
