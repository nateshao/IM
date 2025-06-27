package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/im/internal/model"
)

type BenchmarkClient struct {
	conn   *websocket.Conn
	userID string
	done   chan struct{}
	mu     sync.Mutex
	stats  *ClientStats
}

type ClientStats struct {
	MessagesSent     int64
	MessagesReceived int64
	Errors           int64
	StartTime        time.Time
}

func NewBenchmarkClient(serverURL, userID string) (*BenchmarkClient, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return &BenchmarkClient{
		conn:   conn,
		userID: userID,
		done:   make(chan struct{}),
		stats: &ClientStats{
			StartTime: time.Now(),
		},
	}, nil
}

func (c *BenchmarkClient) Login() error {
	loginMsg := model.WebSocketMessage{
		Type: "login",
		Data: map[string]interface{}{
			"user_id":  c.userID,
			"token":    "benchmark_token",
			"platform": "benchmark",
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(loginMsg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *BenchmarkClient) SendMessage(receiverID, content string) error {
	msg := model.WebSocketMessage{
		Type: "send_message",
		Data: model.SendMessageRequest{
			ReceiverID: receiverID,
			Type:       model.MessageTypeText,
			Content:    content,
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = c.conn.WriteMessage(websocket.TextMessage, data)
	if err == nil {
		c.mu.Lock()
		c.stats.MessagesSent++
		c.mu.Unlock()
	}
	return err
}

func (c *BenchmarkClient) ReadMessages() {
	defer close(c.done)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Client %s read error: %v", c.userID, err)
			return
		}

		var wsMsg model.WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.mu.Lock()
			c.stats.Errors++
			c.mu.Unlock()
			continue
		}

		c.mu.Lock()
		c.stats.MessagesReceived++
		c.mu.Unlock()
	}
}

func (c *BenchmarkClient) StartHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			heartbeat := model.WebSocketMessage{
				Type: "heartbeat",
				Data: model.HeartbeatRequest{
					UserID: c.userID,
				},
				Timestamp: time.Now().Unix(),
			}

			data, _ := json.Marshal(heartbeat)
			c.conn.WriteMessage(websocket.TextMessage, data)
		case <-c.done:
			return
		}
	}
}

func (c *BenchmarkClient) Close() error {
	return c.conn.Close()
}

func (c *BenchmarkClient) GetStats() *ClientStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return &ClientStats{
		MessagesSent:     c.stats.MessagesSent,
		MessagesReceived: c.stats.MessagesReceived,
		Errors:           c.stats.Errors,
		StartTime:        c.stats.StartTime,
	}
}

type BenchmarkResult struct {
	TotalClients      int
	TotalMessagesSent int64
	TotalMessagesRecv int64
	TotalErrors       int64
	Duration          time.Duration
	MessagesPerSecond float64
	ConnectionsPerSec float64
}

func runBenchmark(serverURL string, numClients int, duration time.Duration, messageInterval time.Duration) (*BenchmarkResult, error) {
	fmt.Printf("Starting benchmark with %d clients for %v\n", numClients, duration)

	clients := make([]*BenchmarkClient, numClients)
	var wg sync.WaitGroup

	// 创建并连接所有客户端
	for i := 0; i < numClients; i++ {
		userID := fmt.Sprintf("benchmark_user_%d", i)
		client, err := NewBenchmarkClient(serverURL, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to create client %d: %w", i, err)
		}

		if err := client.Login(); err != nil {
			return nil, fmt.Errorf("failed to login client %d: %w", i, err)
		}

		clients[i] = client
		wg.Add(1)

		// 启动消息读取协程
		go func(c *BenchmarkClient) {
			defer wg.Done()
			c.ReadMessages()
		}(client)

		// 启动心跳协程
		go client.StartHeartbeat()
	}

	fmt.Printf("All %d clients connected successfully\n", numClients)

	// 启动消息发送协程
	stopSending := make(chan struct{})
	go func() {
		ticker := time.NewTicker(messageInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 随机发送消息
				for i, client := range clients {
					receiverID := fmt.Sprintf("benchmark_user_%d", (i+1)%numClients)
					content := fmt.Sprintf("Benchmark message from %s at %v", client.userID, time.Now())

					if err := client.SendMessage(receiverID, content); err != nil {
						log.Printf("Failed to send message from client %s: %v", client.userID, err)
					}
				}
			case <-stopSending:
				return
			}
		}
	}()

	// 等待测试时间
	time.Sleep(duration)
	close(stopSending)

	// 关闭所有客户端
	for _, client := range clients {
		client.Close()
	}

	// 等待所有协程结束
	wg.Wait()

	// 收集统计信息
	var result BenchmarkResult
	result.TotalClients = numClients
	result.Duration = duration

	for _, client := range clients {
		stats := client.GetStats()
		result.TotalMessagesSent += stats.MessagesSent
		result.TotalMessagesRecv += stats.MessagesReceived
		result.TotalErrors += stats.Errors
	}

	result.MessagesPerSecond = float64(result.TotalMessagesSent) / duration.Seconds()
	result.ConnectionsPerSec = float64(numClients) / duration.Seconds()

	return &result, nil
}

func main() {
	serverURL := "ws://localhost:8080/ws"

	// 测试配置
	testCases := []struct {
		name            string
		numClients      int
		duration        time.Duration
		messageInterval time.Duration
	}{
		{
			name:            "Small Load Test",
			numClients:      10,
			duration:        30 * time.Second,
			messageInterval: 1 * time.Second,
		},
		{
			name:            "Medium Load Test",
			numClients:      100,
			duration:        60 * time.Second,
			messageInterval: 2 * time.Second,
		},
		{
			name:            "High Load Test",
			numClients:      1000,
			duration:        120 * time.Second,
			messageInterval: 5 * time.Second,
		},
	}

	fmt.Println("WebSocket Performance Benchmark")
	fmt.Println("================================")

	for _, testCase := range testCases {
		fmt.Printf("\nRunning %s:\n", testCase.name)
		fmt.Printf("- Clients: %d\n", testCase.numClients)
		fmt.Printf("- Duration: %v\n", testCase.duration)
		fmt.Printf("- Message Interval: %v\n", testCase.messageInterval)

		result, err := runBenchmark(serverURL, testCase.numClients, testCase.duration, testCase.messageInterval)
		if err != nil {
			log.Printf("Benchmark failed: %v", err)
			continue
		}

		fmt.Printf("\nResults:\n")
		fmt.Printf("- Total Messages Sent: %d\n", result.TotalMessagesSent)
		fmt.Printf("- Total Messages Received: %d\n", result.TotalMessagesRecv)
		fmt.Printf("- Total Errors: %d\n", result.TotalErrors)
		fmt.Printf("- Messages per second: %.2f\n", result.MessagesPerSecond)
		fmt.Printf("- Connections per second: %.2f\n", result.ConnectionsPerSec)
		fmt.Printf("- Success Rate: %.2f%%\n",
			float64(result.TotalMessagesRecv)/float64(result.TotalMessagesSent)*100)
	}

	fmt.Println("\nBenchmark completed!")
}
