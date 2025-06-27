package snowflake

import (
	"sync"
	"time"

	"github.com/sony/sonyflake"
)

var (
	sf     *sonyflake.Sonyflake
	once   sync.Once
	nodeID uint16
)

// Init 初始化Snowflake生成器
func Init(machineID uint16) {
	once.Do(func() {
		nodeID = machineID
		st := sonyflake.Settings{
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			MachineID: func() (uint16, error) {
				return nodeID, nil
			},
		}
		sf = sonyflake.NewSonyflake(st)
	})
}

// GenerateID 生成唯一ID
func GenerateID() (uint64, error) {
	if sf == nil {
		Init(1) // 默认使用机器ID 1
	}
	return sf.NextID()
}

// GenerateIDString 生成字符串格式的ID
func GenerateIDString() (string, error) {
	id, err := GenerateID()
	if err != nil {
		return "", err
	}
	return string(rune(id)), nil
}

// ParseID 解析ID获取时间戳
func ParseID(id uint64) time.Time {
	// Sonyflake的ID结构：39位时间戳 + 8位序列号 + 16位机器ID
	timestamp := id >> 23 // 右移23位获取时间戳
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return startTime.Add(time.Duration(timestamp) * 10 * time.Millisecond)
}

// GetMachineID 获取机器ID
func GetMachineID() uint16 {
	return nodeID
}
