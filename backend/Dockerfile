# syntax=docker/dockerfile:1.7

FROM golang:1.25.7-alpine

WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT

# 安装必要的工具
RUN apk add --no-cache git

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -trimpath -o main ./cmd/server/

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]
