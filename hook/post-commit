#!/bin/bash

# 重新加载环境变量组件以后再保证环境变量生效
aspnmy_envloader reload
# 确保 aspnmy_envloader 组件已加载
source ~/.bashrc

# 颜色定义
GREEN='\033[0;32m'
NC='\033[0m'

# 日志函数
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }

# 初始化环境变量
LOG_DIR=$(get_config "LOG_DIR")
LOG_DIR=$(eval echo "$LOG_DIR")
changelog_file="$LOG_DIR/changelog"

# 创建日志目录（如果不存在）
mkdir -p "$LOG_DIR"

# 获取最新的提交信息
COMMIT_MSG=$(git log -1 --pretty=format:"%h - %s (%an, %ad)" --date=format:"%Y-%m-%d %H:%M:%S")

# 添加时间戳和提交信息到changelog文件
echo "[$(date '+%Y-%m-%d %H:%M:%S')] $COMMIT_MSG" >> "$changelog_file"

log_info "已记录提交信息到 $changelog_file"