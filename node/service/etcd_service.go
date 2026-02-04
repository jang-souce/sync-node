package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/service/etcd"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdService 封装 Etcd 客户端操作
type EtcdService struct {
	client *etcd.Client
}

// NewEtcdService 创建 Etcd 服务实例
func NewEtcdService(endpoints []string, username, password string) (*EtcdService, error) {
	cli, err := etcd.NewClient(endpoints, username, password)
	if err != nil {
		return nil, err
	}
	return &EtcdService{client: cli}, nil
}

// RegisterNode 注册节点到 Etcd 并保持心跳
func (s *EtcdService) RegisterNode(ctx context.Context, nodeState *model.NodeState) (clientv3.LeaseID, <-chan *clientv3.LeaseKeepAliveResponse, error) {
	// 1. 创建租约
	leaseID, err := s.client.GrantLease(ctx, constant.DefaultLeaseTTL)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to grant lease: %w", err)
	}

	// 2. 写入节点状态（关联租约）
	key := constant.EtcdNodePrefix + nodeState.NodeID
	val, err := json.Marshal(nodeState)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to marshal node state: %w", err)
	}

	_, err = s.client.Put(ctx, key, string(val), clientv3.WithLease(leaseID))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to put node state: %w", err)
	}

	// 3. 开启自动续约
	ch, err := s.client.KeepAlive(ctx, leaseID)
	return leaseID, ch, err
}

// UpdateNodeStatus 更新 Etcd 中的节点状态
// 注意：为了保持租约，需要传入 LeaseID。
// 实际上，心跳模块会定期推送完整状态。
func (s *EtcdService) UpdateNodeStatus(ctx context.Context, nodeState *model.NodeState, leaseID clientv3.LeaseID) error {
	key := constant.EtcdNodePrefix + nodeState.NodeID
	val, err := json.Marshal(nodeState)
	if err != nil {
		return fmt.Errorf("failed to marshal node state: %w", err)
	}
	_, err = s.client.Put(ctx, key, string(val), clientv3.WithLease(leaseID))
	return err
}

// WatchTasks 监听分配给当前节点的任务
func (s *EtcdService) WatchTasks(ctx context.Context, nodeID string) clientv3.WatchChan {
	key := constant.EtcdTaskPrefix + nodeID + "/"
	return s.client.Watch(ctx, key, clientv3.WithPrefix())
}

// GetPendingTasks 获取当前节点的所有 Pending 任务
func (s *EtcdService) GetPendingTasks(ctx context.Context, nodeID string) ([]model.SubTask, error) {
	keyPrefix := constant.EtcdTaskPrefix + nodeID + "/"
	resp, err := s.client.Get(ctx, keyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var tasks []model.SubTask
	for _, kv := range resp.Kvs {
		var task model.SubTask
		if err := json.Unmarshal(kv.Value, &task); err != nil {
			continue
		}
		if task.Status == constant.TaskStatusPending {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

// UpdateTaskStatus 更新任务状态
func (s *EtcdService) UpdateTaskStatus(ctx context.Context, taskID string, status string, errorMsg string, extra map[string]interface{}) error {
	// 更新状态 key: /file_sync/status/<taskID>
	// 中心节点会监听此 key，或者我们需要通过 API 更新数据库中的 SubTask 记录？
	// 根据需求：“子节点上报任务状态”。
	// 通常上报到 Etcd 适合实时监控。
	// constant.go 中定义的 key 结构是 EtcdTaskStatusPrefix = EtcdRootPrefix + "status/"

	key := constant.EtcdTaskStatusPrefix + taskID
	val := map[string]interface{}{
		"status":    status,
		"errorMsg":  errorMsg,
		"updatedAt": time.Now().Unix(),
	}
	if extra != nil {
		for k, v := range extra {
			val[k] = v
		}
	}
	data, _ := json.Marshal(val)

	_, err := s.client.Put(ctx, key, string(data))
	return err
}

// Close 关闭 Etcd 客户端连接
func (s *EtcdService) Close() error {
	return s.client.Close()
}
