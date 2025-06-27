package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/user/im/internal/model"
	"github.com/user/im/internal/store"
	"github.com/user/im/pkg/snowflake"
	"github.com/user/im/pkg/websocket"
)

// MessageStoreBackend 消息存储后端接口
type MessageStoreBackend interface {
	SaveMessage(*model.Message) error
	GetMessage(string) (*model.Message, error)
	GetOfflineMessages(userID string, lastMessageID string, limit int) ([]*model.Message, error)
}

// MessageService 消息服务
type MessageService struct {
	storeBackend MessageStoreBackend
	mysqlStore   *store.MySQLStore
	redisStore   *store.RedisStore
	kafkaStore   *store.KafkaStore
	wsManager    *websocket.Manager
}

// NewMessageServiceWithBackend 支持LevelDB/MySQL后端
func NewMessageServiceWithBackend(
	storeBackend MessageStoreBackend,
	redisStore *store.RedisStore,
	kafkaStore *store.KafkaStore,
	wsManager *websocket.Manager,
) *MessageService {
	var mysqlStore *store.MySQLStore
	if ms, ok := storeBackend.(*store.MySQLStore); ok {
		mysqlStore = ms
	}
	return &MessageService{
		storeBackend: storeBackend,
		mysqlStore:   mysqlStore,
		redisStore:   redisStore,
		kafkaStore:   kafkaStore,
		wsManager:    wsManager,
	}
}

// SendPrivateMessage 发送私聊消息
func (s *MessageService) SendPrivateMessage(senderID, receiverID string, msgType model.MessageType, content string) (*model.Message, error) {
	// 生成消息ID
	messageID, err := snowflake.GenerateIDString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate message ID: %w", err)
	}

	// 创建消息
	message := &model.Message{
		ID:         messageID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Status:     model.MessageStatusSent,
		Timestamp:  time.Now().Unix(),
	}

	// 保存到数据库
	if err := s.storeBackend.SaveMessage(message); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// 缓存消息
	s.redisStore.SetMessageCache(messageID, message)

	// 检查接收者是否在线
	if conn, exists := s.wsManager.GetUserConnection(receiverID); exists {
		// 在线，直接推送
		wsMessage := model.WebSocketMessage{
			Type:      "new_message",
			Data:      message,
			Timestamp: time.Now().Unix(),
			MessageID: messageID,
		}

		data, _ := json.Marshal(wsMessage)
		conn.SendMessage(data)

		// 更新消息状态为已投递
		message.Status = model.MessageStatusDelivered
		s.mysqlStore.UpdateMessageStatus(messageID, model.MessageStatusDelivered)
	} else {
		// 离线，发送到Kafka进行异步投递
		if err := s.kafkaStore.SendOfflineMessage(message); err != nil {
			return nil, fmt.Errorf("failed to send offline message: %w", err)
		}

		// 存储到Redis离线消息队列
		s.redisStore.SetOfflineMessage(receiverID, message)
	}

	return message, nil
}

// SendGroupMessage 发送群聊消息
func (s *MessageService) SendGroupMessage(senderID, groupID string, msgType model.MessageType, content string) (*model.Message, error) {
	// 检查发送者是否为群组成员
	isMember, err := s.mysqlStore.IsGroupMember(groupID, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check group membership: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("user %s is not a member of group %s", senderID, groupID)
	}

	// 生成消息ID
	messageID, err := snowflake.GenerateIDString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate message ID: %w", err)
	}

	// 创建消息
	message := &model.Message{
		ID:        messageID,
		SenderID:  senderID,
		GroupID:   groupID,
		Type:      msgType,
		Content:   content,
		Status:    model.MessageStatusSent,
		Timestamp: time.Now().Unix(),
	}

	// 保存到数据库
	if err := s.storeBackend.SaveMessage(message); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// 缓存消息
	s.redisStore.SetMessageCache(messageID, message)

	// 获取群组成员
	members, err := s.mysqlStore.GetGroupMembers(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}

	// 提取用户ID列表
	var userIDs []string
	for _, member := range members {
		if member.UserID != senderID { // 不发送给自己
			userIDs = append(userIDs, member.UserID)
		}
	}

	// 广播消息给群组成员
	s.wsManager.BroadcastToGroup(userIDs, model.WebSocketMessage{
		Type:      "new_group_message",
		Data:      message,
		Timestamp: time.Now().Unix(),
		MessageID: messageID,
	})

	// 发送到Kafka进行异步处理
	if err := s.kafkaStore.SendGroupMessage(groupID, message); err != nil {
		return nil, fmt.Errorf("failed to send group message to kafka: %w", err)
	}

	return message, nil
}

