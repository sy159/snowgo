FROM golang:1.21 AS builder

# private repository
RUN go env -w GOPRIVATE=github.com/infinitas-plus

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /go/src/app
COPY go.mod go.sum ./
COPY . .

# private repository
#RUN go env -w GOPRIVATE=github.com/xxx

RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/app main.go


FROM alpine:latest
#FROM debian:stable-slim

ENV PROJECT_NAME=snow-service
ENV PORT=8000

WORKDIR /${PROJECT_NAME}
COPY --from=builder /go/bin/app /${PROJECT_NAME}/
COPY --from=builder /go/src/app/config /${PROJECT_NAME}/config/

EXPOSE $PORT

CMD ["./app"]