# 构建阶段
FROM golang:1.21.6-alpine AS builder

# 安装系统依赖
RUN apk add --no-cache \
    gcc \
    musl-dev \
    linux-headers \
    git

# 设置工作目录
WORKDIR /build

# 复制 go.mod 和 go.sum
COPY Src/go.mod Src/go.sum ./
RUN go mod download

# 复制源码
COPY Src/ ./

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -o ollama_scanner

# 运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    zmap \
    masscan \
    sudo \
    ca-certificates \
    tzdata

# 设置时区
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 设置工作目录
WORKDIR /app

# 复制编译产物和配置文件
COPY --from=builder /build/ollama_scanner .
COPY Src/.env ./.env

# 创建数据目录
RUN mkdir -p data

# 创建非 root 用户
RUN adduser -D -u 1000 scanner && \
    chown -R scanner:scanner /app

# 切换到非 root 用户
USER scanner

# 暴露端口
EXPOSE 11434

# 设置入口点
ENTRYPOINT ["/app/ollama_scanner"]
