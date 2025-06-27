# IM系统 API 文档

## 概述

IM系统提供WebSocket和HTTP REST API两种接口方式。

- **WebSocket**: 用于实时消息推送和双向通信
- **HTTP REST API**: 用于消息管理、群组管理等操作

## 基础信息

- **Base URL**: `http://localhost:8080`
- **WebSocket URL**: `ws://localhost:8080/ws`
- **API Version**: `v1`
- **Content-Type**: `application/json`

## 认证

目前使用简单的用户ID头部认证：

```
X-User-ID: your_user_id
```

## WebSocket API

### 连接

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

### 消息格式

所有WebSocket消息都使用以下JSON格式：

```json
{
  "type": "message_type",
  "data": {},
  "timestamp": 1640995200000,
  "message_id": "optional_message_id"
}
```

### 消息类型

#### 1. 登录 (login)

**请求:**
```json
{
  "type": "login",
  "data": {
    "user_id": "user123",
    "token": "auth_token",
    "platform": "web"
  },
  "timestamp": 1640995200000
}
```

**响应:**
```json
{
  "type": "login",
  "data": {
    "success": true,
    "message": "Login successful",
    "user_id": "user123"
  },
  "timestamp": 1640995200000
}
```

#### 2. 心跳 (heartbeat)

**请求:**
```json
{
  "type": "heartbeat",
  "data": {
    "user_id": "user123"
  },
  "timestamp": 1640995200000
}
```

**响应:**
```json
{
  "type": "heartbeat",
  "data": {
    "timestamp": 1640995200000
  },
  "timestamp": 1640995200000
}
```

#### 3. 发送消息 (send_message)

**请求:**
```json
{
  "type": "send_message",
  "data": {
    "receiver_id": "user456",
    "group_id": "optional_group_id",
    "type": "text",
    "content": "Hello, world!"
  },
  "timestamp": 1640995200000
}
```

**响应:**
```json
{
  "type": "send_message",
  "data": {
    "success": true,
    "message_id": "msg_123456",
    "message": {
      "id": "msg_123456",
      "sender_id": "user123",
      "receiver_id": "user456",
      "type": "text",
      "content": "Hello, world!",
      "status": "sent",
      "timestamp": 1640995200000
    }
  },
  "timestamp": 1640995200000
}
```

#### 4. 消息确认 (ack)

**请求:**
```json
{
  "type": "ack",
  "data": {
    "message_id": "msg_123456",
    "status": "read"
  },
  "timestamp": 1640995200000
}
```

#### 5. 同步离线消息 (sync_offline)

**请求:**
```json
{
  "type": "sync_offline",
  "data": {
    "last_message_id": "msg_123456",
    "limit": 50
  },
  "timestamp": 1640995200000
}
```

**响应:**
```json
{
  "type": "sync_offline",
  "data": {
    "messages": [
      {
        "id": "msg_123457",
        "sender_id": "user456",
        "receiver_id": "user123",
        "type": "text",
        "content": "Hi there!",
        "status": "sent",
        "timestamp": 1640995200000
      }
    ],
    "has_more": false
  },
  "timestamp": 1640995200000
}
```

#### 6. 加入群聊 (join_group)

**请求:**
```json
{
  "type": "join_group",
  "data": {
    "group_id": "group123"
  },
  "timestamp": 1640995200000
}
```

#### 7. 离开群聊 (leave_group)

**请求:**
```json
{
  "type": "leave_group",
  "data": {
    "group_id": "group123"
  },
  "timestamp": 1640995200000
}
```

### 推送消息

#### 新消息推送 (new_message)

```json
{
  "type": "new_message",
  "data": {
    "id": "msg_123456",
    "sender_id": "user456",
    "receiver_id": "user123",
    "type": "text",
    "content": "Hello!",
    "status": "sent",
    "timestamp": 1640995200000
  },
  "timestamp": 1640995200000,
  "message_id": "msg_123456"
}
```

#### 群聊消息推送 (new_group_message)

```json
{
  "type": "new_group_message",
  "data": {
    "id": "msg_123456",
    "sender_id": "user456",
    "group_id": "group123",
    "type": "text",
    "content": "Hello everyone!",
    "status": "sent",
    "timestamp": 1640995200000
  },
  "timestamp": 1640995200000,
  "message_id": "msg_123456"
}
```

## HTTP REST API

### 健康检查

#### GET /health

检查服务健康状态。

**响应:**
```json
{
  "status": "ok",
  "timestamp": 1640995200000
}
```

### 消息管理

#### POST /api/v1/messages

发送消息。

**请求头:**
```
X-User-ID: user123
Content-Type: application/json
```

**请求体:**
```json
{
  "receiver_id": "user456",
  "group_id": "optional_group_id",
  "type": "text",
  "content": "Hello, world!"
}
```

**响应:**
```json
{
  "success": true,
  "message": {
    "id": "msg_123456",
    "sender_id": "user123",
    "receiver_id": "user456",
    "type": "text",
    "content": "Hello, world!",
    "status": "sent",
    "timestamp": 1640995200000
  },
  "message_id": "msg_123456"
}
```

#### GET /api/v1/messages/:messageID

获取指定消息。

**请求头:**
```
X-User-ID: user123
```

**响应:**
```json
{
  "message": {
    "id": "msg_123456",
    "sender_id": "user123",
    "receiver_id": "user456",
    "type": "text",
    "content": "Hello, world!",
    "status": "sent",
    "timestamp": 1640995200000
  }
}
```

