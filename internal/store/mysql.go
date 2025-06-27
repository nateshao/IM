package store

import (
	"fmt"
	"time"

	"github.com/user/im/internal/config"
	"github.com/user/im/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLStore MySQL存储实现
type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore 创建MySQL存储实例
func NewMySQLStore(cfg *config.DatabaseConfig) (*MySQLStore, error) {
	dsn := cfg.GetDSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&model.Message{},
		&model.Group{},
		&model.GroupMember{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return &MySQLStore{db: db}, nil
}

// SaveMessage 保存消息
func (s *MySQLStore) SaveMessage(message *model.Message) error {
	return s.db.Create(message).Error
}

// GetMessage 获取消息
func (s *MySQLStore) GetMessage(messageID string) (*model.Message, error) {
	var message model.Message
	err := s.db.Where("id = ?", messageID).First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetOfflineMessages 获取离线消息
func (s *MySQLStore) GetOfflineMessages(userID string, lastMessageID string, limit int) ([]*model.Message, error) {
	var messages []*model.Message

	query := s.db.Where("receiver_id = ? AND group_id = ''", userID)
	if lastMessageID != "" {
		query = query.Where("id > ?", lastMessageID)
	}

	err := query.Order("timestamp ASC").Limit(limit).Find(&messages).Error
	return messages, err
}

// GetGroupMessages 获取群聊消息
func (s *MySQLStore) GetGroupMessages(groupID string, lastMessageID string, limit int) ([]*model.Message, error) {
	var messages []*model.Message

	query := s.db.Where("group_id = ?", groupID)
	if lastMessageID != "" {
		query = query.Where("id > ?", lastMessageID)
	}

	err := query.Order("timestamp ASC").Limit(limit).Find(&messages).Error
	return messages, err
}

// UpdateMessageStatus 更新消息状态
func (s *MySQLStore) UpdateMessageStatus(messageID string, status model.MessageStatus) error {
	return s.db.Model(&model.Message{}).Where("id = ?", messageID).Update("status", status).Error
}

// GetGroup 获取群组信息
func (s *MySQLStore) GetGroup(groupID string) (*model.Group, error) {
	var group model.Group
	err := s.db.Where("id = ?", groupID).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// CreateGroup 创建群组
func (s *MySQLStore) CreateGroup(group *model.Group) error {
	return s.db.Create(group).Error
}

// GetGroupMembers 获取群组成员
func (s *MySQLStore) GetGroupMembers(groupID string) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	err := s.db.Where("group_id = ?", groupID).Find(&members).Error
	return members, err
}

// AddGroupMember 添加群组成员
func (s *MySQLStore) AddGroupMember(member *model.GroupMember) error {
	return s.db.Create(member).Error
}

// RemoveGroupMember 移除群组成员
func (s *MySQLStore) RemoveGroupMember(groupID, userID string) error {
	return s.db.Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&model.GroupMember{}).Error
}

// IsGroupMember 检查是否为群组成员
func (s *MySQLStore) IsGroupMember(groupID, userID string) (bool, error) {
	var count int64
	err := s.db.Model(&model.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&count).Error
	return count > 0, err
}

// Close 关闭数据库连接
func (s *MySQLStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
