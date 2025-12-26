FROM golang:1.24.11-alpine AS builder

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


FROM alpine:latest AS runtime
#FROM debian:stable-slim AS runtime

ENV APP_HOME=/app
ENV PORT=8000
# 环境变量示例，消费使用
ENV CONFIG_PATH=${APP_HOME}/config \
    LOG_PATH=${APP_HOME}/logs \
    TZ=Asia/Shanghai

# 安装运行时依赖并设置时区、用户
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && addgroup -S appgroup \
    && adduser -S -D -H -G appgroup -h ${APP_HOME} -s /sbin/nologin appuser \
    && mkdir -p ${APP_HOME} \
    && chown -R appuser:appgroup ${APP_HOME} \
    && chmod -R 755 ${APP_HOME} \
    && rm -rf /var/cache/apk/*

WORKDIR ${APP_HOME}

COPY --from=builder --chown=appuser:appgroup /out/snowgo ${APP_HOME}/
COPY --from=builder --chown=appuser:appgroup /src/config ${APP_HOME}/config/

EXPOSE ${PORT}
USER appuser

ENTRYPOINT ["/app/snowgo"]
