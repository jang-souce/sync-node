package module

import (
	"context"

	"sync-node/central/handler"
	"sync-node/central/service/node"

	"go.uber.org/fx"
)

// NodeModule 节点模块，提供节点管理服务和处理器
var NodeModule = fx.Module("node",
	fx.Provide(
		node.NewNodeService,
		handler.NewNodeHandler,
		node.NewMonitor,
		node.NewWatcher,
		// Bind NodeProvider interface
		func(s *node.NodeService) node.NodeProvider {
			return s
		},
	),
	fx.Invoke(func(m *node.Monitor, w *node.Watcher, lc fx.Lifecycle) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				m.Start()
				w.Start()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				w.Stop()
				m.Stop()
				return nil
			},
		})
	}),
)
