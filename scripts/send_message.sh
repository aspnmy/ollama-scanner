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
    
    # tg相关变量
    TELEGRAM_URI=$(get_config "TELEGRAM_URI")
    encryptTOKEN=$(get_config "TELEGRAM_BOT_TOKEN")  # 加密的 token
    TELEGRAM_CHAT_ID=$(get_config "TELEGRAM_CHAT_ID")

    # 解密 token，注意使用双引号调用变量中的命令
    decryp_token=$("$aspnmy_crypto_dir" decrypt "$encryptTOKEN")
    #log_info "初始化组件变量: TELEGRAM_URI=$TELEGRAM_URI, TELEGRAM_BOT_TOKEN=$decryp_token, TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID"
}

# 记录日志
log_message() {
    local level=$1
    local message=$2
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [$level] $message" >> "$LOG_DIR/send_message.log"
}

# 发送 Telegram 消息
send_telegram_message() {
    local message="$1"
    # 转义特殊字符
    message=$(echo "$message" | sed 's/"/\\"/g')
  
    log_info "正在发送消息到 Telegram..."
    local res
    res=$(curl -s -X POST "$TELEGRAM_URI/bot$decryp_token/sendMessage" \
        -H "Content-Type: application/json" \
        --data-raw "{\"chat_id\":\"$TELEGRAM_CHAT_ID\",\"text\":\"$message\"}")

    if echo "$res" | grep -q "\"ok\":true"; then
        log_info "消息发送成功"
        return 0
    else
        log_error "消息发送失败: $res"
        return 1
    fi
}

# 显示使用帮助
show_usage() {
    echo "Usage: $0 [OPTIONS] MESSAGE"
    echo "Options:"
    echo "  -c, --channel   指定消息发送渠道 (tg)"
    echo "  -m, --message   要发送的消息内容"
    echo "  -h, --help      显示帮助信息"
    echo ""
    echo "Examples:"
    echo "  $0 -c tg -m \"测试消息\""
    echo "  $0 --channel tg --message \"测试消息\""
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--channel)
                CHANNEL="$2"
                shift 2
                ;;
            -m|--message)
                MESSAGE="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                echo "错误: 未知参数 $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

main() {
    init
    parse_args "$@"
    case "$CHANNEL" in
        "tg")
            send_telegram_message "$MESSAGE"
            ;;
        *)
            echo "错误: 不支持的渠道 $CHANNEL"
            log_message "ERROR" "不支持的渠道: $CHANNEL"
            exit 1
            ;;
    esac
}

main "$@"

