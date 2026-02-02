package constant

const (
	// etcd Key值
	EtcdRootPrefix   = "/file_sync/"
	EtcdNodePrefix   = EtcdRootPrefix + "node/"
	EtcdTaskPrefix   = EtcdRootPrefix + "task/"
	EtcdConfigPrefix = EtcdRootPrefix + "config/"

	// 节点状态
	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"

	// 任务状态
	TaskStatusPending     = "pending"
	TaskStatusDownloading = "downloading"
	TaskStatusCompleted   = "completed"
	TaskStatusFailed      = "failed"

	// 默认配置
	DefaultHeartbeatInterval = 15 // seconds (心跳间隔)
	DefaultLeaseTTL          = 45 // seconds (租约TTL，通常比心跳间隔大，避免临界值过期)

	// 错误码 (API Response Code)
	CodeSuccess      = 200
	CodeInvalidParam = 400
	CodeUnauthorized = 401
	CodeNotFound     = 404
	CodeServerBusy   = 503
	CodeInternalErr  = 500

	// 业务错误码 (1000+)
	CodeTaskNotFound   = 1001
	CodeNodeNotFound   = 1002
	CodeEtcdError      = 1003
	CodeDBError        = 1004
	CodeOSSError       = 1005
	CodeTaskDuplicated = 1006
)
