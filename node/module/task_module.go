package module

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/utils"
	"sync-node/node/config"
	"sync-node/node/service"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TaskModule 任务模块，负责监听和执行任务
type TaskModule struct {
	etcdService *service.EtcdService
	httpService *service.HttpService
	nodeID      string
	workDir     string
	stopChan    chan struct{}
	wg          sync.WaitGroup
	logger      *logrus.Entry
}

// NewTaskModule 创建任务模块实例
func NewTaskModule(etcdService *service.EtcdService, httpService *service.HttpService, nodeID string, workDir string) *TaskModule {
	return &TaskModule{
		etcdService: etcdService,
		httpService: httpService,
		nodeID:      nodeID,
		workDir:     workDir,
		stopChan:    make(chan struct{}),
		logger:      utils.GetLogger("task"),
	}
}

// Start 启动任务模块
func (m *TaskModule) Start() {
	m.wg.Add(1)
	go m.watchTasks()
}

// Stop 停止任务模块
func (m *TaskModule) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

// watchTasks 监听任务变更
func (m *TaskModule) watchTasks() {
	defer m.wg.Done()

	m.logger.Infof("Starting to watch tasks for node: %s", m.nodeID)

	// 1. 先处理遗留的 Pending 任务
	ctx := context.Background()
	pendingTasks, err := m.etcdService.GetPendingTasks(ctx, m.nodeID)
	if err != nil {
		m.logger.Errorf("Failed to get pending tasks: %v", err)
	} else {
		for _, task := range pendingTasks {
			m.logger.Infof("Processing pending task found on startup: %s", task.ID)
			m.wg.Add(1)
			go func(t model.SubTask) {
				defer m.wg.Done()
				m.processTask(t)
			}(task)
		}
	}

	watchChan := m.etcdService.WatchTasks(context.Background(), m.nodeID)

	for {
		select {
		case <-m.stopChan:
			return
		case resp, ok := <-watchChan:
			if !ok {
				m.logger.Warn("Watch channel closed")
				return
			}
			for _, ev := range resp.Events {
				if ev.Type == clientv3.EventTypePut {
					m.handleTaskPut(ev.Kv.Value)
				}
			}
		}
	}
}

// handleTaskPut 处理任务新增或更新事件
func (m *TaskModule) handleTaskPut(value []byte) {
	var task model.SubTask
	if err := json.Unmarshal(value, &task); err != nil {
		m.logger.Errorf("Failed to unmarshal task: %v", err)
		return
	}

	// 过滤已完成或失败的任务 (如果是重新读取旧 key)
	// 理想情况下只处理 PENDING 状态的任务
	if task.Status != constant.TaskStatusPending {
		// 记录日志或忽略
		// 简单起见，仅处理 PENDING 任务
		return
	}

	m.logger.Infof("Received task: %s, file: %s", task.ID, task.FileName)

	// 启动协程处理任务
	m.wg.Add(1)
	go func(t model.SubTask) {
		defer m.wg.Done()
		m.processTask(t)
	}(task)
}

// processTask 处理单个任务
func (m *TaskModule) processTask(task model.SubTask) {
	// 1. 更新状态为下载中
	m.etcdService.UpdateTaskStatus(context.Background(), task.ID, constant.TaskStatusDownloading, "", nil)

	// 2. 确定目标文件路径
	destPath := filepath.Join(m.workDir, task.FileName)

	// 进度上报变量
	var syncedSize int64

	// 启动定时上报协程
	reportCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		var lastReportedSize int64 = -1

		for {
			select {
			case <-reportCtx.Done():
				return
			case <-ticker.C:
				current := atomic.LoadInt64(&syncedSize)
				// 只有进度变化才上报
				if current != lastReportedSize {
					extra := map[string]interface{}{
						"syncedSize": current,
					}
					// 忽略错误，仅记录调试日志
					if err := m.etcdService.UpdateTaskStatus(context.Background(), task.ID, constant.TaskStatusDownloading, "", extra); err != nil {
						m.logger.Debugf("Failed to report progress for task %s: %v", task.ID, err)
					} else {
						lastReportedSize = current
					}
				}
			}
		}
	}()

	// 3. 下载文件 (HttpService 中已包含重试逻辑) + 超时控制
	timeout := config.GlobalConfig.Node.TaskTimeout
	if timeout <= 0 {
		timeout = 300 // 默认 5 分钟
	}
	ctx, cancelTimeout := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancelTimeout()

	err := m.httpService.DownloadFile(ctx, task.OssURL, destPath, func(current int64) {
		atomic.StoreInt64(&syncedSize, current)
	})

	// 停止上报协程
	cancel()

	if err != nil {
		m.logger.Errorf("Task %s failed to download: %v", task.ID, err)
		msg := err.Error()
		if ctx.Err() == context.DeadlineExceeded {
			msg = "timeout"
		}
		// 清理临时文件
		tmpPath := destPath + ".tmp"
		_ = os.Remove(tmpPath)
		m.etcdService.UpdateTaskStatus(context.Background(), task.ID, constant.TaskStatusFailed, msg, nil)
		return
	}

	// 4. 校验 MD5（如果提供了哈希）
	if task.FileHash != "" {
		hash, err := utils.FileMD5(destPath)
		if err != nil {
			m.logger.Errorf("Task %s failed to calculate MD5: %v", task.ID, err)
			m.etcdService.UpdateTaskStatus(context.Background(), task.ID, constant.TaskStatusFailed, "MD5 check error: "+err.Error(), nil)
			return
		}
		if hash != task.FileHash {
			m.logger.Errorf("Task %s MD5 mismatch. Expected %s, got %s", task.ID, task.FileHash, hash)
			m.etcdService.UpdateTaskStatus(context.Background(), task.ID, constant.TaskStatusFailed, "MD5 mismatch", nil)
			return
		}
	}

	// 5. 更新状态为完成
	m.logger.Infof("Task %s completed successfully", task.ID)
	// 完成时上报最终大小
	finalSize := atomic.LoadInt64(&syncedSize)
	extra := map[string]interface{}{
		"syncedSize": finalSize,
	}
	// 完成后清理临时文件（如果残留）
	_ = os.Remove(destPath + ".tmp")
	m.etcdService.UpdateTaskStatus(context.Background(), task.ID, constant.TaskStatusCompleted, "", extra)
}
