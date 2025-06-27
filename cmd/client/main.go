package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/im/internal/model"
)

type Client struct {
	conn   *websocket.Conn
	userID string
	done   chan struct{}
}

func NewClient(serverURL, userID string) (*Client, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		userID: userID,
		done:   make(chan struct{}),
	}, nil
}

func (c *Client) Login() error {
	loginMsg := model.WebSocketMessage{
		Type: "login",
		Data: map[string]interface{}{
			"user_id":  c.userID,
			"token":    "test_token",
			"platform": "test",
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(loginMsg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) SendMessage(receiverID, content string) error {
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

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) SendHeartbeat() error {
	heartbeat := model.WebSocketMessage{
		Type: "heartbeat",
		Data: model.HeartbeatRequest{
			UserID: c.userID,
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) SyncOfflineMessages() error {
	syncMsg := model.WebSocketMessage{
		Type: "sync_offline",
		Data: model.SyncOfflineRequest{
			LastMessageID: "",
			Limit:         50,
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(syncMsg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) ReadMessages() {
	defer close(c.done)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		var wsMsg model.WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		log.Printf("Received [%s]: %+v", wsMsg.Type, wsMsg.Data)
	}
}

func (c *Client) StartHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.SendHeartbeat(); err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run cmd/client/main.go <server_url> <user_id>")
		fmt.Println("Example: go run cmd/client/main.go ws://localhost:8080/ws user1")
		os.Exit(1)
	}

	serverURL := os.Args[1]
	userID := os.Args[2]

	client, err := NewClient(serverURL, userID)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	// 登录
	if err := client.Login(); err != nil {
		log.Fatal("Failed to login:", err)
	}
	log.Printf("Logged in as %s", userID)

	// 启动消息读取协程
	go client.ReadMessages()

	// 启动心跳协程
	go client.StartHeartbeat()

	// 同步离线消息
	if err := client.SyncOfflineMessages(); err != nil {
		log.Printf("Failed to sync offline messages: %v", err)
	}

	// 处理用户输入
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Commands:")
		fmt.Println("  send <receiver_id> <message> - Send a message")
		fmt.Println("  sync - Sync offline messages")
		fmt.Println("  quit - Quit the client")

		for scanner.Scan() {
			text := scanner.Text()
			if text == "quit" {
				client.Close()
				return
			}

			if len(text) >= 4 && text[:4] == "send" {
				// 解析 send 命令
				var receiverID, message string
				fmt.Sscanf(text, "send %s %s", &receiverID, &message)
				if receiverID != "" && message != "" {
					if err := client.SendMessage(receiverID, message); err != nil {
						log.Printf("Failed to send message: %v", err)
					} else {
						log.Printf("Sent message to %s: %s", receiverID, message)
					}
				} else {
					fmt.Println("Usage: send <receiver_id> <message>")
				}
			} else if text == "sync" {
				if err := client.SyncOfflineMessages(); err != nil {
					log.Printf("Failed to sync offline messages: %v", err)
				}
			} else {
				fmt.Println("Unknown command. Type 'quit' to exit.")
			}
		}
	}()

	// 等待中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-interrupt:
		log.Println("Received interrupt signal")
	case <-client.done:
		log.Println("Connection closed")
	}

	// 优雅关闭
	err = client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Printf("Write close error: %v", err)
	}

	select {
	case <-client.done:
	case <-time.After(time.Second):
	}
}
