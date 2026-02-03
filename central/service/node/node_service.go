package node

import (
	"context"
	"encoding/json"
	"sync-node/common/constant"
	"sync-node/common/model"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdClient 定义 NodeService 所需的 Etcd 客户端接口
type EtcdClient interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
}

// NodeService 提供节点信息查询服务
type NodeService struct {
	etcd EtcdClient
}

// NewNodeService 创建 NodeService 实例
func NewNodeService(etcd EtcdClient) *NodeService {
	return &NodeService{etcd: etcd}
}

// GetNodes 获取所有已注册的节点信息
func (s *NodeService) GetNodes(ctx context.Context) ([]*model.NodeState, error) {
	// 使用前缀搜索获取所有节点 Key
	resp, err := s.etcd.Get(ctx, constant.EtcdNodePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var nodes []*model.NodeState
	for _, kv := range resp.Kvs {
		var node model.NodeState
		// 反序列化节点状态信息
		if err := json.Unmarshal(kv.Value, &node); err == nil {
			nodes = append(nodes, &node)
		}
	}
	return nodes, nil
}

// GetNode 获取指定 ID 的节点信息
func (s *NodeService) GetNode(ctx context.Context, id string) (*model.NodeState, error) {
	key := constant.EtcdNodePrefix + id
	resp, err := s.etcd.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	// 如果没有找到对应的 Key，返回 nil
	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var node model.NodeState
	if err := json.Unmarshal(resp.Kvs[0].Value, &node); err != nil {
		return nil, err
	}
	return &node, nil
}