// SyncOfflineMessages 同步离线消息
func (s *MessageService) SyncOfflineMessages(userID, lastMessageID string, limit int) ([]*model.Message, error) {
	// 先从Redis获取离线消息
	messages, err := s.redisStore.GetOfflineMessages(userID, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get offline messages from redis: %w", err)
	}

	// 如果Redis中没有足够的消息，从数据库获取
	if len(messages) < limit {
		dbMessages, err := s.mysqlStore.GetOfflineMessages(userID, lastMessageID, limit-len(messages))
		if err != nil {
			return nil, fmt.Errorf("failed to get offline messages from database: %w", err)
		}
		messages = append(messages, dbMessages...)
	}

	return messages, nil
}

// SyncGroupMessages 同步群聊消息
func (s *MessageService) SyncGroupMessages(groupID, lastMessageID string, limit int) ([]*model.Message, error) {
	return s.mysqlStore.GetGroupMessages(groupID, lastMessageID, limit)
}

// AcknowledgeMessage 确认消息
func (s *MessageService) AcknowledgeMessage(messageID string, status model.MessageStatus) error {
	return s.mysqlStore.UpdateMessageStatus(messageID, status)
}

// GetMessage 获取消息
func (s *MessageService) GetMessage(messageID string) (*model.Message, error) {
	// 先从缓存获取
	if message, err := s.redisStore.GetMessageCache(messageID); err == nil {
		return message, nil
	}

	// 缓存未命中，从数据库获取
	message, err := s.storeBackend.GetMessage(messageID)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	s.redisStore.SetMessageCache(messageID, message)

	return message, nil
}

// CreateGroup 创建群组
func (s *MessageService) CreateGroup(name, description, ownerID string, members []string) (*model.Group, error) {
	// 生成群组ID
	groupID, err := snowflake.GenerateIDString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate group ID: %w", err)
	}

	// 创建群组
	group := &model.Group{
		ID:          groupID,
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		Members:     members,
	}

	if err := s.mysqlStore.CreateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// 添加群组成员
	for _, userID := range members {
		memberID, _ := snowflake.GenerateIDString()
		member := &model.GroupMember{
			ID:       memberID,
			GroupID:  groupID,
			UserID:   userID,
			Role:     "member",
			JoinedAt: time.Now(),
		}

		if userID == ownerID {
			member.Role = "owner"
		}

		if err := s.mysqlStore.AddGroupMember(member); err != nil {
			return nil, fmt.Errorf("failed to add group member: %w", err)
		}
	}

	// 更新Redis缓存
	s.redisStore.SetGroupMembers(groupID, members)

	return group, nil
}

// JoinGroup 加入群组
func (s *MessageService) JoinGroup(groupID, userID string) error {
	// 检查是否已经是群组成员
	isMember, err := s.mysqlStore.IsGroupMember(groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to check group membership: %w", err)
	}
	if isMember {
		return fmt.Errorf("user %s is already a member of group %s", userID, groupID)
	}

	// 添加群组成员
	memberID, err := snowflake.GenerateIDString()
	if err != nil {
		return fmt.Errorf("failed to generate member ID: %w", err)
	}

	member := &model.GroupMember{
		ID:       memberID,
		GroupID:  groupID,
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	if err := s.mysqlStore.AddGroupMember(member); err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	// 更新Redis缓存
	s.redisStore.AddGroupMember(groupID, userID)

	return nil
}

// LeaveGroup 离开群组
func (s *MessageService) LeaveGroup(groupID, userID string) error {
	// 检查是否为群组成员
	isMember, err := s.mysqlStore.IsGroupMember(groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to check group membership: %w", err)
	}
	if !isMember {
		return fmt.Errorf("user %s is not a member of group %s", userID, groupID)
	}

	// 移除群组成员
	if err := s.mysqlStore.RemoveGroupMember(groupID, userID); err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	// 更新Redis缓存
	s.redisStore.RemoveGroupMember(groupID, userID)

	return nil
}

// GetGroup 获取群组信息
func (s *MessageService) GetGroup(groupID string) (*model.Group, error) {
	return s.mysqlStore.GetGroup(groupID)
}

// GetGroupMembers 获取群组成员
func (s *MessageService) GetGroupMembers(groupID string) ([]*model.GroupMember, error) {
	return s.mysqlStore.GetGroupMembers(groupID)
}
