APP := minibox
BUILD_DIR := bin
# -s  strip symbol table 删除符号表
# -w  strip DWARF debug info 删除调试信息
LDFLAGS := -ldflags "-s -w -X 'github.com/kyson/minibox/internal/core/version.Tag=$(shell git describe --tags --always)'"

.PHONY: test build lint

# -v verbose mode 显示详细信息
# -cover coverage mode 显示覆盖率
test:
	go test -v -cover ./...

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP) ./cmd/$(APP)

# golangci-lint 聚合型静态分析工具, 几十个 linter 的统一调度器
lint:
	golangci-lint run