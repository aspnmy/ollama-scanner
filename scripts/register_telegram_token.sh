#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}
# ~/.bashrc
# 检查是否已安装 expect
check_expect() {
    if ! command -v expect &> /dev/null; then
        log_info "正在安装 expect..."
        if command -v apt &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y expect
        elif command -v yum &> /dev/null; then
            sudo yum install -y expect
        else
            log_error "无法安装 expect，请手动安装"
            exit 1
        fi
    fi
}

# 添加到系统环境变量
add_to_environment() {
    local token=$1
    local system_wide=${2:-false}

    # 创建 aspnmy 声明和设置脚本
    local aspnmy_script="#!/bin/bash
declare -A aspnmy
aspnmy[TELEGRAM_BOT_TOKEN]='$token'
export aspnmy"

    if [ "$system_wide" = true ]; then
        # 写入系统级配置
        echo "$aspnmy_script" | sudo tee /etc/profile.d/aspnmy_token.sh > /dev/null
        sudo chmod +x /etc/profile.d/aspnmy_token.sh
        log_info "已添加到系统环境变量"
        
        # 为了向后兼容,同时保持传统方式
        echo "export TELEGRAM_BOT_TOKEN='$token'" | sudo tee /etc/profile.d/telegram_token.sh > /dev/null
        sudo chmod +x /etc/profile.d/telegram_token.sh
    else
        # 添加到用户的 .bashrc
        if grep -q "declare -A aspnmy" "$HOME/.bashrc"; then
            # 如果已存在 aspnmy 声明,只更新 token
            sed -i "/aspnmy\[TELEGRAM_BOT_TOKEN\]=/c\aspnmy[TELEGRAM_BOT_TOKEN]='$token'" "$HOME/.bashrc"
        else
            # 如果不存在,添加完整声明
            echo "$aspnmy_script" >> "$HOME/.bashrc"
        fi
        log_info "已添加到用户环境变量"
    fi
    
    # 立即更新当前会话的变量
    declare -A aspnmy 2>/dev/null || true
    aspnmy[TELEGRAM_BOT_TOKEN]="$token"
    export aspnmy
    export TELEGRAM_BOT_TOKEN="$token"  # 保持兼容性
    
    # 验证设置
    if [ "${aspnmy[TELEGRAM_BOT_TOKEN]}" = "$token" ]; then
        log_info "环境变量已更新,aspnmy[TELEGRAM_BOT_TOKEN] 设置成功"
    else
        log_error "环境变量设置失败"
        return 1
    fi
}

# 测试 Token 是否有效
test_token() {
    local token=$1
    local url="https://api.telegram.org/bot$token/getMe"
    local response

    log_info "正在测试 Token 有效性..."
    response=$(curl -s "$url")
    
    if echo "$response" | grep -q "\"ok\":true"; then
        log_info "Token 验证成功"
        return 0
    else
        log_error "Token 验证失败: $response"
        return 1
    fi
}

# 主函数
main() {
    local token=""
    local system_wide=false

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--token)
                token="$2"
                shift 2
                ;;
            -s|--system)
                system_wide=true
                shift
                ;;
            -h|--help)
                echo "用法: $0 [-t TOKEN] [-s]"
                echo "选项:"
                echo "  -t, --token    指定 Telegram Bot Token"
                echo "  -s, --system   添加到系统级环境变量"
                echo "  -h, --help     显示此帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done

    # 如果未提供 token，则提示输入
    if [ -z "$token" ]; then
        read -p "请输入 Telegram Bot Token: " token
    fi

    # 验证 token 格式
    if [[ ! $token =~ ^[0-9]+:[a-zA-Z0-9_-]+$ ]]; then
        log_error "Token 格式无效"
        exit 1
    fi

    # 测试 token
    if ! test_token "$token"; then
        exit 1
    fi

    # 添加到环境变量
    add_to_environment "$token" "$system_wide"

    log_info "环境变量注册完成"
    log_info "请运行 'source ~/.bashrc' 或重新登录以使更改生效"
}

# 执行主函数
main "$@"
