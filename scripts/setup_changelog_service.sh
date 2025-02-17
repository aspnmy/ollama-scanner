#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 设置服务
setup_service() {
    # 创建必要的目录
    mkdir -p /root/gitdata/ollama_scanner/logs
    chmod 755 /root/gitdata/ollama_scanner/logs

    # 设置脚本权限
    chmod +x ./changelog_monitor.sh
    chmod +x ./send_message.sh
    chmod +x ./load_env.sh

    # 复制服务文件
    if [ -f "../systemd/changelog-monitor.service" ]; then
        sudo cp ../systemd/changelog-monitor.service /etc/systemd/system/
        log_info "服务文件已复制"
    else
        log_error "服务文件不存在"
        exit 1
    fi

    # 重载systemd
    sudo systemctl daemon-reload
    log_info "Systemd已重载"
    
    # 启用并启动服务
    sudo systemctl enable changelog-monitor
    sudo systemctl start changelog-monitor
    
    # 检查状态
    if systemctl is-active --quiet changelog-monitor; then
        log_info "服务已成功启动"
    else
        log_error "服务启动失败"
        systemctl status changelog-monitor
        exit 1
    fi
}

# 安装依赖
log_info "正在安装依赖..."
sudo apt-get update
sudo apt-get install -y jq

# 安装服务
log_info "正在安装服务..."
setup_service

log_info "Changelog监控服务安装完成"
