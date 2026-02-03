package node

import (
	"context"
	"testing"
	"time"

	"sync-node/common/constant"
	"sync-node/common/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNodeProvider 模拟节点提供者
type MockNodeProvider struct {
	mock.Mock
}

func (m *MockNodeProvider) GetNodes(ctx context.Context) ([]*model.NodeState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.NodeState), args.Error(1)
}

// MockAlertService (Copy from task package or redefine)
// MockAlertService 模拟报警服务
type MockAlertService struct {
	mock.Mock
}

func (m *MockAlertService) SendAlert(param map[string]string) error {
	args := m.Called(param)
	return args.Error(0)
}

// TestMonitor_CheckNodes 测试节点监控逻辑
func TestMonitor_CheckNodes(t *testing.T) {
	mockNode := new(MockNodeProvider)
	mockAlert := new(MockAlertService)
	monitor := NewMonitor(mockNode, mockAlert)

	// 模拟数据
	now := time.Now()
	onlineNode := &model.NodeState{
		NodeID:        "node-online",
		Status:        "online",
		LastHeartbeat: now.Unix(),
	}
	// 离线节点（心跳时间超过阈值）
	offlineNode := &model.NodeState{
		NodeID:        "node-offline",
		Status:        "online",
		LastHeartbeat: now.Add(-time.Duration(constant.DefaultHeartbeatInterval*4) * time.Second).Unix(),
	}

	nodes := []*model.NodeState{onlineNode, offlineNode}
	mockNode.On("GetNodes", mock.Anything).Return(nodes, nil)

	// 预期对离线节点发送报警
	mockAlert.On("SendAlert", mock.MatchedBy(func(param map[string]string) bool {
		return param["node"] == "node-offline" && param["reason"] == "Offline"
	})).Return(nil)

	// 手动触发检查
	alertHistory := make(map[string]time.Time)
	monitor.checkNodes(alertHistory)

	// Verify
	// 验证报警是否发送
	mockAlert.AssertExpectations(t)
	assert.Contains(t, alertHistory, "node-offline")

	// 立即再次运行检查，不应再次报警（防抖）
	monitor.checkNodes(alertHistory)
	mockAlert.AssertNumberOfCalls(t, "SendAlert", 1)
}
