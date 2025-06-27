package store

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/user/im/internal/model"
)

// LevelDBStore LevelDB存储实现
type LevelDBStore struct {
	db   *leveldb.DB
	lock sync.RWMutex
}

// NewLevelDBStore 创建LevelDB存储实例
func NewLevelDBStore(dbPath string) (*LevelDBStore, error) {
	db, err := leveldb.OpenFile(filepath.Clean(dbPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	return &LevelDBStore{db: db}, nil
}

// SaveMessage 保存消息
func (s *LevelDBStore) SaveMessage(message *model.Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	key := s.messageKey(message.ID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(key), data, nil)
}

// GetMessage 获取消息
func (s *LevelDBStore) GetMessage(messageID string) (*model.Message, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	key := s.messageKey(messageID)
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		return nil, err
	}
	var message model.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// GetOfflineMessages 获取离线消息（按时间顺序）
func (s *LevelDBStore) GetOfflineMessages(userID string, lastMessageID string, limit int) ([]*model.Message, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	prefix := s.offlineKey(userID)
	var messages []*model.Message
	iter := s.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	count := 0
	for iter.Next() {
		if count >= limit {
			break
		}
		var message model.Message
		if err := json.Unmarshal(iter.Value(), &message); err == nil {
			if lastMessageID == "" || message.ID > lastMessageID {
				messages = append(messages, &message)
				count++
			}
		}
	}
	iter.Release()
	return messages, nil
}

// SetOfflineMessage 添加离线消息
func (s *LevelDBStore) SetOfflineMessage(userID string, message *model.Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	key := s.offlineKey(userID) + message.ID
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(key), data, nil)
}

// RemoveOfflineMessage 删除离线消息
func (s *LevelDBStore) RemoveOfflineMessage(userID, messageID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	key := s.offlineKey(userID) + messageID
	return s.db.Delete([]byte(key), nil)
}

// Close 关闭LevelDB
func (s *LevelDBStore) Close() error {
	return s.db.Close()
}

// messageKey 消息主键
func (s *LevelDBStore) messageKey(messageID string) string {
	return "msg:" + messageID
}

// offlineKey 离线消息前缀
func (s *LevelDBStore) offlineKey(userID string) string {
	return "offline:" + userID + ":"
}
