# IM系统 Makefile

.PHONY: help build clean test benchmark docker-build docker-run docker-stop start stop status

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BINARY_NAME=im-server
CLIENT_NAME=im-client
BENCHMARK_NAME=benchmark
BUILD_DIR=bin
DOCKER_IMAGE=im-server

# 帮助信息
help: ## 显示帮助信息
	@echo "IM系统构建和管理工具"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# 构建目标
build: ## 构建应用
	@echo "构建IM系统..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	go build -o $(BUILD_DIR)/$(CLIENT_NAME) ./cmd/client
	@echo "构建完成"

# 清理构建文件
clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "清理完成"

# 运行测试
test: ## 运行测试
	@echo "运行测试..."
	go test -v ./...
	@echo "测试完成"

# 运行性能测试
benchmark: build ## 运行性能测试
	@echo "构建性能测试工具..."
	go build -o $(BUILD_DIR)/$(BENCHMARK_NAME) ./scripts/benchmark
	@echo "运行性能测试..."
	./$(BUILD_DIR)/$(BENCHMARK_NAME)

# 格式化代码
fmt: ## 格式化代码
	@echo "格式化代码..."
	go fmt ./...
	@echo "格式化完成"

# 代码检查
lint: ## 代码检查
	@echo "代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint未安装，跳过代码检查"; \
	fi

# 下载依赖
deps: ## 下载依赖
	@echo "下载依赖..."
	go mod download
	go mod tidy
	@echo "依赖下载完成"

# Docker相关命令
docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker镜像构建完成"

docker-run: ## 运行Docker容器
	@echo "运行Docker容器..."
	docker-compose up -d
	@echo "Docker容器启动完成"

docker-stop: ## 停止Docker容器
	@echo "停止Docker容器..."
	docker-compose down
	@echo "Docker容器停止完成"

# 服务管理
start: ## 启动所有服务
	@echo "启动所有服务..."
	@bash scripts/start.sh start

stop: ## 停止所有服务
	@echo "停止所有服务..."
	@bash scripts/start.sh stop

restart: ## 重启所有服务
	@echo "重启所有服务..."
	@bash scripts/start.sh restart

status: ## 查看服务状态
	@bash scripts/start.sh status

# 开发相关
dev: deps build ## 开发环境准备
	@echo "开发环境准备完成"
	@echo "运行以下命令启动服务:"
	@echo "  make start"

# 安装
install: build ## 安装到系统
	@echo "安装IM系统..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "安装完成"

# 卸载
uninstall: ## 从系统卸载
	@echo "卸载IM系统..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "卸载完成"

# 生成文档
docs: ## 生成文档
	@echo "生成文档..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "启动godoc服务器: http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc未安装，跳过文档生成"; \
	fi

# 代码覆盖率
coverage: ## 生成代码覆盖率报告
	@echo "生成代码覆盖率报告..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 性能分析
profile: build ## 性能分析
	@echo "启动性能分析..."
	@echo "访问 http://localhost:8080/debug/pprof/ 查看性能数据"

# 安全扫描
security: ## 安全扫描
	@echo "安全扫描..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec未安装，跳过安全扫描"; \
	fi

# 发布
release: clean deps test build ## 发布版本
	@echo "发布版本..."
	@echo "构建文件位于 $(BUILD_DIR)/"
	@echo "发布完成"

# 快速启动（仅启动依赖服务）
quick-start: ## 快速启动（仅依赖服务）
	@echo "快速启动依赖服务..."
	docker-compose up -d mysql redis zookeeper kafka
	@echo "依赖服务启动完成"

# 快速停止（仅停止依赖服务）
quick-stop: ## 快速停止（仅依赖服务）
	@echo "快速停止依赖服务..."
	docker-compose down
	@echo "依赖服务停止完成"

# 查看日志
logs: ## 查看服务日志
	@echo "查看服务日志..."
	docker-compose logs -f

# 数据库迁移
migrate: ## 数据库迁移
	@echo "数据库迁移..."
	@echo "请确保数据库服务已启动"
	@echo "迁移脚本需要手动执行"

# 备份数据
backup: ## 备份数据
	@echo "备份数据..."
	@mkdir -p backups
	docker-compose exec mysql mysqldump -u im_user -pim_password im_db > backups/im_db_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "数据备份完成"

# 恢复数据
restore: ## 恢复数据
	@echo "恢复数据..."
	@echo "请指定备份文件: make restore FILE=backups/im_db_20240101_120000.sql"
	@if [ -n "$(FILE)" ]; then \
		docker-compose exec -T mysql mysql -u im_user -pim_password im_db < $(FILE); \
		echo "数据恢复完成"; \
	else \
		echo "请指定备份文件"; \
	fi 