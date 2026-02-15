package lock

import (
	"context"
	"sync-node/common/service/etcd"
)

// EtcdLocker 基于 Etcd 实现的分布式锁
type EtcdLocker struct {
	client *etcd.Client
}

// NewEtcdLocker 创建 EtcdLocker 实例
func NewEtcdLocker(client *etcd.Client) *EtcdLocker {
	return &EtcdLocker{client: client}
}

// Lock 获取锁
func (l *EtcdLocker) Lock(ctx context.Context, key string, ttl int) (DistributedLock, error) {
	return l.client.AcquireMutex(ctx, key, ttl)
}
