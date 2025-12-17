FROM golang:1.24.10 AS builder

# private repository
# RUN go env -w GOPRIVATE=github.com/sy159

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /go/src/app
COPY go.mod go.sum ./
COPY . .

RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/app ./cmd/http


FROM alpine:latest
#FROM debian:stable-slim

# 安装根证书，防止部分请求失败
RUN apk --no-cache add ca-certificates tzdata

ENV PROJECT_NAME=snowgo-service
ENV PORT=8000

WORKDIR /${PROJECT_NAME}
COPY --from=builder /go/bin/app /${PROJECT_NAME}/
COPY --from=builder /go/src/app/config /${PROJECT_NAME}/config/

EXPOSE $PORT

CMD ["./app"]
