# 定义可执行文件的名称
BINARY_NAME_ZMAP := ollama_scanner-zmap
BINARY_NAME_MASSCAN := ollama_scanner-masscan
BIN_DIR := Releases
BIN_VER ?= v2.2  # 默认值为 v2.2，可通过命令行覆盖

# 定义 Go 命令
GO := go

# 定义支持的平台和架构
PLATFORMS := darwin linux windows
ARCHS := amd64 arm64

# 默认目标：编译所有平台的程序
all: build-all

# 编译所有平台的程序
build-all: build-macos build-linux build-windows

# 编译 macOS 平台的程序
build-macos:
	$(call build,darwin,amd64)
	$(call build,darwin,arm64)

# 编译 Linux 平台的程序
build-linux:
	$(call build,linux,amd64)
	$(call build,linux,arm64)

# 编译 Windows 平台的程序
build-windows:
	$(call build,windows,amd64)

# 定义编译函数
define build
	@echo "Building for $(1)-$(2)..."
	@mkdir -p $(BIN_DIR)/$(BIN_VER)
	@if $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_ZMAP)-$(1)-$(2) ./Src/ollama_scanner-zmap.go; then \
		echo "Success: $(BINARY_NAME_ZMAP)-$(1)-$(2)"; \
	else \
		echo "Failed: $(BINARY_NAME_ZMAP)-$(1)-$(2)"; \
		exit 1; \
	fi
	@if $(GO) build -o $(BIN_DIR)/$(BIN_VER)/$(BINARY_NAME_MASSCAN)-$(1)-$(2) ./Src/ollama_scanner-masscan.go; then \
		echo "Success: $(BINARY_NAME_MASSCAN)-$(1)-$(2)"; \
	else \
		echo "Failed: $(BINARY_NAME_MASSCAN)-$(1)-$(2)"; \
		exit 1; \
	fi
endef

# 清理生成的文件
clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)/$(BIN_VER)
	@echo "Cleanup complete."

# 定义伪目标，避免与同名文件冲突
.PHONY: all build-all build-macos build-linux build-windows clean
