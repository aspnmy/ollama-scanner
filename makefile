# 定义可执行文件的名称
BINARY_NAME_ZMAP := ollama-scanner-zmap
BINARY_NAME_MASSCAN := ollama-scanner-masscan
BIN_DIR := Releases
BIN_VER := v2.2

# 定义 Go 命令
GO := go

# 默认目标：编译所有平台的程序
all: build-macos build-linux build-windows

# 编译 macOS 平台的程序
build-macos:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_ZMAP )-darwin-amd64 ./Src/ollama-scanner-zmap.go
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_MASSCAN)-darwin-amd64 ./Src/ollama-scanner-masscan.go
# 编译 Linux 平台的程序
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_ZMAP)-linux-amd64 ./Src/ollama-scanner-zmap.go
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_MASSCAN)-linux-amd64 ./Src/ollama-scanner-masscan.go

# 编译 Windows 平台的程序
build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_MASSCAN)-windows-amd64.exe ./Src/ollama-scanner-masscan.go

# 清理生成的文件
clean:

	rm -f $(BIN_DIR)/$(BIN_VER)/*
# 定义伪目标，避免与同名文件冲突
.PHONY: all build-macos build-linux build-windows clean
