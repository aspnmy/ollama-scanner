#!/bin/bash

# 全局变量（默认值）
builduser="aspnmy" # 镜像仓库用户名
buildname="ollama-scanner" # 镜像名称
buildver="v2.2" # 镜像版本
buildurl="docker.io" # 镜像推送的 URL 位置

buildtag_masscan="masscan" # masscan 镜像标签
buildtag_zmap="zmap" # zmap 镜像标签

builddir_masscan="dockerfile-masscan" # masscan Dockerfile 目录
builddir_zmap="dockerfile-zmap" # zmap Dockerfile 目录
builddir_zmap_arm64="dockerfile-zmap-arm64" # zmap Dockerfile 目录

TELEGRAM_BOT_TOKEN="your_telegram_bot_token" # Telegram Bot Token
TELEGRAM_CHAT_ID="your_telegram_chat_id" # Telegram 群组 Chat ID

release_dir="Releases/$buildver" # 发布目录

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

# 检测并安装 buildah
check_and_install_buildah() {
    if ! command -v buildah &> /dev/null; then
        echo "buildah 未安装,正在安装 buildah..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y buildah
        elif command -v yum &> /dev/null; then
            sudo yum install -y buildah
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y buildah
        else
            echo "无法自动安装 buildah,请手动安装后重试."
            exit 1
        fi
        echo "buildah 安装完成."
    else
        echo "buildah 已安装."
    fi
}

# 检测并安装 make
check_and_install_make() {
    if ! command -v make &> /dev/null; then
        echo "make 未安装,正在安装 make..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y make
        elif command -v yum &> /dev/null; then
            sudo yum install -y make
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y make
        else
            echo "无法自动安装 make,请手动安装后重试."
            exit 1
        fi
        echo "make 安装完成."
    else
        echo "make 已安装."
    fi
}

# 检测并安装 GitHub CLI (gh)
check_and_install_gh() {
    if ! command -v gh &> /dev/null; then
        echo "GitHub CLI (gh) 未安装,正在安装 gh..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y gh
        elif command -v yum &> /dev/null; then
            sudo yum install -y gh
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y gh
        else
            echo "无法自动安装 GitHub CLI (gh),请手动安装后重试."
            exit 1
        fi
        echo "GitHub CLI (gh) 安装完成."
    else
        echo "GitHub CLI (gh) 已安装."
    fi
}

# 使用 make 构建本体
build_makefile() {
     
    local ver=$1
    # 检查 Makefile 是否存在


    echo "正在使用 makefile 构建程序本体..."

    # 在指定路径下执行 make 命令
    make  BIN_VER="$ver"
    if [ $? -eq 0 ]; then
        echo "成功构建程序本体,标签为 $ver"
    else
        echo "构建程序本体失败,标签为 $ver"
        exit 1
    fi
}

# 使用 buildah 构建 Docker 镜像
build_buildah_image() {
    local dockerfile=$1
    local tag=$2
    echo "正在使用 $dockerfile 构建镜像,标签为 $tag..."
    buildah bud -f $dockerfile -t $tag
    if [ $? -eq 0 ]; then
        echo "成功构建镜像,标签为 $tag"
    else
        echo "构建镜像失败,标签为 $tag"
        exit 1
    fi
}

# 使用 buildah 推送镜像
push_buildah_image() {
    local tag=$1
    echo "正在推送镜像,标签为 $tag..."
    buildah push $tag
    if [ $? -eq 0 ]; then
        echo "成功推送镜像,标签为 $tag"
        send_telegram_message "✅ 镜像推送成功:$tag"
    else
        echo "推送镜像失败,标签为 $tag"
        send_telegram_message "❌ 镜像推送失败:$tag"
        exit 1
    fi
}

# 发送 Telegram 消息
send_telegram_message() {
    local message=$1
    echo "正在发送 Telegram 消息:$message"
    curl -s -X POST "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/sendMessage" \
        -d "chat_id=$TELEGRAM_CHAT_ID" \
        -d "text=$message" \
        -d "parse_mode=Markdown"
    if [ $? -eq 0 ]; then
        echo "Telegram 消息发送成功."
    else
        echo "Telegram 消息发送失败."
    fi
}

# 动态生成 Release Notes
generate_release_notes() {
    local version=$1
    local notes=""

    # 添加标题
    notes+="# Release $version\n\n"

    # 添加构建信息
    notes+="## 构建信息\n"
    notes+="- 构建用户: $builduser\n"
    notes+="- 镜像名称: $buildname\n"
    notes+="- 镜像版本: $buildver\n"
    notes+="- 镜像仓库: $buildurl\n\n"

    # 添加 Git 提交历史
    if command -v git &> /dev/null; then
        notes+="## 提交历史\n"
        notes+="\`\`\`\n"
        notes+="$(git log --oneline -n 5)\n"
        notes+="\`\`\`\n\n"
    else
        notes+="## 提交历史\n"
        notes+="Git 未安装,无法获取提交历史.\n\n"
    fi

    # 添加构建日志
    notes+="## 构建日志\n"
    notes+="构建成功完成,所有镜像已推送至仓库.\n"

    echo -e "$notes"
}

# 发布到 GitHub Releases
publish_to_github_releases() {
    local version=$1
    local release_dir=$2



    echo "正在发布到 GitHub Releases,版本为 $version..."

    # 生成 Release Notes
    local release_notes
    release_notes=$(generate_release_notes "$version")

    # 创建 GitHub Release
    gh release create "$version" "$release_dir"/* --title "Release $version" --notes "$release_notes"
    if [ $? -eq 0 ]; then
        echo "成功发布到 GitHub Releases,版本为 $version"
        send_telegram_message "✅ GitHub Releases 发布成功:版本 $version"
    else
        echo "发布到 GitHub Releases 失败,版本为 $version"
        send_telegram_message "❌ GitHub Releases 发布失败:版本 $version"
        exit 1
    fi
}

# 主函数
main() {
    # 初始化全局变量
    init

    # 检测并安装 make
    check_and_install_make
    # 检测并安装 buildah
    check_and_install_buildah
    # 检测并安装 GitHub CLI (gh)
    check_and_install_gh

    # 使用 make 构建本体
    build_makefile  "$buildver"

    # 发布到 GitHub Releases
    publish_to_github_releases "$buildver" "$release_dir/$buildver"

    # 构建 masscan 镜像
    masscan_tag="$buildurl/$builduser/$buildname:$buildver-$buildtag_masscan"
    build_buildah_image $builddir_masscan $masscan_tag

    # 构建 zmap 镜像
    zmap_tag="$buildurl/$builduser/$buildname:$buildver-$buildtag_zmap"
    build_buildah_image $builddir_zmap $zmap_tag

    # 构建 zmap_arm64 镜像
    zmap_arm64_tag="$buildurl/$builduser/$buildname:$buildver-$builddir_zmap_arm64"
    build_buildah_image $builddir_zmap_arm64 $zmap_arm64_tag

    # 推送 masscan 镜像
    push_buildah_image $masscan_tag

    # 推送 zmap 镜像
    push_buildah_image $zmap_tag
}

# 执行主函数
main
