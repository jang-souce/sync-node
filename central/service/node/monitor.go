package node

import (
	"context"
	"sync"
	"time"

	"sync-node/central/service/alert"
	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/utils"
)

// NodeProvider 定义 Monitor 需要的节点服务接口
type NodeProvider interface {
	GetNodes(ctx context.Context) ([]*model.NodeState, error)
}

// Monitor 负责节点健康检查
type Monitor struct {
	nodeService  NodeProvider
	alertService alert.AlertService
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

func NewMonitor(nodeService NodeProvider, alertService alert.AlertService) *Monitor {
	return &Monitor{
		nodeService:  nodeService,
		alertService: alertService,
		stopChan:     make(chan struct{}),
	}
}

func (m *Monitor) Start() {
	m.wg.Add(1)
	// 启动健康检查循环
	go m.checkLoop()
}

func (m *Monitor) Stop() {
	// 关闭停止信号，通知检查循环退出
	close(m.stopChan)
	// 等待检查循环结束
	m.wg.Wait()
}

func (m *Monitor) checkLoop() {
	defer m.wg.Done()

	// 检查间隔，建议配置化，这里暂定 30 秒
	// 也就是每 30 秒扫描一次所有节点的心跳状态
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 记录已报警的节点，防止重复报警（简单的去重逻辑）
	// key: nodeID, value: lastAlertTime
	alertHistory := make(map[string]time.Time)

	for {
		select {
		case <-m.stopChan:
			// 收到停止信号，退出循环
			return
		case <-ticker.C:
			// 定时触发节点检查
			m.checkNodes(alertHistory)
		}
	}
}

func (m *Monitor) checkNodes(alertHistory map[string]time.Time) {
	// 设置超时，防止获取节点信息卡住
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取当前所有已注册节点
	nodes, err := m.nodeService.GetNodes(ctx)
	if err != nil {
		utils.GetLogger("monitor").Errorf("Failed to get nodes: %v", err)
		return
	}

	// 判定离线的阈值：心跳间隔的 3 倍
	// 默认心跳间隔如果是 5s，那么 15s 没有心跳就算离线
	threshold := constant.DefaultHeartbeatInterval * time.Second * 3
	// 如果没有配置心跳间隔，使用默认值
	if threshold == 0 {
		threshold = constant.DefaultHeartbeatInterval * time.Second * 3
	}

	now := time.Now()
	for _, node := range nodes {
		// 检查 LastHeartbeat
		// 注意：LastHeartbeat 是 int64 时间戳
		lastHeartbeat := time.Unix(node.LastHeartbeat, 0)
		if now.Sub(lastHeartbeat) > threshold {
			// 节点断联
			if lastAlertTime, ok := alertHistory[node.NodeID]; ok {
				// 报警防抖：1小时内不重复报警
				if now.Sub(lastAlertTime) < 1*time.Hour {
					continue
				}
			}

			utils.GetLogger("monitor").Warnf("Node %s is offline. Last heartbeat: %v", node.NodeID, lastHeartbeat)

			// 发送报警
			param := map[string]string{
				"node":   node.NodeID,
				"reason": "Offline",
				"time":   lastHeartbeat.Format("15:04:05"),
			}
			// 节点断联报警
			if err := m.alertService.SendAlert(param); err != nil {
				utils.GetLogger("monitor").Errorf("Failed to send alert for node %s: %v", node.NodeID, err)
			} else {
				// 记录报警时间，用于后续防抖
				alertHistory[node.NodeID] = now
			}
		} else {
			// 节点恢复（心跳正常），清除报警记录
			delete(alertHistory, node.NodeID)
		}
	}
}
