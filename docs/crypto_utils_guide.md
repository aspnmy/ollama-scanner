# aspnmy_crypto_utils.so 使用指南

## 1. 环境准备

### 1.1 复制共享库文件
```bash
# 复制 .so 文件到系统库目录
sudo cp aspnmy_crypto_utils.so /usr/local/lib/
sudo ldconfig
```

### 1.2 验证安装
```bash
# 检查库是否正确安装
ldconfig -p | grep aspnmy_crypto_utils
```

## 2. 基础使用

### 2.1 生成密钥
```bash
./crypto_utils.sh genkey
# 生成的密钥将保存在 ~/.crypto_key
```

### 2.2 加密解密
```bash
# 加密文本
./crypto_utils.sh encrypt "需要加密的文本"

# 解密文本
./crypto_utils.sh decrypt "U2FsdGVkX1+..."
```

## 3. 在项目中使用

### 3.1 环境变量集成
```bash
source ./crypto_utils.sh

# 加密敏感信息
TOKEN="abc123"
ENCRYPTED_TOKEN=$(encrypt "$TOKEN")

# 使用加密后的值
echo "TELEGRAM_BOT_TOKEN=$ENCRYPTED_TOKEN" >> .env
```

### 3.2 动态解密
```bash
# 在需要使用原文的地方解密
DECRYPTED_TOKEN=$(decrypt "${aspnmy[TELEGRAM_BOT_TOKEN]}")
```

## 4. 故障排除

### 4.1 常见问题
```bash
# 库未找到时
export LD_LIBRARY_PATH="/usr/local/lib:$LD_LIBRARY_PATH"

# 权限问题
sudo chmod 755 /usr/local/lib/aspnmy_crypto_utils.so
```

### 4.2 注意事项
- 加密数据以 "U2FsdGVkX1+" 开头
- 密钥文件位置: ~/.crypto_key
- 可通过 CRYPTO_KEY 环境变量覆盖默认密钥

## 5. 安全建议
1. 定期更换加密密钥
2. 不要在日志中打印解密后的敏感信息
3. 确保 .env 文件权限正确设置
4. 避免密钥进入版本控制系统
