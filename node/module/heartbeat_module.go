package module

import (
	"context"
	"os"
	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/utils"
	"sync-node/node/config"
	"sync-node/node/service"
	"sync-node/node/util"
	"time"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// HeartbeatModule 心跳模块，负责节点注册和状态上报
type HeartbeatModule struct {
	etcdService *service.EtcdService
	nodeID      string
	stopChan    chan struct{}
	logger      *logrus.Entry
}

// NewHeartbeatModule 创建心跳模块实例
func NewHeartbeatModule(etcdService *service.EtcdService, nodeID string) *HeartbeatModule {
	return &HeartbeatModule{
		etcdService: etcdService,
		nodeID:      nodeID,
		stopChan:    make(chan struct{}),
		logger:      utils.GetLogger("heartbeat"),
	}
}

// Start 启动心跳模块
func (m *HeartbeatModule) Start() {
	go m.run()
}

// Stop 停止心跳模块
func (m *HeartbeatModule) Stop() {
	close(m.stopChan)
}

// run 心跳主循环
func (m *HeartbeatModule) run() {
	ticker := time.NewTicker(time.Duration(config.GlobalConfig.Node.HeartbeatInterval) * time.Second)
	if config.GlobalConfig.Node.HeartbeatInterval == 0 {
		ticker = time.NewTicker(constant.DefaultHeartbeatInterval * time.Second)
	}
	defer ticker.Stop()

	for {
		// 注册/重新注册逻辑
		leaseID, keepAliveCh, err := m.register()
		if err != nil {
			m.logger.Errorf("Failed to register node: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		m.logger.Infof("Node registered with LeaseID: %x", leaseID)

		// 保持心跳和定期更新状态
	loop:
		for {
			select {
			case <-m.stopChan:
				return
			case _, ok := <-keepAliveCh:
				if !ok {
					m.logger.Warn("KeepAlive channel closed. Re-registering...")
					break loop
				}
				// 续约成功，无需操作
			case <-ticker.C:
				// 更新节点状态 (如磁盘使用率)
				if err := m.updateStatus(leaseID); err != nil {
					m.logger.Errorf("Failed to update node status: %v", err)
				}
			}
		}
	}
}

// register 注册节点
func (m *HeartbeatModule) register() (clientv3.LeaseID, <-chan *clientv3.LeaseKeepAliveResponse, error) {
	state, err := m.collectNodeState()
	if err != nil {
		return 0, nil, err
	}
	return m.etcdService.RegisterNode(context.Background(), state)
}

// updateStatus 更新节点状态
func (m *HeartbeatModule) updateStatus(leaseID clientv3.LeaseID) error {
	state, err := m.collectNodeState()
	if err != nil {
		return err
	}
	return m.etcdService.UpdateNodeStatus(context.Background(), state, leaseID)
}

// collectNodeState 收集节点状态信息
func (m *HeartbeatModule) collectNodeState() (*model.NodeState, error) {
	hostname, _ := os.Hostname()
	// 在实际场景中，应该正确检测 IP
	ip := config.GlobalConfig.Node.Hostname
	if ip == "" {
		var err error
		ip, err = util.GetLocalIP()
		if err != nil {
			m.logger.Warnf("Failed to get local IP: %v", err)
			ip = "127.0.0.1" // Fallback
		}
	}

	used, free, err := util.GetDiskUsage(config.GlobalConfig.Node.WorkDir)
	if err != nil {
		// Log error but continue with 0
		m.logger.Warnf("Failed to get disk usage: %v", err)
	}

	total := used + free
	var usagePercent float64
	if total > 0 {
		usagePercent = float64(used) / float64(total) * 100
	}

	return &model.NodeState{
		NodeID:        m.nodeID,
		Hostname:      hostname,
		IP:            ip,
		Port:          config.GlobalConfig.Node.HttpPort,
		Status:        constant.NodeStatusOnline,
		DiskUsage:     usagePercent,
		TotalDisk:     int64(total),
		AvailableDisk: int64(free),
		LastHeartbeat: time.Now().Unix(),
	}, nil
}
