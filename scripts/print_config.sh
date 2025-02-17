#!/bin/bash

# 加载颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 加载环境变量
source "$(dirname "$0")/load_env.sh"
load_env

# 打印分隔线
print_separator() {
    echo -e "${BLUE}----------------------------------------${NC}"
}

# 打印分组标题
print_group() {
    echo -e "\n${GREEN}[$1]${NC}"
    print_separator
}

# 打印配置项
print_config() {
    local key=$1
    local value="${aspnmy[$key]}"
    echo -e "${YELLOW}$key${NC}=$value"
}

# 按组打印配置
print_group_configs() {
    local group=$1
    shift
    local keys=("$@")
    
    print_group "$group"
    for key in "${keys[@]}"; do
        print_config "$key"
    done
}

# 主函数
main() {
    echo "Ollama Scanner 配置信息"
    echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
    print_separator

    # 基础配置组
    print_group_configs "基础配置" \
        "PORT" "TIMEZONE" "VERSION"

    # 文件路径配置组
    print_group_configs "文件路径配置" \
        "GATEWAY_MAC" "INPUT_FILE" "OUTPUT_FILE" \
        "LOG_PATH" "LOG_DIR" "LOG_LEVEL" "ENABLE_LOG"

    # Telegram配置组
    print_group_configs "Telegram配置" \
        "TELEGRAM_URI" "TELEGRAM_CHAT_ID" "TELEGRAM_BOT_TOKEN"

    # Kafka配置组
    print_group_configs "Kafka配置" \
        "KAFKA_BROKERS" "KAFKA_PORT" "KAFKA_USERNAME" \
        "KAFKA_PASSWORD" "KAFKA_TIMEOUT" "KAFKA_GROUP_ID"

    # HTTP配置组
    print_group_configs "HTTP配置" \
        "HTTP_PORT" "HTTP_MAX_RESPONSE_SIZE" \
        "HTTP_RATE_LIMIT" "HTTP_BURST_LIMIT" \
        "START_PORT" "INSTANCE_COUNT"

    # 构建信息组
    print_group_configs "构建信息" \
        "BUILD_USER" "BUILD_NAME" "BUILD_VERSION" "BUILD_URL"

    # 证书配置组
    print_group_configs "证书配置" \
        "CERT_DIR" "CERT_COUNTRY" "CERT_STATE" \
        "CERT_CITY" "CERT_ORG" "CERT_CN"

    # Caddy配置组
    print_group_configs "Caddy配置" \
        "CADDY_DOMAIN" "CADDY_EMAIL" "CADDY_HTTPS_PORT"

    print_separator
    echo -e "总配置项数量: ${GREEN}${#aspnmy[@]}${NC}"
    echo "配置已加载完成"
}

# 执行主函数
main
