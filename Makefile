.PHONY: build clean version install help

# 版本信息
VERSION := v1.1.0
BUILD_TIME := $(shell date +%Y-%m-%d\ %H:%M:%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# 编译标志
LDFLAGS := -ldflags "-s -w \
	-X 'dockship/cmd.Version=$(VERSION)' \
	-X 'dockship/cmd.BuildTime=$(BUILD_TIME)' \
	-X 'dockship/cmd.GitCommit=$(GIT_COMMIT)'"

# 目标二进制文件
BINARY := dockship

help: ## 显示帮助信息
	@echo "Dockship Makefile 命令:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 编译项目
	@echo "🔨 正在编译 $(BINARY)..."
	@echo "   Version:    $(VERSION)"
	@echo "   Git Commit: $(GIT_COMMIT)"
	@echo "   Build Time: $(BUILD_TIME)"
	@go mod tidy
	@go build $(LDFLAGS) -o $(BINARY)
	@echo "✅ 编译完成: ./$(BINARY)"

clean: ## 清理编译产物
	@echo "🧹 清理编译产物..."
	@rm -f $(BINARY)
	@echo "✅ 清理完成"

install: build ## 安装到 $GOPATH/bin
	@echo "📦 安装到 $$GOPATH/bin..."
	@cp $(BINARY) $$GOPATH/bin/
	@echo "✅ 安装完成"

version: ## 显示版本信息
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"

run: build ## 编译并运行
	@./$(BINARY)

# 默认目标
.DEFAULT_GOAL := build
