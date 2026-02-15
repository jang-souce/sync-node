package model

import "gorm.io/gorm"

// MainTask 主任务表，记录一次批量任务提交的整体信息
type MainTask struct {
	ID         string         `json:"id" gorm:"primaryKey;type:varchar(36);comment:主任务ID"`
	TotalCount int            `json:"totalCount" gorm:"not null;comment:子任务总数"`
	Status     string         `json:"status" gorm:"type:varchar(20);default:'PENDING';comment:总体状态"`
	TotalSize  int64          `json:"totalSize" gorm:"default:0;comment:总文件大小(字节)"`
	SyncedSize int64          `json:"syncedSize" gorm:"default:0;comment:已同步总大小(字节)"`
	CreatedAt  int64          `json:"createdAt" gorm:"autoCreateTime;comment:创建时间戳(Unix)"`
	UpdatedAt  int64          `json:"updatedAt" gorm:"autoUpdateTime;comment:更新时间戳(Unix)"`
	DeletedAt  gorm.DeletedAt `json:"deletedAt" gorm:"index;comment:软删除时间"`
	SubTasks   []SubTask      `json:"subTasks" gorm:"foreignKey:MainTaskID"` // 关联子任务
}

// SubTask 子任务表 (原 TaskMeta)，具体的文件分发任务，一对一绑定节点
type SubTask struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(36);comment:子任务ID"`
	MainTaskID string `json:"mainTaskId" gorm:"type:varchar(36);index;not null;comment:关联主任务ID"`
	NodeID     string `json:"nodeId" gorm:"type:varchar(36);index;not null;comment:目标节点ID"`

	// 文件信息
	FileName  string `json:"fileName" binding:"required" gorm:"type:varchar(255);not null;comment:文件名"`
	FileSize  int64  `json:"fileSize" gorm:"not null;comment:文件大小"`
	FileHash  string `json:"fileHash" gorm:"type:varchar(64);not null;comment:文件哈希"`
	OssURL    string `json:"ossUrl" gorm:"type:text;not null;comment:OSS下载链接"`
	OssKey    string `json:"ossKey" gorm:"type:varchar(255);comment:OSS对象Key"`
	SourceURL string `json:"sourceUrl" binding:"required,url" gorm:"type:text;comment:原始文件来源URL"`
	Tag       string `json:"tag" gorm:"type:varchar(255);comment:标签"`
	TaskType  int    `json:"taskType" binding:"oneof=1 2" gorm:"default:1;comment:任务类型 1:只下载 2:下载后同步删除OSS"`

	// 任务状态 (原 TaskState 内容)
	Status     string         `json:"status" gorm:"type:varchar(20);default:'PENDING';comment:状态"`
	ErrorMsg   string         `json:"errorMsg" gorm:"type:text;comment:错误信息"`
	RetryCount int            `json:"retryCount" gorm:"default:0;comment:重试次数"`
	SyncedSize int64          `json:"syncedSize" gorm:"default:0;comment:已同步字节数"`
	UpdatedAt  int64          `json:"updatedAt" gorm:"autoUpdateTime;comment:最后更新时间戳(Unix)"`
	CreatedAt  int64          `json:"createdAt" gorm:"autoCreateTime;comment:创建时间戳(Unix)"`
	DeletedAt  gorm.DeletedAt `json:"deletedAt" gorm:"index;comment:软删除时间"`
}

// NodeState 表示子节点的运行时状态。
// 由子节点通过心跳在 Etcd 中维护。
type NodeState struct {
	NodeID        string  `json:"nodeId" gorm:"primaryKey;type:varchar(36);comment:节点ID"`
	Hostname      string  `json:"hostname" gorm:"type:varchar(255);comment:主机名"`
	IP            string  `json:"ip" gorm:"type:varchar(50);comment:IP地址"`
	Port          string  `json:"port" gorm:"type:varchar(10);comment:端口"`
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
