# aspnmy_crypto 使用手册

## 1. 简介

aspnmy_crypto 是一个基于 OpenSSL 的加密工具，用于敏感数据的加密和解密。支持命令行操作和程序集成。

## 2. 功能特性

- AES-256-CBC 加密算法
- Base64 编码输出
- 自动密钥管理
- 支持环境变量配置
- 交互式和命令行模式
- 标准输入输出支持

## 3. 安装和配置

### 3.1 环境要求
- Linux/Unix 系统
- OpenSSL 1.1.1 或更高版本

### 3.2 运行环境设置
```bash
# 给予执行权限
chmod +x aspnmy_crypto

# 设置环境变量（可选）
export CRYPTO_KEY="your_custom_key"
```

## 4. 基本用法

### 4.1 生成密钥
```bash
./aspnmy_crypto genkey
# 输出: 密钥已更新: f3a1b8c9...
```

### 4.2 加密数据
```bash
# 命令行方式
./aspnmy_crypto encrypt "要加密的文本"

# 交互式输入
./aspnmy_crypto encrypt
输入要加密的文本: _

# 通过管道输入
echo "要加密的文本" | ./aspnmy_crypto encrypt
```

### 4.3 解密数据
```bash
# 命令行方式
./aspnmy_crypto decrypt "U2FsdGVkX1..."

# 交互式输入
./aspnmy_crypto decrypt
输入要解密的文本: _
```

## 5. 进阶用法

### 5.1 在脚本中使用
```bash
#!/bin/bash

# 加密数据
encrypted=$(./aspnmy_crypto encrypt "sensitive data")
echo "加密结果: $encrypted"

# 解密数据
decrypted=$(./aspnmy_crypto decrypt "$encrypted")
echo "解密结果: $decrypted"
```

### 5.2 批量处理
```bash
# 加密文件内容
while IFS= read -r line; do
    ./aspnmy_crypto encrypt "$line" >> encrypted.txt
done < input.txt

# 解密文件内容
while IFS= read -r line; do
    ./aspnmy_crypto decrypt "$line" >> decrypted.txt
done < encrypted.txt
```

### 5.3 管道操作
```bash
# 加密后直接解密
echo "test" | ./aspnmy_crypto encrypt | ./aspnmy_crypto decrypt
```

## 6. 配置说明

### 6.1 密钥管理
- 默认密钥位置: `~/.crypto_key`
- 环境变量覆盖: `CRYPTO_KEY`
- 密钥文件权限: `600`

### 6.2 环境变量
```bash
# 设置自定义密钥
export CRYPTO_KEY="your_custom_key"

# 使用自定义密钥
./aspnmy_crypto encrypt "test"
```

## 7. 安全建议

1. 定期更换加密密钥
2. 避免明文传输密钥
3. 及时清理临时文件
4. 注意文件权限设置
5. 不要在日志中记录敏感信息

## 8. 常见问题

### 8.1 密钥未找到
```bash
错误: 未找到密钥，请先运行 genkey
# 解决方案: 执行 ./aspnmy_crypto genkey
```

### 8.2 解密失败
```bash
# 常见原因:
1. 密文格式错误
2. 使用了错误的密钥
3. 数据在传输过程中被破坏
```

### 8.3 性能优化
- 批量处理时使用管道
- 避免频繁生成新密钥
- 合理使用缓存机制

## 9. 最佳实践

1. 在配置文件中使用加密值
```bash
# .env 文件示例
DB_PASSWORD=$(./aspnmy_crypto encrypt "real_password")
API_KEY=$(./aspnmy_crypto encrypt "secret_key")
```

2. 自动化密钥备份
```bash
# 备份脚本示例
backup_key() {
    cp ~/.crypto_key ~/.crypto_key.backup
    chmod 600 ~/.crypto_key.backup
}
```

3. 集成到服务中
```bash
# 系统服务启动前解密配置
export DB_PASSWORD=$(./aspnmy_crypto decrypt "$encrypted_password")
```

## 10. 技术支持

- 问题反馈：创建 Issue
- 代码贡献：提交 Pull Request
- 安全问题：直接联系维护者

## 11. 版本历史

- v1.0.0: 首次发布
  - 基本加密解密功能
  - 密钥管理系统
  - 命令行界面
