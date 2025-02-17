#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'
# 重新加载环境变量组件以后再保证环境变量生效
aspnmy_envloader reload
# 确保 aspnmy_envloader 组件已加载（例如：source ~/.bashrc）
source ~/.bashrc
# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}
# 初始化全局变量及组件必须变量
init() {
    
    # 获取脚本所在目录
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    # 调用 get_config 函数获取组件必须变量
    BASE_DIR=$(get_config "ollama_scannerBaseDir")  # 项目根目录
    LIB_DIR=$(get_config "ollama_scannerLibDir")     # 项目 lib 目录
    # 将可能包含 ~ 的 LIB_DIR 转换为绝对路径
    BASE_DIR=$(eval echo "$BASE_DIR")
    LIB_DIR=$(eval echo "$LIB_DIR")
    # 工具所在路径
    env_loader_dir=${env_loader_dir:-"$LIB_DIR/env_loader/aspnmy_env"}  # 项目 env 路径
    aspnmy_crypto_dir=${aspnmy_crypto_dir:-"$LIB_DIR/crypto/aspnmy_crypto"}  # 项目加密工具路径

    # 检查加密工具是否存在并可执行
    if [ ! -x "$aspnmy_crypto_dir" ]; then
        echo "错误: 加密工具未找到或不可执行: $aspnmy_crypto_dir"
        exit 1
    fi

    # 日志所在目录
    LOG_DIR=${LOG_DIR:-"$BASE_DIR/logs"}
    mkdir -p "$LOG_DIR"    
    chmod 755 "$LOG_DIR"
}
# 设置服务
setup_service() {

    # 复制服务文件
    if [ -f "$BASE_DIR/systemd/changelog-monitor.service" ]; then
        sudo cp $BASE_DIR/systemd/changelog-monitor.service /etc/systemd/system/
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



main(){
    init
    # 安装服务
    log_info "正在安装服务..."
    setup_service

    log_info "Changelog监控服务安装完成"

}
main