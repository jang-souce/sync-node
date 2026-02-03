package etcd

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type Client struct {
	cli *clientv3.Client
}

// NewClient 创建一个新的 Etcd 客户端
func NewClient(endpoints []string, username, password string) (*Client, error) {
	config := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		Username:    username,
		Password:    password,
	}
	cli, err := clientv3.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	return c.cli.Close()
}

// Put 写入键值对
func (c *Client) Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return c.cli.Put(ctx, key, value, opts...)
}

// Get 获取键值对
func (c *Client) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return c.cli.Get(ctx, key, opts...)
}

// Watch 监听键或前缀的变化
func (c *Client) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return c.cli.Watch(ctx, key, opts...)
}

// GrantLease 申请一个租约
func (c *Client) GrantLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error) {
	resp, err := c.cli.Grant(ctx, ttl)
	if err != nil {
		return 0, err
	}
	return resp.ID, nil
}

// KeepAlive 保持租约 (心跳)
func (c *Client) KeepAlive(ctx context.Context, leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	return c.cli.KeepAlive(ctx, leaseID)
}

// Delete 删除键
func (c *Client) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	return c.cli.Delete(ctx, key, opts...)
}

// GetClient 获取底层客户端实例 (用于高级操作)
func (c *Client) GetClient() *clientv3.Client {
	return c.cli
}

// NewMutex 创建分布式锁
func (c *Client) NewMutex(pfx string) (*concurrency.Mutex, error) {
	// 创建会话
	session, err := concurrency.NewSession(c.cli)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd session: %w", err)
	}
	// 基于会话创建锁
	return concurrency.NewMutex(session, pfx), nil
}
