# IM 系统 Docker Compose 部署与联调排障记录

## 1. 环境准备与服务启动

### 1.1 依赖服务启动
- 使用 `docker-compose up -d` 启动 MySQL、Redis、Zookeeper、Kafka、IM Server 等服务。
- 遇到端口冲突（如 3306、6379、9092），需先停止本地已占用端口的服务或容器（如本地 Redis、MySQL、Kafka）。

### 1.2 配置文件挂载与修正
- `config.yaml` 中的服务主机名需与 Docker Compose 服务名一致：
  - `database.host: mysql`
  - `redis.host: redis`
  - `kafka.brokers: ["kafka:9092"]`
- 确认 `docker-compose.yml` 中 `im-server` 服务的 `volumes` 配置为：
  ```yaml
  volumes:
    - ./config.yaml:/app/config.yaml
  ```
- Dockerfile 中注释掉 `COPY --from=builder /app/config.yaml .`，避免覆盖挂载的配置文件。

### 1.3 镜像重建与容器重启
- 变更配置后需执行：
  ```sh
  docker-compose build im-server
  docker-compose up -d
  ```
- 若配置未生效，需彻底 `docker-compose down`，删除旧镜像后再 `up`。

---

## 2. 常见问题与排查

### 2.1 容器端口冲突
- 端口被本地服务占用时，需停止本地服务或相关容器（如 `brew services stop redis`、`docker rm -f <容器ID>`）。

### 2.2 配置未生效
- 容器内 `/app/config.yaml` 不是最新内容，通常是 Dockerfile 复制覆盖或挂载路径错误。
- 需注释 Dockerfile 的 `COPY config.yaml`，并确认 `volumes` 挂载无误。

### 2.3 容器 DNS 解析失败
- 日志报 `lookup kafka on 127.0.0.11:53: no such host`，说明 Kafka 容器未启动或网络未 ready。
- 需先确保 Kafka 容器健康运行，再启动依赖它的服务。

### 2.4 Kafka 容器未启动
- 用 `docker ps -a` 查看 Kafka 容器状态，若为 Exited，需用 `docker logs <kafka容器ID>` 查看启动失败原因并修复。

### 2.5 服务依赖未 ready
- Docker Compose 的 `depends_on` 只保证容器启动，不保证服务 ready。
- 需等待依赖服务完全 ready 后再启动主服务，或在主服务中实现重试机制。

---

## 3. WebSocket/HTTP API 验证建议

### 3.1 WebSocket 功能验证
- 终端1：运行 WebSocket 客户端，登录 user1
  ```sh
  go run cmd/client/main.go ws://localhost:8080/ws user1
  ```
- 终端2：运行 WebSocket 客户端，登录 user2
  ```sh
  go run cmd/client/main.go ws://localhost:8080/ws user2
  ```
- 在 user1 客户端输入：
  ```
  send user2 hello
  ```
- 检查 user2 是否收到消息。

### 3.2 WebSocket 性能/并发验证
  ```sh
  go run scripts/benchmark/websocket_test.go
  ```

### 3.3 HTTP REST API 验证
- 用 curl/Postman 按 API 文档测试 REST 接口。

---

> 本文档可持续补充，建议每次联调遇到新问题时及时记录。 