-- IM系统数据库初始化脚本

-- 创建数据库
CREATE DATABASE IF NOT EXISTS im_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE im_db;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    nickname VARCHAR(100),
    avatar VARCHAR(255),
    status ENUM('online', 'offline', 'away') DEFAULT 'offline',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_username (username),
    INDEX idx_status (status)
);

-- 消息表
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT PRIMARY KEY,
    from_user_id BIGINT NOT NULL,
    to_user_id BIGINT,
    group_id BIGINT,
    content TEXT NOT NULL,
    msg_type ENUM('text', 'image', 'file', 'voice') DEFAULT 'text',
    status ENUM('sent', 'delivered', 'read') DEFAULT 'sent',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_from_user (from_user_id),
    INDEX idx_to_user (to_user_id),
    INDEX idx_group (group_id),
    INDEX idx_created_at (created_at)
);

-- 群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id BIGINT NOT NULL,
    avatar VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_owner (owner_id)
);

-- 群组成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role ENUM('owner', 'admin', 'member') DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_group_user (group_id, user_id),
    INDEX idx_group_id (group_id),
    INDEX idx_user_id (user_id)
);

-- 插入测试数据
INSERT IGNORE INTO users (id, username, nickname, status) VALUES
(1, 'user1', '用户1', 'offline'),
(2, 'user2', '用户2', 'offline'),
(3, 'user3', '用户3', 'offline');

INSERT IGNORE INTO groups (id, name, description, owner_id) VALUES
(1, '测试群组1', '这是一个测试群组', 1),
(2, '测试群组2', '另一个测试群组', 2);

INSERT IGNORE INTO group_members (group_id, user_id, role) VALUES
(1, 1, 'owner'),
(1, 2, 'member'),
(1, 3, 'member'),
(2, 2, 'owner'),
(2, 1, 'member'); 