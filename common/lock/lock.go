package lock

import "context"

// DistributedLock 定义分布式锁实例接口
type DistributedLock interface {
	// Unlock 释放锁
	Unlock(ctx context.Context) error
}

// Locker 定义获取分布式锁的接口
type Locker interface {
	// Lock 获取锁
	// key: 锁的键
	// ttl: 锁的过期时间（秒）
	Lock(ctx context.Context, key string, ttl int) (DistributedLock, error)
}
