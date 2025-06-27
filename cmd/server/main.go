package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/user/im/internal/config"
	"github.com/user/im/internal/model"
	"github.com/user/im/internal/service"
	"github.com/user/im/internal/store"
	"github.com/user/im/pkg/logger"
	"github.com/user/im/pkg/snowflake"
	"github.com/user/im/pkg/websocket"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(cfg.Log.Level, cfg.Log.Format); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting IM Server...")

	// 初始化Snowflake ID生成器
	snowflake.Init(1)

	// 初始化存储层
	var (
		mysqlStore   *store.MySQLStore
		leveldbStore *store.LevelDBStore
		storeBackend interface {
			SaveMessage(*model.Message) error
			GetMessage(string) (*model.Message, error)
			GetOfflineMessages(string, string, int) ([]*model.Message, error)
		}
	)

	if cfg.Store.Type == "leveldb" {
		leveldbStore, err = store.NewLevelDBStore(cfg.Store.LevelDBPath)
		if err != nil {
			logger.Fatal("Failed to initialize LevelDB store", logger.ErrorField(err))
		}
		defer leveldbStore.Close()
		storeBackend = leveldbStore
		logger.Info("Using LevelDB as message store", logger.String("path", cfg.Store.LevelDBPath))
	} else {
		mysqlStore, err = store.NewMySQLStore(&cfg.Database)
		if err != nil {
			logger.Fatal("Failed to initialize MySQL store", logger.ErrorField(err))
		}
		defer mysqlStore.Close()
		storeBackend = mysqlStore
		logger.Info("Using MySQL as message store")
	}

	redisStore, err := store.NewRedisStore(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis store", logger.ErrorField(err))
	}
	defer redisStore.Close()

	kafkaStore, err := store.NewKafkaStore(&cfg.Kafka)
	if err != nil {
		logger.Fatal("Failed to initialize Kafka store", logger.ErrorField(err))
	}
	defer kafkaStore.Close()

	// 初始化WebSocket管理器
	wsManager := websocket.NewManager()

	// 初始化消息服务
	messageService := service.NewMessageServiceWithBackend(storeBackend, redisStore, kafkaStore, wsManager)

	// 启动Kafka消费者
	go startKafkaConsumers(kafkaStore, messageService, wsManager)

	// 启动心跳检测
	go startHeartbeatChecker(wsManager, redisStore)

	// 创建HTTP服务器
	router := gin.Default()

	// 添加中间件
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// 监控指标
	if cfg.Monitor.Enabled {
		router.GET(cfg.Monitor.Path, gin.WrapH(promhttp.Handler()))
	}

	// WebSocket路由
	router.GET("/ws", func(c *gin.Context) {
		wsManager.HandleWebSocket(c.Writer, c.Request)
	})

	// API路由
	api := router.Group("/api/v1")
	{
		// 消息相关API
		api.POST("/messages", handleSendMessage(messageService))
		api.GET("/messages/:messageID", handleGetMessage(messageService))
		api.POST("/messages/:messageID/ack", handleAckMessage(messageService))

		// 离线消息同步
		api.GET("/messages/offline", handleSyncOfflineMessages(messageService))

		// 群组相关API
		api.POST("/groups", handleCreateGroup(messageService))
		api.GET("/groups/:groupID", handleGetGroup(messageService))
		api.GET("/groups/:groupID/members", handleGetGroupMembers(messageService))
		api.POST("/groups/:groupID/join", handleJoinGroup(messageService))
		api.POST("/groups/:groupID/leave", handleLeaveGroup(messageService))

		// 统计信息
		api.GET("/stats", handleGetStats(wsManager))
	}

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		logger.Info("Starting HTTP server",
			logger.String("addr", server.Addr),
			logger.Int("port", cfg.Server.Port))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", logger.ErrorField(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logger.ErrorField(err))
	}

	// 关闭所有WebSocket连接
	wsManager.CloseAll()

	logger.Info("Server exited")
}

