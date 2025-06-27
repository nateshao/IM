package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user/im/internal/config"
	"github.com/user/im/internal/model"
)

// RedisStore Redis存储实现
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore 创建Redis存储实例
func NewRedisStore(cfg *config.RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetAddr(),
		Password: cfg.Password,
		DB:       cfg.Database,
		PoolSize: cfg.PoolSize,
	})

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{
		client: client,
		ctx:    ctx,
	}, nil
}

// SetUserStatus 设置用户状态
func (s *RedisStore) SetUserStatus(userID string, status *model.UserStatus) error {
	key := fmt.Sprintf("user:status:%s", userID)
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	// 设置过期时间为30分钟
	return s.client.Set(s.ctx, key, data, 30*time.Minute).Err()
}

// GetUserStatus 获取用户状态
func (s *RedisStore) GetUserStatus(userID string) (*model.UserStatus, error) {
	key := fmt.Sprintf("user:status:%s", userID)
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var status model.UserStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SetUserConnection 设置用户连接信息
func (s *RedisStore) SetUserConnection(userID, connID string) error {
	key := fmt.Sprintf("user:conn:%s", userID)
	return s.client.Set(s.ctx, key, connID, 30*time.Minute).Err()
}

// GetUserConnection 获取用户连接信息
func (s *RedisStore) GetUserConnection(userID string) (string, error) {
	key := fmt.Sprintf("user:conn:%s", userID)
	return s.client.Get(s.ctx, key).Result()
}

// RemoveUserConnection 移除用户连接信息
func (s *RedisStore) RemoveUserConnection(userID string) error {
	key := fmt.Sprintf("user:conn:%s", userID)
	return s.client.Del(s.ctx, key).Err()
}

// PublishMessage 发布消息到频道
func (s *RedisStore) PublishMessage(channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return s.client.Publish(s.ctx, channel, data).Err()
}

// Subscribe 订阅频道
func (s *RedisStore) Subscribe(channels ...string) *redis.PubSub {
	return s.client.Subscribe(s.ctx, channels...)
}

// SetOfflineMessage 设置离线消息
func (s *RedisStore) SetOfflineMessage(userID string, message *model.Message) error {
	key := fmt.Sprintf("offline:msg:%s", userID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 使用List存储离线消息，过期时间7天
	return s.client.LPush(s.ctx, key, data).Err()
}

// GetOfflineMessages 获取离线消息
func (s *RedisStore) GetOfflineMessages(userID string, limit int64) ([]*model.Message, error) {
	key := fmt.Sprintf("offline:msg:%s", userID)

	// 获取并删除离线消息
	data, err := s.client.LRange(s.ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	var messages []*model.Message
	for _, item := range data {
		var message model.Message
		if err := json.Unmarshal([]byte(item), &message); err != nil {
			continue
		}
		messages = append(messages, &message)
	}

	// 删除已获取的消息
	if len(messages) > 0 {
		s.client.LTrim(s.ctx, key, int64(len(messages)), -1)
	}

	return messages, nil
}

// SetGroupMembers 设置群组成员
func (s *RedisStore) SetGroupMembers(groupID string, members []string) error {
	key := fmt.Sprintf("group:members:%s", groupID)

	// 删除旧数据
	s.client.Del(s.ctx, key)

	// 添加新数据
	if len(members) > 0 {
		args := make([]interface{}, len(members))
		for i, member := range members {
			args[i] = member
		}
		return s.client.SAdd(s.ctx, key, args...).Err()
	}

	return nil
}

// GetGroupMembers 获取群组成员
func (s *RedisStore) GetGroupMembers(groupID string) ([]string, error) {
	key := fmt.Sprintf("group:members:%s", groupID)
	return s.client.SMembers(s.ctx, key).Result()
}

// AddGroupMember 添加群组成员
func (s *RedisStore) AddGroupMember(groupID, userID string) error {
	key := fmt.Sprintf("group:members:%s", groupID)
	return s.client.SAdd(s.ctx, key, userID).Err()
}

// RemoveGroupMember 移除群组成员
func (s *RedisStore) RemoveGroupMember(groupID, userID string) error {
	key := fmt.Sprintf("group:members:%s", groupID)
	return s.client.SRem(s.ctx, key, userID).Err()
}

// IsGroupMember 检查是否为群组成员
func (s *RedisStore) IsGroupMember(groupID, userID string) (bool, error) {
	key := fmt.Sprintf("group:members:%s", groupID)
	return s.client.SIsMember(s.ctx, key, userID).Result()
}

// SetMessageCache 设置消息缓存
func (s *RedisStore) SetMessageCache(messageID string, message *model.Message) error {
	key := fmt.Sprintf("msg:cache:%s", messageID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 缓存1小时
	return s.client.Set(s.ctx, key, data, time.Hour).Err()
}

// GetMessageCache 获取消息缓存
func (s *RedisStore) GetMessageCache(messageID string) (*model.Message, error) {
	key := fmt.Sprintf("msg:cache:%s", messageID)
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var message model.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}

	return &message, nil
}

// Close 关闭Redis连接
func (s *RedisStore) Close() error {
	return s.client.Close()
}
