#!/bin/bash

# 更新共享库路径为相对路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/lib"
mkdir -p "$LIB_DIR"
#echo $LIB_DIR
# 设置库路径并编译so链接
LD_LIBRARY_PATH="$LIB_DIR:$LD_LIBRARY_PATH"
export LD_LIBRARY_PATH

# 统一库名称和路径
LIB_PATH="$LIB_DIR/aspnmy_crypto_utils.so"

# 检查共享库是否存在
if [ ! -f "$LIB_PATH" ]; then
    echo "错误: 未找到共享库 $LIB_PATH"
    exit 1
fi

# 加载共享库函数
generate_key() {
    "$LIB_PATH" generate_key
}

encrypt_text() {
    "$LIB_PATH" encrypt_text "$1" "$2"
}

decrypt_text() {
    "$LIB_PATH" decrypt_text "$1" "$2"
}

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 修改生成密钥函数
get_or_generate_key() {
    if [ -f ~/.crypto_key ]; then
        cat ~/.crypto_key
    else
        generate_key | tee ~/.crypto_key
    fi
}

# 修改加密函数
encrypt() {
    local plain_text="$1"
    local key="${CRYPTO_KEY:-$(get_or_generate_key)}"
    
    if [ -z "$plain_text" ]; then
        echo "错误: 未提供要加密的文本" >&2
        return 1
    fi

    encrypt_text "$plain_text" "$key"
}

# 修改解密函数
decrypt() {
    local encrypted_text="$1"
    local key="${CRYPTO_KEY:-$(get_or_generate_key)}"
    
    if [ -z "$encrypted_text" ]; then
        echo "错误: 未提供要解密的文本" >&2
        return 1
    fi

    decrypt_text "$encrypted_text" "$key"
}

# 主函数
main() {
    case "$1" in
        "encrypt")
            if [ -z "$2" ]; then
                read -p "输入要加密的文本: " text
            else
                text="$2"
            fi
            encrypt "$text"
            ;;
        "decrypt")
            if [ -z "$2" ]; then
                read -p "输入要解密的文本: " text
            else
                text="$2"
            fi
            decrypt "$text"
            ;;
        "genkey")
            echo "生成新密钥..."
            rm -f ~/.crypto_key
            key=$(get_or_generate_key)
            echo -e "${GREEN}密钥已更新: $key${NC}"
            ;;
        *)
            echo "用法: $0 {encrypt|decrypt|genkey} [text]"
            echo "  encrypt [text] - 加密文本"
            echo "  decrypt [text] - 解密文本"
            echo "  genkey        - 生成新密钥"
            exit 1
            ;;
    esac
}

# 仅在直接执行时运行主函数
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
