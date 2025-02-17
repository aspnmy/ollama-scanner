#!/bin/bash

# 声明全局关联数组
declare -A aspnmy

# 从 .bashrc 卸载
uninstall_from_bashrc() {
    # 删除旧配置
    sed -i '/^# BEGIN ASPNMY CONFIG/,/^# END ASPNMY CONFIG/d' ~/.bashrc
    
    # 清理环境
    unset -f get_config
    unset aspnmy
    
    echo "配置已从 .bashrc 卸载"
}

# 修改生成脚本函数
generate_aspnmy_script() {
    cat << 'EOF'
# BEGIN ASPNMY CONFIG
declare -A aspnmy

EOF
    
    # 写入变量
    for key in "${!aspnmy[@]}"; do
        value="${aspnmy[$key]}"
        # 转义值中的特殊字符
        value=${value//\\/\\\\}
        value=${value//\'/\\\'}
        echo "aspnmy['$key']='$value'"
    done
    
    # 写入函数定义
    cat << 'EOF'

# 导出变量和函数
export aspnmy

# 环境变量获取函数
get_config() {
    local key="$1"
    echo "${aspnmy[$key]:-${!key}}"
}
export -f get_config

# END ASPNMY CONFIG
EOF
}

# 简化加载函数
load_env() {
    # 清理现有数组
    unset aspnmy
    declare -A aspnmy
    
    # 查找 .env 文件
    local current_dir="$PWD"
    while [[ ! -f "$current_dir/.env" && "$current_dir" != "/" ]]; do
        current_dir="$(dirname "$current_dir")"
    done

    if [[ ! -f "$current_dir/.env" ]]; then
        echo "错误: 未找到 .env 文件"
        return 1
    fi

    # 读取并解析 .env 文件
    while IFS= read -r line; do
        # 跳过注释和空行
        [[ $line =~ ^[[:space:]]*# || -z $line ]] && continue
        
        # 提取键值对
        if [[ $line =~ ^([A-Za-z0-9_]+)[[:space:]]*=[[:space:]]*(.*)$ ]]; then
            key="${BASH_REMATCH[1]}"
            value="${BASH_REMATCH[2]}"
            
            # 清理值（保留空值）
            value=$(echo "$value" | sed -e 's/^[[:space:]"'"'"']*//g' -e 's/[[:space:]"'"'"']*$//g')
            
            # 存储到数组
            aspnmy["$key"]="$value"
            export "$key"="$value"
        fi
    done < "$current_dir/.env"

    # 生成并安装配置
    generate_aspnmy_script > /tmp/aspnmy_config
    # 安装新配置
    sed -i '/^# BEGIN ASPNMY CONFIG/,/^# END ASPNMY CONFIG/d' ~/.bashrc
    cat /tmp/aspnmy_config >> ~/.bashrc
    rm -f /tmp/aspnmy_config

    # 立即生效
    . ~/.bashrc

    echo "成功加载 ${#aspnmy[@]} 个变量"
    return 0
}

# 简化打印函数
print_all_vars() {
    # 颜色定义
    local GREEN='\033[0;32m'
    local YELLOW='\033[1;33m'
    local BLUE='\033[0;34m'
    local NC='\033[0m'
    
    # 重新加载变量（如果需要）
    if [ ${#aspnmy[@]} -eq 0 ]; then
        load_env >/dev/null
        . ~/.bashrc
    fi
    
    echo -e "${BLUE}========== ASPNMY 环境变量 ==========${NC}"
    echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo -e "${BLUE}------------------------------------${NC}"
    
    # 简化变量打印
    local keys=(${!aspnmy[@]})
    IFS=$'\n' sorted_keys=($(sort <<<"${keys[*]}"))
    unset IFS
    
    for key in "${sorted_keys[@]}"; do
        echo -e "${GREEN}$key${NC}=${YELLOW}${aspnmy[$key]:-<empty>}${NC}"
    done
    
    echo -e "${BLUE}------------------------------------${NC}"
    echo -e "共计: ${GREEN}${#aspnmy[@]}${NC} 个变量"
}

# 命令行解析
case "${1:-}" in
    "install")
        load_env
        ;;
    "uninstall")
        uninstall_from_bashrc
        ;;
    "reload")
        uninstall_from_bashrc
        load_env
        ;;
    "print")
        if [ ${#aspnmy[@]} -eq 0 ]; then
            load_env >/dev/null 2>&1
        fi
        print_all_vars
        ;;
    *)
        echo "用法: $0 {install|uninstall|reload|print}"
        echo "  install   - 加载并安装环境变量"
        echo "  uninstall - 卸载环境变量"
        echo "  reload    - 重新加载环境变量"
        echo "  print     - 打印所有环境变量"
        exit 1
        ;;
esac