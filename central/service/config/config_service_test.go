package config

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

func (m *MockEtcdClient) Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	args := m.Called(ctx, key, value, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientv3.PutResponse), args.Error(1)
}

// TestConfigService_GetGlobalConfig 测试获取全局配置
func TestConfigService_GetGlobalConfig(t *testing.T) {
	mockEtcd := new(MockEtcdClient)
	service := NewConfigService(mockEtcd)

	// 场景 1: 配置存在
	cfg := model.GlobalConfig{HeartbeatInterval: 10, TaskTimeout: 300}
	val, _ := json.Marshal(cfg)
	mockResp := &clientv3.GetResponse{
		Kvs: []*mvccpb.KeyValue{
			{Value: val},
		},
	}
	mockEtcd.On("Get", mock.Anything, constant.EtcdConfigPrefix, mock.Anything).Return(mockResp, nil).Once()

	result, err := service.GetGlobalConfig(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 10, result.HeartbeatInterval)

	// Case 2: Config not exists (default)
	// 场景 2: 配置不存在（返回默认值）
	mockRespEmpty := &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{}}
	mockEtcd.On("Get", mock.Anything, constant.EtcdConfigPrefix, mock.Anything).Return(mockRespEmpty, nil).Once()

	result, err = service.GetGlobalConfig(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, constant.DefaultHeartbeatInterval, result.HeartbeatInterval)
}

// TestConfigService_UpdateGlobalConfig 测试更新全局配置
func TestConfigService_UpdateGlobalConfig(t *testing.T) {
	mockEtcd := new(MockEtcdClient)
	service := NewConfigService(mockEtcd)

	cfg := &model.GlobalConfig{HeartbeatInterval: 20, TaskTimeout: 600}

	mockEtcd.On("Put", mock.Anything, constant.EtcdConfigPrefix, mock.Anything, mock.Anything).Return(nil, nil)

	err := service.UpdateGlobalConfig(context.Background(), cfg)
	assert.NoError(t, err)
	mockEtcd.AssertExpectations(t)
}
