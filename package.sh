#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# 设置默认变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ "$SCRIPT_DIR" != "$PWD" ]; then
    log_error "脚本必须在项目根目录下运行"
    log_error "当前目录: $PWD"
    log_error "脚本目录: $SCRIPT_DIR"
    exit 1
fi

# 基础目录结构和项目名称设置
BASE_DIR="$SCRIPT_DIR"
PROJECT_NAME=$(basename "$BASE_DIR")
BUILD_DIR="$BASE_DIR/build"
VERSION="${VERSION:-1.0.0}"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

# 日志函数
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 确保必要目录存在
init_directories() {
    log_info "正在初始化目录结构..."
    local dirs=(
        "$BASE_DIR/scripts"
        "$BASE_DIR/docs"
        "$BASE_DIR/config"
        "$BUILD_DIR"
        "$BUILD_DIR/tmp"
        "$BUILD_DIR/packages"
        "$BUILD_DIR/checksums"
    )
    
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
            log_info "创建目录: $dir"
        fi
    done

    # 验证目录权限
    if [ ! -w "$BUILD_DIR" ]; then
        log_error "构建目录无写入权限: $BUILD_DIR"
        exit 1
    fi
}

# 创建打包配置
create_package_config() {
    local config_file="$BASE_DIR/.packagerc"
    cat > "$config_file" << EOF
# 打包配置文件
PACKAGE_NAME="${PROJECT_NAME}"
VERSION="${VERSION:-1.0.0}"
BASE_DIR="$BASE_DIR"

# 需要打包的目录和文件
INCLUDE_DIRS=(
    "scripts"
    "docs"
    "config"
    "bin"
)

# 需要打包的具体文件
INCLUDE_FILES=(
    "README.md"
    ".env.example"
)

# 需要排除的模式
EXCLUDE_PATTERNS=(
    "*.log"
    "*.tmp"
    "*.swp"
    ".git"
    "build"
    "node_modules"
)

# 构建信息
BUILD_INFO=(
    "BUILD_USER=${BUILD_USER:-$(whoami)}"
    "BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')"
    "BUILD_VERSION=${VERSION}"
    "BUILD_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
)
EOF
    log_info "已创建默认打包配置文件: $config_file (项目名: $PROJECT_NAME)"
}

# 加载打包配置
load_package_config() {
    local config_file="$BASE_DIR/.packagerc"
    if [ ! -f "$config_file" ]; then
        log_error "未找到打包配置文件: $config_file"
        create_package_config
        exit 1
    fi
    source "$config_file"
    
    # 验证基础目录
    if [ "$BASE_DIR" != "$SCRIPT_DIR" ]; then
        log_error "配置文件中的基础目录与当前不匹配"
        log_error "配置: $BASE_DIR"
        log_error "当前: $SCRIPT_DIR"
        exit 1
    fi
}

# 准备打包目录
prepare_package() {
    local package_dir="$BUILD_DIR/packages/${PACKAGE_NAME}_${VERSION}"
    rm -rf "$package_dir"
    mkdir -p "$package_dir"
    
    # 复制目录
    for dir in "${INCLUDE_DIRS[@]}"; do
        local src_dir="$BASE_DIR/$dir"
        if [ -d "$src_dir" ]; then
            log_info "复制目录: $dir"
            cp -r "$src_dir" "$package_dir/"
        else
            log_info "跳过不存在的目录: $dir"
        fi
    done
    
    # 复制文件
    for file in "${INCLUDE_FILES[@]}"; do
        if [ -f "$PROJECT_ROOT/$file" ]; then
            log_info "复制文件: $file"
            cp "$PROJECT_ROOT/$file" "$package_dir/"
        fi
    done
    
    # 创建版本信息
    create_version_file "$package_dir/version.txt"
    
    echo "$package_dir"
}

# 创建版本信息文件
create_version_file() {
    local version_file="$1"
    {
        echo "# 构建信息"
        for info in "${BUILD_INFO[@]}"; do
            echo "$info"
        done
        echo "PACKAGE_NAME=$PACKAGE_NAME"
        echo "PACKAGE_VERSION=$VERSION"
        echo "BUILD_TIMESTAMP=$TIMESTAMP"
    } > "$version_file"
}

# 打包函数
do_package() {
    # 准备打包内容
    local tmp_dir="$BUILD_DIR/tmp"
    local package_name="${PACKAGE_NAME}_${VERSION}"
    local package_dir="$tmp_dir/$package_name"
    
    # 清理并创建临时目录
    rm -rf "$tmp_dir"
    mkdir -p "$package_dir"
    
    # 复制目录
    for dir in "${INCLUDE_DIRS[@]}"; do
        local src_dir="$BASE_DIR/$dir"
        if [ -d "$src_dir" ]; then
            log_info "复制目录: $dir"
            cp -r "$src_dir" "$package_dir/"
        else
            log_info "跳过不存在的目录: $dir"
        fi
    done
    
    # 复制文件
    for file in "${INCLUDE_FILES[@]}"; do
        local src_file="$BASE_DIR/$file"
        if [ -f "$src_file" ]; then
            log_info "复制文件: $file"
            cp "$src_file" "$package_dir/"
        fi
    done
    
    # 创建版本信息
    create_version_file "$package_dir/version.txt"
    
    # 切换到临时目录并创建压缩包
    cd "$tmp_dir" || exit 1
    
    # 创建最终输出目录
    mkdir -p "$BUILD_DIR/packages"
    
    # 创建压缩包
    log_info "创建压缩包..."
    tar czf "$BUILD_DIR/packages/$package_name.tar.gz" "$package_name"
    
    # 生成校验和
    cd "$BUILD_DIR/packages" || exit 1
    sha256sum "$package_name.tar.gz" > "$package_name.tar.gz.sha256"
    
    # 清理临时目录
    rm -rf "$tmp_dir"
    
    log_info "打包完成: packages/$package_name.tar.gz"
    log_info "校验文件: packages/$package_name.tar.gz.sha256"
    
    # 返回到原始目录
    cd "$BASE_DIR" || exit 1
}

# 显示帮助
show_help() {
    echo "用法: $0 [选项]"
    echo "选项:"
    echo "  init     - 创建默认打包配置"
    echo "  pack     - 执行打包"
    echo "  clean    - 清理构建目录"
    echo "  help     - 显示此帮助"
}

# 主函数
main() {
    # 验证脚本运行位置
    if [ "$SCRIPT_DIR" != "$PWD" ]; then
        log_error "请在项目根目录下运行此脚本"
        exit 1
    fi
    
    # 确保目录结构完整
    init_directories
    
    case "${1:-pack}" in
        "init")
            create_package_config
            ;;
        "pack")
            load_package_config
            do_package
            ;;
        "clean")
            rm -rf "$BUILD_DIR"/*
            init_directories
            log_info "清理并重建目录完成"
            ;;
        "help")
            show_help
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
