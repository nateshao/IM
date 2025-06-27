#!/bin/bash

# IM系统启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    print_info "检查依赖..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose未安装，请先安装Docker Compose"
        exit 1
    fi
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        print_error "Go未安装，请先安装Go 1.21+"
        exit 1
    fi
    
    print_success "依赖检查完成"
}

# 启动依赖服务
start_dependencies() {
    print_info "启动依赖服务..."
    
    # 启动MySQL, Redis, Kafka等
    docker-compose up -d mysql redis zookeeper kafka
    
    # 等待服务启动
    print_info "等待服务启动..."
    sleep 30
    
    # 检查服务状态
    if ! docker-compose ps | grep -q "Up"; then
        print_error "依赖服务启动失败"
        docker-compose logs
        exit 1
    fi
    
    print_success "依赖服务启动完成"
}

# 构建应用
build_app() {
    print_info "构建IM应用..."
    
    # 下载依赖
    go mod download
    
    # 构建应用
    go build -o bin/im-server ./cmd/server
    go build -o bin/im-client ./cmd/client
    
    print_success "应用构建完成"
}

# 启动应用
start_app() {
    print_info "启动IM服务器..."
    
    # 检查配置文件
    if [ ! -f "config.yaml" ]; then
        print_error "配置文件config.yaml不存在"
        exit 1
    fi
    
    # 启动服务器
    ./bin/im-server &
    SERVER_PID=$!
    
    # 等待服务器启动
    sleep 5
    
    # 检查服务器是否启动成功
    if ! curl -f http://localhost:8080/health &> /dev/null; then
        print_error "服务器启动失败"
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
    
    print_success "IM服务器启动完成，PID: $SERVER_PID"
    echo $SERVER_PID > .server.pid
}

# 运行测试
run_tests() {
    print_info "运行测试..."
    
    # 运行单元测试
    go test ./...
    
    # 运行集成测试
    if [ -f "scripts/test/integration_test.sh" ]; then
        bash scripts/test/integration_test.sh
    fi
    
    print_success "测试完成"
}

# 运行性能测试
run_benchmark() {
    print_info "运行性能测试..."
    
    # 构建性能测试工具
    go build -o bin/benchmark ./scripts/benchmark
    
    # 运行性能测试
    ./bin/benchmark
    
    print_success "性能测试完成"
}

# 显示状态
show_status() {
    print_info "系统状态:"
    
    echo "=== 服务状态 ==="
    docker-compose ps
    
    echo "=== 服务器状态 ==="
    if [ -f ".server.pid" ]; then
        SERVER_PID=$(cat .server.pid)
        if ps -p $SERVER_PID > /dev/null; then
            print_success "IM服务器运行中 (PID: $SERVER_PID)"
        else
            print_error "IM服务器未运行"
        fi
    else
        print_error "IM服务器未运行"
    fi
    
    echo "=== 端口状态 ==="
    echo "IM服务器: http://localhost:8080"
    echo "健康检查: http://localhost:8080/health"
    echo "监控指标: http://localhost:8080/metrics"
    echo "Kafka UI: http://localhost:8081"
    echo "Grafana: http://localhost:3000 (admin/admin)"
    echo "Prometheus: http://localhost:9090"
}

# 停止服务
stop_services() {
    print_info "停止服务..."
    
    # 停止IM服务器
    if [ -f ".server.pid" ]; then
        SERVER_PID=$(cat .server.pid)
        if ps -p $SERVER_PID > /dev/null; then
            kill $SERVER_PID
            print_success "IM服务器已停止"
        fi
        rm -f .server.pid
    fi
    
    # 停止Docker服务
    docker-compose down
    
    print_success "所有服务已停止"
}

# 清理
cleanup() {
    print_info "清理资源..."
    
    # 停止服务
    stop_services
    
    # 清理Docker资源
    docker-compose down -v
    
    # 清理构建文件
    rm -rf bin/
    
    print_success "清理完成"
}

# 显示帮助
show_help() {
    echo "IM系统管理脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start       启动所有服务"
    echo "  stop        停止所有服务"
    echo "  restart     重启所有服务"
    echo "  status      显示服务状态"
    echo "  build       构建应用"
    echo "  test        运行测试"
    echo "  benchmark   运行性能测试"
    echo "  clean       清理所有资源"
    echo "  help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 start     # 启动所有服务"
    echo "  $0 status    # 查看服务状态"
    echo "  $0 stop      # 停止所有服务"
}

# 主函数
main() {
    case "${1:-help}" in
        start)
            check_dependencies
            start_dependencies
            build_app
            start_app
            show_status
            ;;
        stop)
            stop_services
            ;;
        restart)
            stop_services
            sleep 2
            check_dependencies
            start_dependencies
            build_app
            start_app
            show_status
            ;;
        status)
            show_status
            ;;
        build)
            build_app
            ;;
        test)
            run_tests
            ;;
        benchmark)
            run_benchmark
            ;;
        clean)
            cleanup
            ;;
        help|*)
            show_help
            ;;
    esac
}

# 捕获中断信号
trap 'print_info "收到中断信号，正在停止服务..."; stop_services; exit 0' INT TERM

# 运行主函数
main "$@" 