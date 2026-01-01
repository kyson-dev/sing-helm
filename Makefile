# --- 项目元数据 ---
APP := minibox
BUILD_DIR := bin
# 获取 Git commit hash 和 tag，用于注入版本号
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)

# --- 编译核心配置 (关键) ---
# 1. CGO_ENABLED=0: 强制关闭 CGO，确保生成纯静态二进制文件。
#    这样编译出来的程序，扔到任何 Linux 发行版（Alpine, CentOS, Ubuntu）都能跑，不需要 glibc。
ENV := CGO_ENABLED=0

# 2. TAGS: 激活 sing-box 的隐藏功能
TAGS := with_quic,with_wireguard,with_utls,with_real_ip,with_clash_api,with_gvisor

# 3. LDFLAGS: 链接器参数
# -s: 省略符号表 (Symbol Table) -> 减小体积
# -w: 省略 DWARF 调试信息 -> 减小体积
# -X: 注入变量值 (把 Makefile 里的 VERSION 塞进 Go 代码里)
LDFLAGS := -s -w \
	-X 'github.com/kyson/minibox/internal/core/version.Tag=$(VERSION)' \
	-X 'github.com/kyson/minibox/internal/core/version.Commit=$(COMMIT)' \
	-X 'github.com/kyson/minibox/internal/core/version.Date=$(DATE)'

# 4. TRIMPATH: 移除二进制文件中的绝对路径信息 (保护隐私，且让构建可复现)
FLAGS := -tags "$(TAGS)" -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all test test-verbose test-short test-coverage build lint clean

all: lint test build

# 简洁模式测试（推荐日常使用）
test:
	@echo "Running tests..."
	@go test  ./... -cover

# 详细模式测试（查看所有输出）
test-verbose:
	go test -v -cover ./...

# 快速测试（跳过慢速测试）
test-short:
	go test -short ./...

# CI 测试（不使用 -cover 避免 covdata 问题）
test-ci:
	@echo "Running tests for CI..."
	@go test -v ./...

# 生成覆盖率报告
test-coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 默认构建 (当前系统架构)
build:
	@echo "Building $(APP) for local os/arch..."
	$(ENV) go build $(FLAGS) -o $(BUILD_DIR)/$(APP) ./cmd/$(APP)
	@echo "Build success! Size: $$(du -h $(BUILD_DIR)/$(APP) | cut -f1)"

# 开发模式构建 (同 build，环境通过 --home 参数指定)
build-dev:
	@echo "Building $(APP) for development..."
	@echo "Binary:   $(BUILD_DIR)/$(APP)"
	$(ENV) go build $(FLAGS) -o $(BUILD_DIR)/$(APP) ./cmd/$(APP)
	@echo ""
	@echo "Build success! Size: $$(du -h $(BUILD_DIR)/$(APP) | cut -f1)"
	@echo ""
	@echo "Usage:"
	@echo "  ./bin/minibox run                    # Use default home (~/.minibox)"
	@echo "  ./bin/minibox run --home ./bin/dev   # Use custom directory"
	@echo ""
	@echo "Environment is auto-detected or can be specified with --home flag"

# --- 交叉编译 (Cross Compilation) ---
# Go 的一大杀器：一条命令打出 Windows, Linux, macOS 包
build-all:
# 	@echo "Building for Linux (amd64)..."
# 	GOOS=linux GOARCH=amd64 $(ENV) go build $(FLAGS) -o $(BUILD_DIR)/$(APP)-linux-amd64 ./cmd/$(APP)
	
# 	@echo "Building for Windows (amd64)..."
# 	GOOS=windows GOARCH=amd64 $(ENV) go build $(FLAGS) -o $(BUILD_DIR)/$(APP)-windows-amd64.exe ./cmd/$(APP)
	
	@echo "Building for macOS (arm64/M1)..."
	GOOS=darwin GOARCH=arm64 $(ENV) go build $(FLAGS) -o $(BUILD_DIR)/$(APP)-darwin-arm64 ./cmd/$(APP)
	
	@echo "All builds finished in $(BUILD_DIR)/"


# golangci-lint 聚合型静态分析工具, 几十个 linter 的统一调度器
lint:
	golangci-lint run

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html