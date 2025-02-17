#!/bin/bash

# 加载环境变量
source "$(dirname "$0")/load_env.sh"
load_env

# 初始化全局变量
init() {
    # 获取脚本所在目录
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    # 获取项目根目录
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
    # 环境配置文件路径
    ENV_FILE="$PROJECT_ROOT/.env"

    # 检查环境配置文件是否存在
    if [ ! -f "$ENV_FILE" ]; then
        echo "错误: 环境配置文件不存在: $ENV_FILE"
        exit 1
    }

    # 加载环境变量
    set -a
    source "$ENV_FILE"
    set +a

    # 验证必要的环境变量
    if [ -z "$TELEGRAM_URI" ] || [ -z "$TELEGRAM_BOT_TOKEN" ] || [ -z "$TELEGRAM_CHAT_ID" ]; then
        echo "错误: 缺少必要的环境变量配置"
        exit 1
    }

    # 设置日志目录
    LOG_DIR=${LOG_DIR:-"$PROJECT_ROOT/logs"}
    mkdir -p "$LOG_DIR"
}

# 记录日志
log_message() {
    local level=$1
    local message=$2
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [$level] $message" >> "$LOG_DIR/send_message.log"
}

# 发送 Telegram 消息
send_telegram_message() {
    local message=$1
    
    # 从 aspnmy 获取配置
    local uri="${aspnmy[TELEGRAM_URI]}"
    local token=$(get_sensitive_config TELEGRAM_BOT_TOKEN) || exit 1
    local chat_id="${aspnmy[TELEGRAM_CHAT_ID]}"
    
    log_message "INFO" "正在发送消息: $message"
    
    message=$(echo "$message" | sed 's/"/\\"/g')
    res=$(curl -s -X POST "$uri/bot$token/sendMessage" \
        -H "Content-Type: application/json" \
        --data-raw "{\"chat_id\":\"$chat_id\",\"text\":\"$message\"}")
    
    if [ $? -eq 0 ]; then
        log_message "INFO" "消息发送成功: $res"
    else
        log_message "ERROR" "消息发送失败: $res"
    fi
    echo $res
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

    # 验证参数
    if [ -z "$CHANNEL" ] || [ -z "$MESSAGE" ]; then
        echo "错误: 缺少必要参数"
        show_usage
        exit 1
    fi
}

main() {
    # 初始化全局变量
    init

    # 解析命令行参数
    parse_args "$@"

    # 根据渠道发送消息
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

# 执行主函数，传入所有命令行参数
main "$@"
