#!/bin/bash

# 设置 Go 版本
GO_VERSION="1.22.6"

# 检查系统架构
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        GOARCH="amd64"
        ;;
    aarch64)
        GOARCH="arm64"
        ;;
    *)
        echo "不支持的架构: $ARCH"
        exit 1
        ;;
esac

# 设置下载 URL
DOWNLOAD_URL="https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz"

# 创建临时目录
TMP_DIR=$(mktemp -d)
cd $TMP_DIR

echo "下载 Go ${GO_VERSION}..."
wget $DOWNLOAD_URL -O go.tar.gz || {
    echo "下载失败"
    exit 1
}

echo "删除已有的 Go 安装..."
sudo rm -rf /usr/local/go

echo "安装 Go..."
sudo tar -C /usr/local -xzf go.tar.gz
rm go.tar.gz

# 设置环境变量
echo "设置环境变量..."
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh

# 使环境变量生效
source /etc/profile.d/go.sh

# 验证安装
go version

echo "Go ${GO_VERSION} 安装完成！"
