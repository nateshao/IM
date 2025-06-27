package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/user/im/internal/config"
	"github.com/user/im/internal/model"
)

// KafkaStore Kafka存储实现
type KafkaStore struct {
	config *config.KafkaConfig
	ctx    context.Context
}

// NewKafkaStore 创建Kafka存储实例
func NewKafkaStore(cfg *config.KafkaConfig) (*KafkaStore, error) {
	ctx := context.Background()

	// 测试连接
	conn, err := kafka.DialLeader(ctx, "tcp", cfg.Brokers[0], cfg.Topics.MessageQueue, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()

	return &KafkaStore{
		config: cfg,
		ctx:    ctx,
	}, nil
}

// SendMessage 发送消息到队列
func (s *KafkaStore) SendMessage(topic string, message *model.Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(s.config.Brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	return writer.WriteMessages(s.ctx, kafka.Message{
		Key:   []byte(message.ID),
		Value: data,
	})
}

// SendGroupMessage 发送群聊消息
func (s *KafkaStore) SendGroupMessage(groupID string, message *model.Message) error {
	return s.SendMessage(s.config.Topics.GroupChat, message)
}

// SendOfflineMessage 发送离线消息
func (s *KafkaStore) SendOfflineMessage(message *model.Message) error {
	return s.SendMessage(s.config.Topics.OfflineMsg, message)
}

// ConsumeMessages 消费消息
func (s *KafkaStore) ConsumeMessages(topic string, handler func(*model.Message) error) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  s.config.Brokers,
		Topic:    topic,
		GroupID:  s.config.GroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(s.ctx)
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}

		var message model.Message
		if err := json.Unmarshal(msg.Value, &message); err != nil {
			continue
		}

		if err := handler(&message); err != nil {
			// 记录错误但继续处理
			fmt.Printf("Error handling message: %v\n", err)
		}
	}
}

// ConsumeGroupMessages 消费群聊消息
func (s *KafkaStore) ConsumeGroupMessages(handler func(*model.Message) error) error {
	return s.ConsumeMessages(s.config.Topics.GroupChat, handler)
}

// ConsumeOfflineMessages 消费离线消息
func (s *KafkaStore) ConsumeOfflineMessages(handler func(*model.Message) error) error {
	return s.ConsumeMessages(s.config.Topics.OfflineMsg, handler)
}

// CreateTopic 创建主题
func (s *KafkaStore) CreateTopic(topic string, partitions int, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", s.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}

// GetTopicInfo 获取主题信息
func (s *KafkaStore) GetTopicInfo(topic string) (*kafka.Topic, error) {
	conn, err := kafka.Dial("tcp", s.config.Brokers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to read partitions: %w", err)
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("topic %s not found", topic)
	}

	return &kafka.Topic{
		Name:       topic,
		Partitions: partitions,
	}, nil
}

// GetConsumerGroups 获取消费者组信息（segmentio/kafka-go不支持，返回未实现）
func (s *KafkaStore) GetConsumerGroups() (interface{}, error) {
	return nil, fmt.Errorf("GetConsumerGroups not implemented for segmentio/kafka-go")
}

// Close 关闭Kafka连接
func (s *KafkaStore) Close() error {
	// Kafka连接会在使用时自动管理
	return nil
}
