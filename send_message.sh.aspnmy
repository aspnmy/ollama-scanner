#!/bin/bash
# 初始化全局变量
init() {
    local env_file="./env.json"

    # 检查 env.json 文件是否存在
    if [ ! -f "$env_file" ]; then
        echo "警告: env.json 文件不存在,使用默认值."
        return
    fi

    # 使用 jq 读取 env.json 文件并更新全局变量
    if command -v jq &> /dev/null; then
        builduser=$(jq -r '.builduser // empty' "$env_file")
        if [ -n "$builduser" ]; then
            echo "从 env.json 读取 builduser: $builduser"
        else
            builduser="aspnmy"
        fi

        buildname=$(jq -r '.buildname // empty' "$env_file")
        if [ -n "$buildname" ]; then
            echo "从 env.json 读取 buildname: $buildname"
        else
            buildname="ollama-scanner"
        fi

        buildver=$(jq -r '.buildver // empty' "$env_file")
        if [ -n "$buildver" ]; then
            echo "从 env.json 读取 buildver: $buildver"
        else
            buildver="v2.2"
        fi

        buildurl=$(jq -r '.buildurl // empty' "$env_file")
        if [ -n "$buildurl" ]; then
            echo "从 env.json 读取 buildurl: $buildurl"
        else
            buildurl="docker.io"
        fi
        TELEGRAM_URI=$(jq -r '.TELEGRAM_URI // empty' "$env_file")
        if [ -n "$TELEGRAM_URI" ]; then
            echo "从 env.json 读取 TELEGRAM_URI: $TELEGRAM_URI"
        else
            TELEGRAM_URI="ELEGRAM_URI"
        fi
        TELEGRAM_BOT_TOKEN=$(jq -r '.TELEGRAM_BOT_TOKEN // empty' "$env_file")
        if [ -n "$TELEGRAM_BOT_TOKEN" ]; then
            echo "从 env.json 读取 TELEGRAM_BOT_TOKEN: $TELEGRAM_BOT_TOKEN"
        else
            TELEGRAM_BOT_TOKEN="your_telegram_bot_token"
        fi

        TELEGRAM_CHAT_ID=$(jq -r '.TELEGRAM_CHAT_ID // empty' "$env_file")
        if [ -n "$TELEGRAM_CHAT_ID" ]; then
            echo "从 env.json 读取 TELEGRAM_CHAT_ID: $TELEGRAM_CHAT_ID"
        else
            TELEGRAM_CHAT_ID="your_telegram_chat_id"
        fi

        release_dir=$(jq -r '.release_dir // empty' "$env_file")
        if [ -n "$release_dir" ]; then
            echo "从 env.json 读取 release_dir: $release_dir"
        else
            release_dir="Releases/$buildver"
        fi
    else
        echo "错误: jq 未安装,无法解析 env.json 文件."
        exit 1
    fi

    # 打印初始化后的全局变量
    echo "初始化全局变量:"
    echo "builduser=$builduser"
    echo "buildname=$buildname"
    echo "buildver=$buildver"
    echo "buildurl=$buildurl"
    echo "TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN"
    echo "TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID"
    echo "release_dir=$release_dir"
}
# 发送 Telegram 消息
send_telegram_message() {
    local message=$1
    # 转义特殊字符
    message=$(echo "$message" | sed 's/"/\\"/g')
    echo "TELEGRAM_URI: $TELEGRAM_URI"
    res=$(curl -s -X POST "$TELEGRAM_URI/bot$TELEGRAM_BOT_TOKEN/sendMessage" \
        -H "Content-Type: application/json" \
        --data-raw "{\"chat_id\":\"$TELEGRAM_CHAT_ID\",\"text\":\"$message\"}")
    echo $res
}


main() {
    # 初始化全局变量
    init
    tag="ollama_scanner:v2.2-dockerfile-zmap-arm64"
    send_telegram_message "✅ 镜像推送成功:$tag"
}
main