// startKafkaConsumers 启动Kafka消费者
func startKafkaConsumers(kafkaStore *store.KafkaStore, messageService *service.MessageService, wsManager *websocket.Manager) {
	// 消费离线消息
	go func() {
		if err := kafkaStore.ConsumeOfflineMessages(func(message *model.Message) error {
			// 检查用户是否在线
			if conn, exists := wsManager.GetUserConnection(message.ReceiverID); exists {
				// 发送消息给在线用户
				wsMessage := model.WebSocketMessage{
					Type:      "new_message",
					Data:      message,
					Timestamp: time.Now().Unix(),
					MessageID: message.ID,
				}

				data, _ := json.Marshal(wsMessage)
				conn.SendMessage(data)

				// 更新消息状态
				messageService.AcknowledgeMessage(message.ID, model.MessageStatusDelivered)
			}
			return nil
		}); err != nil {
			logger.Error("Failed to consume offline messages", logger.ErrorField(err))
		}
	}()

	// 消费群聊消息
	go func() {
		if err := kafkaStore.ConsumeGroupMessages(func(message *model.Message) error {
			// 获取群组成员并广播消息
			members, err := messageService.GetGroupMembers(message.GroupID)
			if err != nil {
				return err
			}

			var userIDs []string
			for _, member := range members {
				if member.UserID != message.SenderID {
					userIDs = append(userIDs, member.UserID)
				}
			}

			wsManager.BroadcastToGroup(userIDs, model.WebSocketMessage{
				Type:      "new_group_message",
				Data:      message,
				Timestamp: time.Now().Unix(),
				MessageID: message.ID,
			})

			return nil
		}); err != nil {
			logger.Error("Failed to consume group messages", logger.ErrorField(err))
		}
	}()
}

// startHeartbeatChecker 启动心跳检测
func startHeartbeatChecker(wsManager *websocket.Manager, redisStore *store.RedisStore) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 检查连接状态
		connectionCount := wsManager.GetConnectionCount()
		onlineUserCount := wsManager.GetOnlineUserCount()

		logger.Debug("Heartbeat check",
			logger.Int("connections", connectionCount),
			logger.Int("online_users", onlineUserCount))
	}
}

// HTTP处理器函数
func handleSendMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ReceiverID string `json:"receiver_id"`
			GroupID    string `json:"group_id"`
			Type       string `json:"type"`
			Content    string `json:"content"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// 从请求中获取发送者ID（实际应用中应该从认证中获取）
		senderID := c.GetHeader("X-User-ID")
		if senderID == "" {
			c.JSON(401, gin.H{"error": "User ID required"})
			return
		}

		var message *model.Message
		var err error

		if req.GroupID != "" {
			// 发送群聊消息
			message, err = messageService.SendGroupMessage(senderID, req.GroupID, model.MessageType(req.Type), req.Content)
		} else {
			// 发送私聊消息
			message, err = messageService.SendPrivateMessage(senderID, req.ReceiverID, model.MessageType(req.Type), req.Content)
		}

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"success":    true,
			"message":    message,
			"message_id": message.ID,
		})
	}
}

func handleGetMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		messageID := c.Param("messageID")

		message, err := messageService.GetMessage(messageID)
		if err != nil {
			c.JSON(404, gin.H{"error": "Message not found"})
			return
		}

		c.JSON(200, gin.H{"message": message})
	}
}

func handleAckMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		messageID := c.Param("messageID")

		var req struct {
			Status string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		err := messageService.AcknowledgeMessage(messageID, model.MessageStatus(req.Status))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"success": true})
	}
}

func handleSyncOfflineMessages(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(401, gin.H{"error": "User ID required"})
			return
		}

		lastMessageID := c.Query("last_message_id")
		limit := 50 // 默认限制

		messages, err := messageService.SyncOfflineMessages(userID, lastMessageID, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"messages": messages,
			"has_more": len(messages) == limit,
		})
	}
}

func handleCreateGroup(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Members     []string `json:"members"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		ownerID := c.GetHeader("X-User-ID")
		if ownerID == "" {
			c.JSON(401, gin.H{"error": "User ID required"})
			return
		}

		group, err := messageService.CreateGroup(req.Name, req.Description, ownerID, req.Members)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"group": group})
	}
}

func handleGetGroup(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("groupID")

		group, err := messageService.GetGroup(groupID)
		if err != nil {
			c.JSON(404, gin.H{"error": "Group not found"})
			return
		}

		c.JSON(200, gin.H{"group": group})
	}
}

func handleGetGroupMembers(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("groupID")

		members, err := messageService.GetGroupMembers(groupID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"members": members})
	}
}

func handleJoinGroup(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("groupID")
		userID := c.GetHeader("X-User-ID")

		if userID == "" {
			c.JSON(401, gin.H{"error": "User ID required"})
			return
		}

		err := messageService.JoinGroup(groupID, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"success": true})
	}
}

func handleLeaveGroup(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("groupID")
		userID := c.GetHeader("X-User-ID")

		if userID == "" {
			c.JSON(401, gin.H{"error": "User ID required"})
			return
		}

		err := messageService.LeaveGroup(groupID, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"success": true})
	}
}

func handleGetStats(wsManager *websocket.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"connections":  wsManager.GetConnectionCount(),
			"online_users": wsManager.GetOnlineUserCount(),
			"timestamp":    time.Now().Unix(),
		})
	}
}
