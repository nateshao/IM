package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/im/internal/model"
)

// Connection WebSocket连接
type Connection struct {
	ID      string
	UserID  string
	Conn    *websocket.Conn
	Send    chan []byte
	Manager *Manager
	mu      sync.Mutex
	closed  bool
}

// Manager WebSocket连接管理器
type Manager struct {
	connections map[string]*Connection // connID -> Connection
	users       map[string]*Connection // userID -> Connection
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

// NewManager 创建连接管理器
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
		users:       make(map[string]*Connection),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境需要限制
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket 处理WebSocket连接
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade connection: %v\n", err)
		return
	}

	connection := &Connection{
		ID:      generateConnID(),
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Manager: m,
	}

	m.addConnection(connection)

	// 启动读写协程
	go connection.readPump()
	go connection.writePump()
}

// addConnection 添加连接
func (m *Manager) addConnection(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[conn.ID] = conn
}

// removeConnection 移除连接
func (m *Manager) removeConnection(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.connections, conn.ID)
	if conn.UserID != "" {
		delete(m.users, conn.UserID)
	}
}

// setUserConnection 设置用户连接
func (m *Manager) setUserConnection(userID string, conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果用户已有连接，先关闭旧连接
	if oldConn, exists := m.users[userID]; exists {
		oldConn.close()
	}

	m.users[userID] = conn
	conn.UserID = userID
}

// GetUserConnection 获取用户连接
func (m *Manager) GetUserConnection(userID string) (*Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.users[userID]
	return conn, exists
}

// SendToUser 发送消息给用户
func (m *Manager) SendToUser(userID string, message interface{}) error {
	conn, exists := m.GetUserConnection(userID)
	if !exists {
		return fmt.Errorf("user %s not connected", userID)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return conn.SendMessage(data)
}

// BroadcastToGroup 广播消息给群组
func (m *Manager) BroadcastToGroup(groupMembers []string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("Failed to marshal message: %v\n", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, userID := range groupMembers {
		if conn, exists := m.users[userID]; exists {
			conn.SendMessage(data)
		}
	}
}

// GetConnectionCount 获取连接数
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// GetOnlineUserCount 获取在线用户数
func (m *Manager) GetOnlineUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

// CloseAll 关闭所有连接
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connections {
		conn.close()
	}
}

// readPump 读取消息泵
func (c *Connection) readPump() {
	defer func() {
		c.Manager.removeConnection(c)
		c.close()
	}()

	c.Conn.SetReadLimit(512) // 限制消息大小
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket read error: %v\n", err)
			}
			break
		}

		// 处理消息
		c.handleMessage(message)
	}
}

// writePump 写入消息泵
func (c *Connection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage 发送消息
func (c *Connection) SendMessage(message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("connection is closed")
	}

	select {
	case c.Send <- message:
		return nil
	default:
		return fmt.Errorf("send buffer is full")
	}
}

// close 关闭连接
func (c *Connection) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	close(c.Send)
	c.Conn.Close()
}

// handleMessage 处理消息
func (c *Connection) handleMessage(data []byte) {
	var wsMessage model.WebSocketMessage
	if err := json.Unmarshal(data, &wsMessage); err != nil {
		c.sendError("Invalid message format")
		return
	}

	switch wsMessage.Type {
	case "login":
		c.handleLogin(wsMessage.Data)
	case "heartbeat":
		c.handleHeartbeat(wsMessage.Data)
	case "send_message":
		c.handleSendMessage(wsMessage.Data)
	case "ack":
		c.handleAck(wsMessage.Data)
	case "sync_offline":
		c.handleSyncOffline(wsMessage.Data)
	case "join_group":
		c.handleJoinGroup(wsMessage.Data)
	case "leave_group":
		c.handleLeaveGroup(wsMessage.Data)
	default:
		c.sendError("Unknown message type")
	}
}

// handleLogin 处理登录
func (c *Connection) handleLogin(data interface{}) {
	// 这里应该验证用户身份
	// 简化处理，直接设置用户ID
	if userData, ok := data.(map[string]interface{}); ok {
		if userID, ok := userData["user_id"].(string); ok {
			c.Manager.setUserConnection(userID, c)
			c.sendResponse("login", model.LoginResponse{
				Success: true,
				Message: "Login successful",
				UserID:  userID,
			})
			return
		}
	}
	c.sendError("Invalid login data")
}

// handleHeartbeat 处理心跳
func (c *Connection) handleHeartbeat(data interface{}) {
	c.sendResponse("heartbeat", model.HeartbeatResponse{
		Timestamp: time.Now().Unix(),
	})
}

// handleSendMessage 处理发送消息
func (c *Connection) handleSendMessage(data interface{}) {
	// 这里应该实现消息发送逻辑
	c.sendResponse("send_message", map[string]interface{}{
		"success": true,
		"message": "Message sent",
	})
}

// handleAck 处理消息确认
func (c *Connection) handleAck(data interface{}) {
	// 这里应该实现消息确认逻辑
}

// handleSyncOffline 处理同步离线消息
func (c *Connection) handleSyncOffline(data interface{}) {
	// 这里应该实现离线消息同步逻辑
	c.sendResponse("sync_offline", model.SyncOfflineResponse{
		Messages: []*model.Message{},
		HasMore:  false,
	})
}

// handleJoinGroup 处理加入群聊
func (c *Connection) handleJoinGroup(data interface{}) {
	// 这里应该实现加入群聊逻辑
}

// handleLeaveGroup 处理离开群聊
func (c *Connection) handleLeaveGroup(data interface{}) {
	// 这里应该实现离开群聊逻辑
}

// sendResponse 发送响应
func (c *Connection) sendResponse(msgType string, data interface{}) {
	response := model.WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Failed to marshal response: %v\n", err)
		return
	}

	c.SendMessage(responseData)
}

// sendError 发送错误响应
func (c *Connection) sendError(message string) {
	c.sendResponse("error", map[string]interface{}{
		"error": message,
	})
}

// generateConnID 生成连接ID
func generateConnID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}
