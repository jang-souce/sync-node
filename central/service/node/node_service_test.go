package node

import (
	"context"
	"encoding/json"
	"testing"

	"sync-node/common/constant"
	"sync-node/common/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// MockEtcdClient 模拟 Etcd 客户端
type MockEtcdClient struct {
	mock.Mock
}

func (m *MockEtcdClient) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	args := m.Called(ctx, key, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientv3.GetResponse), args.Error(1)
}

// TestNodeService_GetNodes 测试获取节点列表
func TestNodeService_GetNodes(t *testing.T) {
	mockEtcd := new(MockEtcdClient)
	service := NewNodeService(mockEtcd)

	// Mock Data
	// 模拟数据
	node1 := model.NodeState{NodeID: "node-1", Status: "online"}
	node2 := model.NodeState{NodeID: "node-2", Status: "offline"}
	val1, _ := json.Marshal(node1)
	val2, _ := json.Marshal(node2)

	mockResp := &clientv3.GetResponse{
		Kvs: []*mvccpb.KeyValue{
			{Key: []byte(constant.EtcdNodePrefix + "node-1"), Value: val1},
			{Key: []byte(constant.EtcdNodePrefix + "node-2"), Value: val2},
		},
	}

	mockEtcd.On("Get", mock.Anything, constant.EtcdNodePrefix, mock.Anything).Return(mockResp, nil)

	nodes, err := service.GetNodes(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)
	assert.Equal(t, "node-1", nodes[0].NodeID)
	assert.Equal(t, "node-2", nodes[1].NodeID)
}

// TestNodeService_GetNode 测试获取指定节点信息
func TestNodeService_GetNode(t *testing.T) {
	mockEtcd := new(MockEtcdClient)
	service := NewNodeService(mockEtcd)

	// 模拟数据
	node := model.NodeState{NodeID: "node-1", Status: "online"}
	val, _ := json.Marshal(node)

	mockResp := &clientv3.GetResponse{
		Kvs: []*mvccpb.KeyValue{
			{Key: []byte(constant.EtcdNodePrefix + "node-1"), Value: val},
		},
	}

	mockEtcd.On("Get", mock.Anything, constant.EtcdNodePrefix+"node-1", mock.Anything).Return(mockResp, nil)

	result, err := service.GetNode(context.Background(), "node-1")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "node-1", result.NodeID)
}
