#!/bin/bash

# 定义变量
builduser="aspnmy" # 镜像仓库用户名
buildname="ollama-scanner" # 镜像名称
buildver="v2.2" # 镜像版本
buildurl="docker.io" # 镜像推送的 URL 位置（默认为 Docker Hub）

buildtag_masscan="masscan" # masscan 镜像标签
buildtag_zmap="zmap" # zmap 镜像标签

builddir_masscan="dockerfile-masscan" # masscan Dockerfile 目录
builddir_zmap="dockerfile-zmap" # zmap Dockerfile 目录

# 检测并安装 buildah
check_and_install_buildah() {
    if ! command -v buildah &> /dev/null; then
        echo "buildah 未安装，正在安装 buildah..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y buildah
        elif command -v yum &> /dev/null; then
            sudo yum install -y buildah
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y buildah
        else
            echo "无法自动安装 buildah，请手动安装后重试。"
            exit 1
        fi
        echo "buildah 安装完成。"
    else
        echo "buildah 已安装。"
    fi
}

# 使用 buildah 构建 Docker 镜像
build_buildah_image() {
    local dockerfile=$1
    local tag=$2
    echo "正在使用 $dockerfile 构建镜像，标签为 $tag..."
    buildah bud -f $dockerfile -t $tag
    if [ $? -eq 0 ]; then
        echo "成功构建镜像，标签为 $tag"
    else
        echo "构建镜像失败，标签为 $tag"
        exit 1
    fi
}

# 使用 buildah 推送镜像
push_buildah_image() {
    local tag=$1
    echo "正在推送镜像，标签为 $tag..."
    buildah push $tag
    if [ $? -eq 0 ]; then
        echo "成功推送镜像，标签为 $tag"
    else
        echo "推送镜像失败，标签为 $tag"
        exit 1
    fi
}

# 主函数
main() {
    # 检测并安装 buildah
    check_and_install_buildah

    # 构建 masscan 镜像
    masscan_tag="$buildurl/$builduser/$buildname:$buildver-$buildtag_masscan"
    build_buildah_image $builddir_masscan $masscan_tag

    # 构建 zmap 镜像
    zmap_tag="$buildurl/$builduser/$buildname:$buildver-$buildtag_zmap"
    build_buildah_image $builddir_zmap $zmap_tag

    # 推送 masscan 镜像
    push_buildah_image $masscan_tag

    # 推送 zmap 镜像
    push_buildah_image $zmap_tag
}

# 执行主函数
main
