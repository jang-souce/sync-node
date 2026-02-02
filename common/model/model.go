package model

import "gorm.io/gorm"

// TaskMeta 表示文件同步任务的元数据。
// 由中心节点创建并存储在 Etcd 和数据库中。
type TaskMeta struct {
	ID        string         `json:"id" gorm:"primaryKey;type:varchar(36);comment:唯一任务 ID (例如 UUID)"` // 唯一任务 ID (例如 UUID)
	FileName  string         `json:"fileName" gorm:"type:varchar(255);not null;comment:文件名"`
	FileSize  int64          `json:"fileSize" gorm:"not null;comment:文件大小"`
	FileHash  string         `json:"fileHash" gorm:"type:varchar(64);not null;comment:文件哈希"`
	OssURL    string         `json:"ossUrl" gorm:"type:text;not null;comment:OSS下载链接"`
	SourceURL string         `json:"sourceUrl" gorm:"type:text;comment:原始文件来源URL"`                    // 原始文件来源 URL
	Tag       string         `json:"tag" gorm:"type:varchar(255);comment:标签"`                         // 标签
	TaskType  int            `json:"taskType" gorm:"default:1;comment:任务类型 1:只下载 2:下载后同步删除OSS"`       // 任务类型 1:只下载 2:下载后同步删除OSS
	Status    string         `json:"status" gorm:"type:varchar(20);default:'PENDING';comment:任务整体状态"` // 任务整体状态
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index;comment:软删除时间"`                            // 软删除时间
	CreatedAt int64          `json:"createdAt" gorm:"autoCreateTime;comment:创建时间戳(Unix)"`             // 创建时间戳 (Unix)
}

// TaskState 表示特定节点上的任务状态。
// 由子节点更新以向 Etcd 报告进度。
type TaskState struct {
	ID         uint   `json:"-" gorm:"primaryKey;comment:内部数据库 ID"` // 内部数据库 ID
	TaskID     string `json:"taskId" gorm:"type:varchar(36);index;not null;comment:任务ID"`
	NodeID     string `json:"nodeId" gorm:"type:varchar(36);index;not null;comment:节点ID"`
	Status     string `json:"status" gorm:"type:varchar(20);not null;comment:状态"`
	ErrorMsg   string `json:"errorMsg" gorm:"type:text;comment:错误信息"`
	RetryCount int    `json:"retryCount" gorm:"default:0;comment:重试次数"`              // 重试次数
	SyncedSize int64  `json:"syncedSize" gorm:"default:0;comment:已同步字节数"`            // 已同步字节数
	UpdatedAt  int64  `json:"updatedAt" gorm:"autoUpdateTime;comment:最后更新时间戳(Unix)"` // 最后更新时间戳 (Unix)
}

// NodeState 表示子节点的运行时状态。
// 由子节点通过心跳在 Etcd 中维护。
type NodeState struct {
	NodeID        string  `json:"nodeId" gorm:"primaryKey;type:varchar(36);comment:节点ID"`
	Hostname      string  `json:"hostname" gorm:"type:varchar(255);comment:主机名"`
	IP            string  `json:"ip" gorm:"type:varchar(50);comment:IP地址"`
	Status        string  `json:"status" gorm:"type:varchar(20);comment:状态"`
	DiskUsage     float64 `json:"diskUsage" gorm:"type:decimal(5,2);comment:磁盘使用率百分比"` // 磁盘使用率百分比
	TotalDisk     int64   `json:"totalDisk" gorm:"comment:总磁盘空间(字节)"`                  // 总磁盘空间 (字节)
	AvailableDisk int64   `json:"availableDisk" gorm:"comment:可用磁盘空间(字节)"`             // 可用磁盘空间 (字节)
	LastHeartbeat int64   `json:"lastHeartbeat" gorm:"index;comment:最后心跳时间戳(Unix)"`    // 最后心跳时间戳 (Unix)
}

// GlobalConfig 表示存储在 Etcd 中的系统全局配置。
type GlobalConfig struct {
	ID                uint  `json:"-" gorm:"primaryKey;comment:主键ID"`
	HeartbeatInterval int   `json:"heartbeatInterval" gorm:"comment:心跳间隔(秒)"` // 心跳间隔 (秒)
	TaskTimeout       int   `json:"taskTimeout" gorm:"comment:任务超时时间(秒)"`     // 任务超时时间 (秒)
	UpdatedAt         int64 `json:"updatedAt" gorm:"autoUpdateTime;comment:更新时间"`
}
