# Ollama 模型下载与运行指南

## 基础命令

### 1. 下载模型
```bash
# 下载基础模型
ollama pull llama2
ollama pull deepseek-coder
ollama pull dolphin-phi

# 下载特定版本
ollama pull llama2:7b
ollama pull llama2:13b
ollama pull llama2:70b
```

### 2. 运行模型
```bash
# 基础运行
ollama run llama2

# 指定系统提示词运行
ollama run llama2 "你是一个AI助手"

# 使用流式输出
ollama run llama2 --stream
```

### 3. 模型管理
```bash
# 列出所有模型
ollama list

# 删除模型
ollama rm llama2
ollama rm llama2:13b

# 获取模型信息
ollama show llama2
```

## 性能优化

### 1. GPU 设置
```bash
# 使用 GPU 运行
CUDA_VISIBLE_DEVICES=0 ollama run llama2

# 多 GPU 运行
CUDA_VISIBLE_DEVICES=0,1 ollama run llama2
```

### 2. 内存优化
```bash
# 设置最大内存使用量
OLLAMA_HOST=0.0.0.0 OLLAMA_MODELS=/path/to/models ollama serve

# 使用量化模型减少内存占用
ollama pull llama2:7b-q4_0
ollama pull llama2:7b-q4_K_M
```

## 常用模型推荐

### 1. 通用对话
- llama2:7b - 基础对话模型
- deepseek-chat - 中英双语对话
- vicuna-chat - 优化的对话体验

### 2. 代码开发
- deepseek-coder:6.7b - 代码开发助手
- wizard-coder - 代码理解与生成
- phind-codellama - 专业代码解释

### 3. 中文优化
- chinese-llama2 - 中文优化版本
- chatglm3 - 专门针对中文优化
- baichuan - 高性能中文模型

## 常见问题解决

### 1. 下载问题
```bash
# 下载失败重试
ollama pull llama2 --retry 3

# 使用代理下载
export https_proxy=http://proxy.example.com:port
ollama pull llama2
```

### 2. 运行错误
```bash
# 重置服务
systemctl restart ollama

# 清理缓存
rm -rf ~/.ollama/models/*
ollama pull llama2
```

### 3. 性能问题
```bash
# 检查GPU使用情况
nvidia-smi -l 1

# 监控内存使用
htop
```

## 最佳实践

1. 模型选择
   - 7B 模型适合普通PC
   - 13B 模型需要较好显卡
   - 70B 模型建议企业服务器

2. 资源管理
   - 定期清理未使用模型
   - 使用量化版本节省资源
   - 避免同时运行多个大模型

3. 安全建议
   - 设置访问控制
   - 定期更新版本
   - 监控系统资源

## 参考资源

- [Ollama 官方文档](https://ollama.ai/docs)
- [模型下载地址](https://ollama.ai/library)
- [GitHub 仓库](https://github.com/ollama/ollama)
