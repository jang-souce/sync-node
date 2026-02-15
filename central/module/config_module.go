package module

import (
	"context"
	appconfig "sync-node/central/config"
	"sync-node/central/handler"
	"sync-node/central/service/config"
	"sync-node/common/model"
	"sync-node/common/service/etcd"

	"go.uber.org/fx"
)

// ConfigModule 配置模块，提供配置服务和处理器
var ConfigModule = fx.Module("config",
	fx.Provide(
		config.NewConfigService,
		handler.NewConfigHandler,
		// Provide Dynamic Config (from Etcd)
		// 提供动态配置（从 Etcd 获取）
		func(s *config.ConfigService) (*model.GlobalConfig, error) {
			return s.GetGlobalConfig(context.Background())
		},
		// Provide Static Config (from file)
		// 提供静态配置（从配置文件加载）
		func() *appconfig.Config {
			return appconfig.GlobalConfig
		},
		// Bind EtcdClient interface
		// 绑定 Etcd 客户端接口
		func(c *etcd.Client) config.EtcdClient {
			return c
		},
	),
)
