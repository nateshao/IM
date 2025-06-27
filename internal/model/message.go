package model

import (
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeFile   MessageType = "file"
	MessageTypeVoice  MessageType = "voice"
	MessageTypeVideo  MessageType = "video"
	MessageTypeSystem MessageType = "system"
)

// MessageStatus 消息状态
type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// Message 消息模型
type Message struct {
	ID         string        `json:"id" gorm:"primaryKey;type:varchar(64)"`
	SenderID   string        `json:"sender_id" gorm:"type:varchar(64);index"`
	ReceiverID string        `json:"receiver_id" gorm:"type:varchar(64);index"`
	GroupID    string        `json:"group_id" gorm:"type:varchar(64);index"`
	Type       MessageType   `json:"type" gorm:"type:varchar(20)"`
	Content    string        `json:"content" gorm:"type:text"`
	Status     MessageStatus `json:"status" gorm:"type:varchar(20);default:'sent'"`
	Timestamp  int64         `json:"timestamp" gorm:"index"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// IsGroupMessage 判断是否为群聊消息
func (m *Message) IsGroupMessage() bool {
	return m.GroupID != ""
}

// IsPrivateMessage 判断是否为私聊消息
func (m *Message) IsPrivateMessage() bool {
	return m.GroupID == ""
}

// WebSocketMessage WebSocket消息格式
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	MessageID string      `json:"message_id,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	UserID   string `json:"user_id"`
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	ReceiverID string      `json:"receiver_id"`
	GroupID    string      `json:"group_id,omitempty"`
	Type       MessageType `json:"type"`
	Content    string      `json:"content"`
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	Success   bool     `json:"success"`
	MessageID string   `json:"message_id"`
	Message   *Message `json:"message"`
}

// AckRequest 消息确认请求
type AckRequest struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

// SyncOfflineRequest 同步离线消息请求
type SyncOfflineRequest struct {
	LastMessageID string `json:"last_message_id"`
	Limit         int    `json:"limit"`
}

// SyncOfflineResponse 同步离线消息响应
type SyncOfflineResponse struct {
	Messages []*Message `json:"messages"`
	HasMore  bool       `json:"has_more"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	UserID string `json:"user_id"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Timestamp int64 `json:"timestamp"`
}

// JoinGroupRequest 加入群聊请求
type JoinGroupRequest struct {
	GroupID string `json:"group_id"`
}

// LeaveGroupRequest 离开群聊请求
type LeaveGroupRequest struct {
	GroupID string `json:"group_id"`
}

// GroupMessage 群聊消息
type GroupMessage struct {
	GroupID string   `json:"group_id"`
	Message *Message `json:"message"`
}

// UserStatus 用户状态
type UserStatus struct {
	UserID   string    `json:"user_id"`
	Status   string    `json:"status"` // online, offline, away
	LastSeen time.Time `json:"last_seen"`
	Platform string    `json:"platform"`
	ConnID   string    `json:"conn_id"`
}

// Group 群组模型
type Group struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	Name        string    `json:"name" gorm:"type:varchar(100)"`
	Description string    `json:"description" gorm:"type:text"`
	OwnerID     string    `json:"owner_id" gorm:"type:varchar(64)"`
	Members     []string  `json:"members" gorm:"type:json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupMember 群组成员
type GroupMember struct {
	ID       string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	GroupID  string    `json:"group_id" gorm:"type:varchar(64);index"`
	UserID   string    `json:"user_id" gorm:"type:varchar(64);index"`
	Role     string    `json:"role" gorm:"type:varchar(20)"` // owner, admin, member
	JoinedAt time.Time `json:"joined_at"`
}
