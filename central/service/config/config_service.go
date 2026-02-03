package config

import (
	"context"
	"encoding/json"
	"sync-node/common/constant"
	"sync-node/common/model"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdClient 定义 ConfigService 所需的 Etcd 客户端接口
type EtcdClient interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
}

// ConfigService 提供全局配置管理服务
type ConfigService struct {
	etcd EtcdClient
}

// NewConfigService 创建 ConfigService 实例
func NewConfigService(etcd EtcdClient) *ConfigService {
	return &ConfigService{etcd: etcd}
}

// GetGlobalConfig 获取全局配置
func (s *ConfigService) GetGlobalConfig(ctx context.Context) (*model.GlobalConfig, error) {
	resp, err := s.etcd.Get(ctx, constant.EtcdConfigPrefix)
	if err != nil {
		return nil, err
	}
	// 如果配置不存在，返回默认值
	if len(resp.Kvs) == 0 {
		return &model.GlobalConfig{
			HeartbeatInterval: constant.DefaultHeartbeatInterval,
			TaskTimeout:       3600,
		}, nil
	}

	var cfg model.GlobalConfig
	if err := json.Unmarshal(resp.Kvs[0].Value, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateGlobalConfig 更新全局配置
func (s *ConfigService) UpdateGlobalConfig(ctx context.Context, cfg *model.GlobalConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	// 将配置写入 Etcd
	_, err = s.etcd.Put(ctx, constant.EtcdConfigPrefix, string(data))
	return err
}
