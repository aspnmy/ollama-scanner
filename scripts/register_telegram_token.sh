#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'
# 重新加载环境变量组件以后再保证环境变量生效
aspnmy_envloader reload
# 确保 aspnmy_envloader 组件已加载（例如：source ~/.bashrc）
source ~/.bashrc
# 日志函数
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

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
}

save_token() {
    local utoken=$1
    
    if [ -z "$utoken" ]; then
        log_error "token 不能为空"
        return 1
    fi

    encryp_token=$("$aspnmy_crypto_dir" encrypt "$utoken")
    if [ $? -ne 0 ]; then
        log_error "token 加密失败"
        return 1
    fi

    # 如果.env文件存在，删除原有的TELEGRAM_BOT_TOKEN行
    if [ -f "$BASE_DIR/.env" ]; then
        sed -i '/^TELEGRAM_BOT_TOKEN=/d' "$BASE_DIR/.env"
    fi

    #log_info "加密后的token: $encryp_token"
    # 使用换行符确保新token另起一行
    echo -e "\nTELEGRAM_BOT_TOKEN=$encryp_token" >> "$BASE_DIR/.env"
    
    res=$("$env_loader_dir" reload)
    # 重新加载后需重新生效去除上一次缓存避免出现错误
    source ~/.bashrc
    if [ $? -eq 0 ]; then
        
        newtoken=$(get_config "TELEGRAM_BOT_TOKEN")
        decryp_token=$("$aspnmy_crypto_dir" decrypt "$newtoken")
        log_info "解密的toker : $decryp_token"
        return 0
    else
        log_error "token保存失败"
        return 1
    fi
    if [ $? -eq 0 ]; then
        log_info "token保存成功"
        BASE_DIR=$(get_config "ollama_scannerBaseDir")
        return 0
    else
        log_error "token保存失败"
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
    init
    local token=""
  

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--token)
                token="$2"
                shift 2
                ;;
            -s|--save)
                token="$2"
                if [ -z "$token" ]; then
                    log_error "使用 -s/--save 选项时必须提供 token"
                    exit 1
                fi
                save_token "$token"
                exit 0
                ;;
            -h|--help)
                echo "用法: $0 [-t TOKEN] [-s]"
                echo "选项:"
                echo "  -t, --token    指定 Telegram Bot Token"
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

    
}

# 执行主函数
main "$@"
