#!/bin/bash

# 加载环境变量
source "$(dirname "$0")/load_env.sh"

# 设置日志文件
LOG_FILE="$LOG_DIR/changelog_monitor.log"

# 记录日志
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
    echo "$1"
}

# 获取最后修改时间
get_last_modified() {
    stat -c %Y "$CHANGELOG_FILE" 2>/dev/null || echo "0"
}

# 监控变更并发送通知
monitor_changes() {
    local last_modified=""
    local changelog_file="$LOG_DIR/changelog.json"

    log "开始监控变更记录文件: $changelog_file"

    while true; do
        if [ -f "$changelog_file" ]; then
            current_modified=$(get_last_modified "$changelog_file")
            
            if [ "$current_modified" != "$last_modified" ] && [ "$last_modified" != "" ]; then
                log "检测到文件变更"
                
                # 读取最新的变更记录
                latest_change=$(tail -n 1 "$changelog_file")
                if [ ! -z "$latest_change" ]; then
                    # 发送通知
                    ./send_message.sh -c tg -m "检测到新的变更记录：$latest_change"
                    log "已发送变更通知"
                fi
            fi
            
            last_modified=$current_modified
        fi
        
        sleep 5
    done
}

# 创建必要的目录
mkdir -p "$LOG_DIR"

# 启动监控
monitor_changes
