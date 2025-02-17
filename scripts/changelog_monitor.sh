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
    LOG_DIR=$(get_config "LOG_DIR")
    # 将可能包含 ~ 的 LIB_DIR 转换为绝对路径
    BASE_DIR=$(eval echo "$BASE_DIR")
    LIB_DIR=$(eval echo "$LIB_DIR")
    LOG_DIR=$(eval echo "$LOG_DIR")
    changelog_file="$LOG_DIR/changelog"
    # 工具所在路径
    env_loader_dir=${env_loader_dir:-"$LIB_DIR/env_loader/aspnmy_env"}  # 项目 env 路径
    aspnmy_crypto_dir=${aspnmy_crypto_dir:-"$LIB_DIR/crypto/aspnmy_crypto"}  # 项目加密工具路径
}
# 获取最后修改时间
get_last_modified() {
    stat -c %Y "$changelog_file" 2>/dev/null 
}

# 监控变更并发送通知
monitor_changes() {
    local last_modified=""    

    log_info "开始监控变更记录文件: $changelog_file"
    while true; do
        if [ -f "$changelog_file" ]; then
            current_modified=$(get_last_modified "$changelog_file")
            log_info "当前文件最后修改时间: $current_modified"
            if [ "$current_modified" != "$last_modified" ] && [ "$last_modified" != "" ]; then
                log_info "检测到文件变更"
                
                # 读取最新的变更记录
                latest_change=$(tail -n 1 "$changelog_file")
                if [ ! -z "$latest_change" ]; then
                    # 发送通知
                    bash $SCRIPT_DIR/send_message.sh -c tg -m "检测到新的变更记录：$latest_change"
                    log_info "已发送变更通知"
                fi
            fi
            
            last_modified=$current_modified
        fi
        
        sleep 5
    done
}



# 主函数

main() {
    init
    
    # 启动监控
    monitor_changes
}
main