#### POST /api/v1/messages/:messageID/ack

确认消息状态。

**请求头:**
```
X-User-ID: user123
Content-Type: application/json
```

**请求体:**
```json
{
  "status": "read"
}
```

**响应:**
```json
{
  "success": true
}
```

#### GET /api/v1/messages/offline

同步离线消息。

**请求头:**
```
X-User-ID: user123
```

**查询参数:**
- `last_message_id` (可选): 最后消息ID
- `limit` (可选): 限制数量，默认50

**响应:**
```json
{
  "messages": [
    {
      "id": "msg_123456",
      "sender_id": "user456",
      "receiver_id": "user123",
      "type": "text",
      "content": "Hi there!",
      "status": "sent",
      "timestamp": 1640995200000
    }
  ],
  "has_more": false
}
```

### 群组管理

#### POST /api/v1/groups

创建群组。

**请求头:**
```
X-User-ID: user123
Content-Type: application/json
```

**请求体:**
```json
{
  "name": "My Group",
  "description": "A test group",
  "members": ["user456", "user789"]
}
```

**响应:**
```json
{
  "group": {
    "id": "group123",
    "name": "My Group",
    "description": "A test group",
    "owner_id": "user123",
    "members": ["user456", "user789"],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### GET /api/v1/groups/:groupID

获取群组信息。

**请求头:**
```
X-User-ID: user123
```

**响应:**
```json
{
  "group": {
    "id": "group123",
    "name": "My Group",
    "description": "A test group",
    "owner_id": "user123",
    "members": ["user456", "user789"],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### GET /api/v1/groups/:groupID/members

获取群组成员。

**请求头:**
```
X-User-ID: user123
```

**响应:**
```json
{
  "members": [
    {
      "id": "member123",
      "group_id": "group123",
      "user_id": "user456",
      "role": "member",
      "joined_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### POST /api/v1/groups/:groupID/join

加入群组。

**请求头:**
```
X-User-ID: user123
```

**响应:**
```json
{
  "success": true
}
```

#### POST /api/v1/groups/:groupID/leave

离开群组。

**请求头:**
```
X-User-ID: user123
```

**响应:**
```json
{
  "success": true
}
```

### 统计信息

#### GET /api/v1/stats

获取系统统计信息。

**响应:**
```json
{
  "connections": 150,
  "online_users": 120,
  "timestamp": 1640995200000
}
```

## 错误处理

### 错误响应格式

```json
{
  "error": "Error message description"
}
```

### 常见错误码

- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: 未认证
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误

## 消息类型

支持的消息类型：

- `text`: 文本消息
- `image`: 图片消息
- `file`: 文件消息
- `voice`: 语音消息
- `video`: 视频消息
- `system`: 系统消息

## 消息状态

消息状态流转：

- `sent`: 已发送
- `delivered`: 已投递
- `read`: 已读
- `failed`: 发送失败

## 示例代码

### JavaScript WebSocket客户端

```javascript
class IMClient {
  constructor(url, userId) {
    this.url = url;
    this.userId = userId;
    this.ws = null;
  }

  connect() {
    this.ws = new WebSocket(this.url);
    
    this.ws.onopen = () => {
      console.log('Connected to IM server');
      this.login();
    };
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };
    
    this.ws.onclose = () => {
      console.log('Disconnected from IM server');
    };
  }

  login() {
    const loginMsg = {
      type: 'login',
      data: {
        user_id: this.userId,
        token: 'test_token',
        platform: 'web'
      },
      timestamp: Date.now()
    };
    this.ws.send(JSON.stringify(loginMsg));
  }

  sendMessage(receiverId, content) {
    const msg = {
      type: 'send_message',
      data: {
        receiver_id: receiverId,
        type: 'text',
        content: content
      },
      timestamp: Date.now()
    };
    this.ws.send(JSON.stringify(msg));
  }

  handleMessage(message) {
    switch (message.type) {
      case 'new_message':
        console.log('New message:', message.data);
        break;
      case 'new_group_message':
        console.log('New group message:', message.data);
        break;
      default:
        console.log('Received message:', message);
    }
  }
}

// 使用示例
const client = new IMClient('ws://localhost:8080/ws', 'user123');
client.connect();
```

### Python HTTP客户端

```python
import requests
import json

class IMClient:
    def __init__(self, base_url, user_id):
        self.base_url = base_url
        self.user_id = user_id
        self.headers = {'X-User-ID': user_id}

    def send_message(self, receiver_id, content, group_id=None):
        url = f"{self.base_url}/api/v1/messages"
        data = {
            'receiver_id': receiver_id,
            'type': 'text',
            'content': content
        }
        if group_id:
            data['group_id'] = group_id
        
        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

    def get_offline_messages(self, last_message_id=None, limit=50):
        url = f"{self.base_url}/api/v1/messages/offline"
        params = {'limit': limit}
        if last_message_id:
            params['last_message_id'] = last_message_id
        
        response = requests.get(url, params=params, headers=self.headers)
        return response.json()

    def create_group(self, name, description, members):
        url = f"{self.base_url}/api/v1/groups"
        data = {
            'name': name,
            'description': description,
            'members': members
        }
        
        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

# 使用示例
client = IMClient('http://localhost:8080', 'user123')
result = client.send_message('user456', 'Hello!')
print(result)
``` 