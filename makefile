# 定义可执行文件名称
BINARY_NAME := ollama_scanner
BINARY_NAME_MONGODB := ollama_scanner_mongoDB

# 定义发布目录和版本
BIN_DIR := Releases
BIN_VER ?= v2.2.1-r1

# 定义 Go 命令和编译参数
GO := go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION := $(BIN_VER)
BUILD_TIME := $(shell date +%Y%m%d-%H%M%S)
# 定义链接标志，包含版本信息和构建时间
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 支持的操作系统平台和CPU架构
PLATFORMS := darwin linux windows
ARCHS := amd64 arm64

# 平台特定的编译标志
DARWIN_FLAGS := -tags darwin    # macOS 特定标志
LINUX_FLAGS := -tags linux     # Linux 特定标志
WINDOWS_FLAGS := -tags windows # Windows 特定标志
MONGODB_FLAGS := -tags mongodb # MongoDB 支持标志

# 默认构建目标：构建所有平台版本
all: build-all

# 构建所有平台的可执行文件
build-all: build-macos build-linux build-windows

# macOS 平台构建目标
build-macos:
	echo "正在构建 macOS 版本..."
	mkdir -p $(BIN_DIR)/$(BIN_VER)/darwin
	$(MAKE) _build_darwin GOARCH=amd64
	$(MAKE) _build_darwin GOARCH=arm64

# Linux 平台构建目标
build-linux:
	echo "正在构建 Linux 版本..."
	mkdir -p $(BIN_DIR)/$(BIN_VER)/linux
	$(MAKE) _build_linux GOARCH=amd64
	$(MAKE) _build_linux GOARCH=arm64

# Windows 平台构建目标
build-windows:
	echo "正在构建 Windows 版本..."
	mkdir -p $(BIN_DIR)/$(BIN_VER)/windows
	$(MAKE) _build_windows GOARCH=amd64

# macOS 构建实现
_build_darwin:
	echo "正在构建 macOS-$(GOARCH) 标准版..."
	GOOS=darwin $(GO) build $(LDFLAGS) -tags "darwin" \
		-o "$(BIN_DIR)/$(BIN_VER)/darwin/$(BINARY_NAME)-darwin-$(GOARCH)" ./Src/ollama_scanner.go
	echo "正在构建 macOS-$(GOARCH) MongoDB版..."
	GOOS=darwin $(GO) build $(LDFLAGS) -tags "darwin mongodb" \
		-o "$(BIN_DIR)/$(BIN_VER)/darwin/$(BINARY_NAME_MONGODB)-darwin-$(GOARCH)" ./Src/ollama_scanner_mongoDB.go

# Linux 构建实现
_build_linux:
	echo "正在构建 Linux-$(GOARCH) 标准版..."
	GOOS=linux $(GO) build $(LDFLAGS) -tags "linux" \
		-o "$(BIN_DIR)/$(BIN_VER)/linux/$(BINARY_NAME)-linux-$(GOARCH)" ./Src/ollama_scanner.go
	echo "正在构建 Linux-$(GOARCH) MongoDB版..."
	GOOS=linux $(GO) build $(LDFLAGS) -tags "linux mongodb" \
		-o "$(BIN_DIR)/$(BIN_VER)/linux/$(BINARY_NAME_MONGODB)-linux-$(GOARCH)" ./Src/ollama_scanner_mongoDB.go

# Windows 构建实现
_build_windows:
	echo "正在构建 Windows-$(GOARCH) 标准版..."
	GOOS=windows $(GO) build $(LDFLAGS) -tags "windows" \
		-o "$(BIN_DIR)/$(BIN_VER)/windows/$(BINARY_NAME)-windows-$(GOARCH).exe" ./Src/ollama_scanner.go
	echo "正在构建 Windows-$(GOARCH) MongoDB版..."
	GOOS=windows $(GO) build $(LDFLAGS) -tags "windows mongodb" \
		-o "$(BIN_DIR)/$(BIN_VER)/windows/$(BINARY_NAME_MONGODB)-windows-$(GOARCH).exe" ./Src/ollama_scanner_mongoDB.go

# 打包所有构建结果
package: build-all
	@echo "正在打包发布文件..."
	@cd $(BIN_DIR)/$(BIN_VER) && \
	for platform in darwin linux windows; do \
		zip -r "$(BINARY_NAME)-$(VERSION)-$$platform.zip" "$$platform"/*; \
	done
	@echo "打包完成"

# 清理构建文件
clean:
	@echo "正在清理构建文件..."
	@rm -rf $(BIN_DIR)/$(BIN_VER)
	@echo "清理完成"

upmod:
	@echo "正在更新依赖..."
	@bash UpdateGoMod.sh
	@echo "更新完成"

# 定义伪目标，避免与实际文件冲突
.PHONY: all build-all build-macos build-linux build-windows clean package _build_darwin _build_linux _build_windows upmod