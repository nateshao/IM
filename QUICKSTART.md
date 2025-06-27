# IM系统快速开始指南

## 🚀 5分钟快速启动

### 1. 环境准备

确保你的系统已安装以下软件：

- **Docker** (20.10+) 和 **Docker Compose** (2.0+)
- **Go** (1.21+)
- **Git**

### 2. 克隆项目

```bash
git clone <repository-url>
cd IM
```

### 3. 一键启动

```bash
# 使用启动脚本
chmod +x scripts/start.sh
./scripts/start.sh start

# 或者使用Makefile
make start
```

### 4. 验证服务

访问以下地址验证服务是否正常：

- **IM服务器**: http://localhost:8080
- **健康检查**: http://localhost:8080/health
- **监控指标**: http://localhost:8080/metrics
- **Kafka UI**: http://localhost:8081
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

## 📱 测试客户端

### 启动测试客户端

```bash
# 构建客户端
make build

# 启动客户端1
./bin/im-client ws://localhost:8080/ws user1

# 启动客户端2（新终端）
./bin/im-client ws://localhost:8080/ws user2
```

### 发送消息

在客户端中输入：

```
send user2 Hello, this is a test message!
```

## 🔧 常用命令

### 服务管理

```bash
# 启动所有服务
make start

# 停止所有服务
make stop

# 重启所有服务
make restart

# 查看服务状态
make status

# 查看服务日志
make logs
```

### 开发相关

```bash
# 构建应用
make build

# 运行测试
make test

# 运行性能测试
make benchmark

# 格式化代码
make fmt

# 代码检查
make lint
```

### Docker相关

```bash
# 构建Docker镜像
make docker-build

# 启动Docker容器
make docker-run

# 停止Docker容器
make docker-stop
```

## 📊 性能测试

### 运行基准测试

```bash
# 运行WebSocket性能测试
make benchmark
```

测试结果示例：
```
WebSocket Performance Benchmark
================================

Running Small Load Test:
- Clients: 10
- Duration: 30s
- Message Interval: 1s

Results:
- Total Messages Sent: 300
- Total Messages Received: 300
- Total Errors: 0
- Messages per second: 10.00
- Connections per second: 0.33
- Success Rate: 100.00%
```

### 压力测试

```bash
# 使用wrk进行HTTP API压力测试
wrk -t12 -c400 -d30s http://localhost:8080/health

# 使用websocat进行WebSocket压力测试
for i in {1..100}; do
  websocat ws://localhost:8080/ws &
done
```

## 🔍 监控和调试

### 查看系统指标

1. **Grafana仪表板**: http://localhost:3000
   - 用户名: `admin`
   - 密码: `admin`

2. **Prometheus指标**: http://localhost:9090

3. **Kafka管理界面**: http://localhost:8081

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f im-server
docker-compose logs -f mysql
docker-compose logs -f redis
docker-compose logs -f kafka
```

### 调试模式

```bash
# 以调试模式启动
LOG_LEVEL=debug make start

# 查看详细日志
tail -f logs/im-server.log
```

## 🛠️ 配置说明

### 主要配置文件

- `config.yaml`: 应用主配置文件
- `docker-compose.yml`: Docker服务编排
- `Dockerfile`: Docker镜像构建

### 关键配置项

```yaml
server:
  port: 8080                    # 服务端口
  max_connections: 100000       # 最大连接数
  heartbeat_interval: 30s       # 心跳间隔

database:
  host: localhost               # 数据库地址
  port: 3306                    # 数据库端口
  database: im_db               # 数据库名

redis:
  host: localhost               # Redis地址
  port: 6379                    # Redis端口

kafka:
  brokers: ["localhost:9092"]   # Kafka地址
```

## 🔧 故障排除

### 常见问题

#### 1. 端口被占用

```bash
# 查看端口占用
lsof -i :8080

# 杀死占用进程
kill -9 <PID>
```

#### 2. 数据库连接失败

```bash
# 检查MySQL状态
docker-compose ps mysql

# 查看MySQL日志
docker-compose logs mysql

# 重启MySQL
docker-compose restart mysql
```

#### 3. Redis连接失败

```bash
# 检查Redis状态
docker-compose ps redis

# 测试Redis连接
docker-compose exec redis redis-cli ping
```

#### 4. Kafka连接失败

```bash
# 检查Kafka状态
docker-compose ps kafka

# 查看Kafka日志
docker-compose logs kafka

# 重启Kafka
docker-compose restart kafka
```

### 性能调优

#### 1. 增加连接数

```yaml
# config.yaml
server:
  max_connections: 200000  # 增加最大连接数
```

#### 2. 调整数据库连接池

```yaml
# config.yaml
database:
  max_open: 200    # 增加最大连接数
  max_idle: 20     # 增加空闲连接数
```

#### 3. 调整Redis连接池

```yaml
# config.yaml
redis:
  pool_size: 50    # 增加连接池大小
```

## 📚 进阶使用

### 1. 集群部署

```bash
# 启动多个实例
docker-compose up -d --scale im-server=3
```

### 2. 负载均衡

```bash
# 使用Nginx进行负载均衡
docker-compose -f docker-compose.yml -f docker-compose.nginx.yml up -d
```

### 3. 数据备份

```bash
# 备份数据库
make backup

# 恢复数据库
make restore FILE=backups/im_db_20240101_120000.sql
```

### 4. 监控告警

```bash
# 配置Prometheus告警规则
cp deployments/prometheus/alerts.yml /etc/prometheus/

# 配置Grafana告警
# 在Grafana界面中配置告警规则
```

## 🎯 开发指南

### 1. 添加新功能

```bash
# 创建新分支
git checkout -b feature/new-feature

# 开发完成后提交
git add .
git commit -m "feat: add new feature"
git push origin feature/new-feature
```

### 2. 运行测试

```bash
# 运行所有测试
make test

# 运行特定测试
go test ./internal/service -v

# 生成测试覆盖率报告
make coverage
```

### 3. 代码质量检查

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 安全扫描
make security
```

## 📞 获取帮助

### 文档

- [API文档](docs/api/README.md)
- [架构设计](docs/design/architecture.md)
- [部署指南](docs/deployment/README.md)

### 社区

- 提交Issue: [GitHub Issues](https://github.com/your-repo/issues)
- 讨论: [GitHub Discussions](https://github.com/your-repo/discussions)

### 联系方式

- 邮箱: your-email@example.com
- 微信: your-wechat-id

## 🎉 恭喜！

你已经成功启动了IM系统！现在可以开始探索和使用这个高性能的即时通讯系统了。

如果遇到任何问题，请查看故障排除部分或联系技术支持。

## 🗄️ 切换为LevelDB本地存储

如需极致本地性能，可切换为LevelDB：

1. 编辑 `config.yaml`：

```yaml
store:
  type: "leveldb"
  leveldb_path: "./data/leveldb"
```

2. 重启服务即可。

> LevelDB 适合单机高性能场景，所有消息数据存储在本地目录。 