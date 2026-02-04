package node

import (
	"context"
	"encoding/json"
	"sync"
	"sync-node/central/service/alert"
	"sync-node/central/service/task"
	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/service/etcd"
	"sync-node/common/utils"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

// Watcher 负责监听节点状态和任务状态变化
type Watcher struct {
	etcdClient   *etcd.Client
	db           *gorm.DB
	alertService alert.AlertService
	taskService  *task.TaskService
	stopChan     chan struct{}
	wg           sync.WaitGroup

	// 在线节点统计缓存
	// key: nodeID, value: *model.NodeState
	onlineNodes sync.Map
}

func NewWatcher(etcdClient *etcd.Client, db *gorm.DB, alertService alert.AlertService, taskService *task.TaskService) *Watcher {
	return &Watcher{
		etcdClient:   etcdClient,
		db:           db,
		alertService: alertService,
		taskService:  taskService,
		stopChan:     make(chan struct{}),
	}
}

func (w *Watcher) Start() {
	w.wg.Add(2)
	// 启动节点状态监听循环
	go w.watchNodeLoop()
	// 启动任务状态监听循环
	go w.watchTaskStatusLoop()

	// 初始化：加载当前所有节点
	w.loadInitialNodes()
}

func (w *Watcher) Stop() {
	// 关闭停止信号通道，通知所有协程退出
	close(w.stopChan)
	// 等待所有协程结束
	w.wg.Wait()
}

func (w *Watcher) loadInitialNodes() {
	// 设置超时上下文，防止加载时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 从 Etcd 获取所有节点信息（前缀搜索）
	resp, err := w.etcdClient.Get(ctx, constant.EtcdNodePrefix, clientv3.WithPrefix())
	if err != nil {
		utils.GetLogger("watcher").Errorf("Failed to load initial nodes: %v", err)
		return
	}

	// 遍历并缓存节点信息
	for _, kv := range resp.Kvs {
		var node model.NodeState
		if err := json.Unmarshal(kv.Value, &node); err == nil {
			w.onlineNodes.Store(node.NodeID, &node)
			// 同步到数据库
			if err := w.db.Save(&node).Error; err != nil {
				utils.GetLogger("watcher").Errorf("Failed to save initial node to DB for %s: %v", node.NodeID, err)
			}
			utils.GetLogger("watcher").Infof("Loaded node: %s (status: %s)", node.NodeID, node.Status)
		}
	}
}

func (w *Watcher) watchNodeLoop() {
	defer w.wg.Done()

	// 监听节点前缀的变化事件
	watchChan := w.etcdClient.Watch(context.Background(), constant.EtcdNodePrefix, clientv3.WithPrefix())
	utils.GetLogger("watcher").Info("Started watching node changes...")

	for {
		select {
		case <-w.stopChan:
			// 收到停止信号，退出循环
			return
		case resp, ok := <-watchChan:
			if !ok {
				utils.GetLogger("watcher").Error("Node watch channel closed unexpectedly")
				return
			}
			// 处理所有变动事件
			for _, ev := range resp.Events {
				w.handleNodeEvent(ev)
			}
		}
	}
}

func (w *Watcher) watchTaskStatusLoop() {
	defer w.wg.Done()

	// 监听任务状态上报前缀
	watchChan := w.etcdClient.Watch(context.Background(), constant.EtcdTaskStatusPrefix, clientv3.WithPrefix())
	utils.GetLogger("watcher").Info("Started watching task status changes...")

	for {
		select {
		case <-w.stopChan:
			// 收到停止信号，退出循环
			return
		case resp, ok := <-watchChan:
			if !ok {
				utils.GetLogger("watcher").Error("Task status watch channel closed unexpectedly")
				return
			}
			// 处理所有任务状态变动事件
			for _, ev := range resp.Events {
				w.handleTaskStatusEvent(ev)
			}
		}
	}
}

func (w *Watcher) handleNodeEvent(ev *clientv3.Event) {
	// 从 Key 中解析 NodeID
	nodeID := string(ev.Kv.Key)[len(constant.EtcdNodePrefix):]

	switch ev.Type {
	case clientv3.EventTypePut:
		// 节点上线或心跳更新
		var node model.NodeState
		if err := json.Unmarshal(ev.Kv.Value, &node); err != nil {
			utils.GetLogger("watcher").Errorf("Failed to unmarshal node state for %s: %v", nodeID, err)
			return
		}

		// 更新缓存
		w.onlineNodes.Store(nodeID, &node)

		// 同步到数据库 (Upsert)
		if err := w.db.Save(&node).Error; err != nil {
			utils.GetLogger("watcher").Errorf("Failed to save node state to DB for %s: %v", nodeID, err)
		}

		utils.GetLogger("watcher").Infof("Node updated: %s (status: %s, heartbeat: %d)", nodeID, node.Status, node.LastHeartbeat)

	case clientv3.EventTypeDelete:
		// 节点被删除（可能是过期或主动注销）
		w.onlineNodes.Delete(nodeID)
		utils.GetLogger("watcher").Warnf("Node removed from etcd (offline): %s", nodeID)

		// 更新数据库状态为 offline，而不是删除
		if err := w.db.Model(&model.NodeState{}).Where("node_id = ?", nodeID).Update("status", constant.NodeStatusOffline).Error; err != nil {
			utils.GetLogger("watcher").Errorf("Failed to update node status to offline in DB for %s: %v", nodeID, err)
		}

		// 触发报警：节点被移除/过期
		w.sendNodeAlert(nodeID, "NodeRemoved/Expired")
	}
}

// SubTaskStatusReport 定义 Agent 上报的状态结构
type SubTaskStatusReport struct {
	ID         string `json:"id"` // SubTask ID
	NodeID     string `json:"nodeId"`
	Status     string `json:"status"`
	ErrorMsg   string `json:"errorMsg"`
	SyncedSize int64  `json:"syncedSize"`
}

func (w *Watcher) handleTaskStatusEvent(ev *clientv3.Event) {
	// 只关注 PUT 事件（状态更新）
	if ev.Type != clientv3.EventTypePut {
		return
	}

	var report SubTaskStatusReport
	if err := json.Unmarshal(ev.Kv.Value, &report); err != nil {
		utils.GetLogger("watcher").Errorf("Failed to unmarshal task status report: %v", err)
		return
	}

	// 补充 ID (从 Key 中解析)
	if report.ID == "" {
		report.ID = string(ev.Kv.Key)[len(constant.EtcdTaskStatusPrefix):]
	}

	// 调用 TaskService.UpdateTaskStatus 更新到数据库并处理后续逻辑（如报警）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.taskService.UpdateTaskStatus(ctx, report.ID, report.Status, report.ErrorMsg, report.SyncedSize); err != nil {
		utils.GetLogger("watcher").Errorf("Failed to update task status for task %s on node %s: %v",
			report.ID, report.NodeID, err)
	} else {
		utils.GetLogger("watcher").Infof("Successfully updated task %s status to %s on node %s via Watcher",
			report.ID, report.Status, report.NodeID)
	}
}

func (w *Watcher) sendNodeAlert(nodeID, reason string) {
	param := map[string]string{
		"node":   nodeID,
		"reason": reason,
		"time":   time.Now().Format("15:04:05"),
	}
	// 异步发送报警
	go func() {
		if err := w.alertService.SendAlert(param); err != nil {
			utils.GetLogger("watcher").Errorf("Failed to send alert for node %s: %v", nodeID, err)
		}
	}()
}

// GetOnlineNodeCount 获取当前在线节点数量
func (w *Watcher) GetOnlineNodeCount() int {
	count := 0
	w.onlineNodes.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
