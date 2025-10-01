.PHONY: help build run test clean lint fmt tidy install-tools

# 默认目标
.DEFAULT_GOAL := help

# 项目配置
APP_NAME := siriusx-api
BUILD_DIR := bin
CMD_DIR := cmd/server
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')

# Go 命令
GOCMD := go
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# 构建标志
LDFLAGS := -ldflags="-s -w"
BUILD_FLAGS := -v

## help: 显示帮助信息
help:
	@echo "可用的 Make 命令:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""

## build: 编译项目
build:
	@echo "🔨 编译项目..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "✅ 编译完成: $(BUILD_DIR)/$(APP_NAME)"

## run: 启动开发服务器
run:
	@echo "🚀 启动开发服务器..."
	$(GORUN) ./$(CMD_DIR)

## test: 运行测试
test:
	@echo "🧪 运行测试..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "✅ 测试完成"

## test-coverage: 运行测试并生成覆盖率报告
test-coverage: test
	@echo "📊 生成覆盖率报告..."
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

## clean: 清理构建产物
clean:
	@echo "🧹 清理构建产物..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	$(GOCLEAN)
	@echo "✅ 清理完成"

## lint: 代码检查 (需要 golangci-lint)
lint:
	@echo "🔍 运行代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
		echo "✅ 代码检查完成"; \
	else \
		echo "⚠️  golangci-lint 未安装，跳过代码检查"; \
		echo "💡 安装方法: make install-tools"; \
	fi

## fmt: 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	$(GOFMT) -s -w $(GO_FILES)
	@echo "✅ 代码格式化完成"

## tidy: 整理依赖
tidy:
	@echo "📦 整理依赖..."
	$(GOMOD) tidy
	@echo "✅ 依赖整理完成"

## install-tools: 安装开发工具
install-tools:
	@echo "🔧 安装开发工具..."
	@echo "安装 golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin; \
	else \
		echo "golangci-lint 已安装"; \
	fi
	@echo "✅ 工具安装完成"

## docker-build: 构建 Docker 镜像
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	docker build -t $(APP_NAME):latest .
	@echo "✅ Docker 镜像构建完成"

## docker-run: 运行 Docker 容器
docker-run:
	@echo "🐳 启动 Docker 容器..."
	docker run -p 8080:8080 --name $(APP_NAME) $(APP_NAME):latest

## docker-stop: 停止 Docker 容器
docker-stop:
	@echo "🛑 停止 Docker 容器..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

## all: 格式化、测试、构建
all: fmt tidy test build
	@echo "✅ 所有任务完成"
