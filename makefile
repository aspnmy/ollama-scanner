# 定义可执行文件的名称
BINARY_NAME := ollama-scanner

# 定义 Go 命令
GO := go

# 默认目标：编译所有平台的程序
all: build-macos build-linux build-windows

# 编译 macOS 平台的程序
build-macos:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-darwin-amd64 ./ollama-scanner.go

# 编译 Linux 平台的程序
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux-amd64 ./ollama-scanner.go

# 编译 Windows 平台的程序
build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-windows-amd64.exe ./ollama-scanner.go

# 清理生成的文件
clean:
	rm -f $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-windows-amd64.exe

# 定义伪目标，避免与同名文件冲突
.PHONY: all build-macos build-linux build-windows clean
