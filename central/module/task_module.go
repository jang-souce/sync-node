package module

import (
	"sync-node/central/handler"
	"sync-node/central/service/alert"
	"sync-node/central/service/task"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
)

// TaskModule 任务模块，提供任务管理、报警服务和处理器
var TaskModule = fx.Module("task",
	fx.Provide(
		alert.NewAlertService,
		task.NewTaskService,
		handler.NewTaskHandler,
		// Bind AlertService interface
		// 绑定 AlertService 接口
		func(s *alert.AliyunSMSAlertService) alert.AlertService {
			return s
		},
		// Bind EtcdClient interface for TaskService
		// 绑定 Etcd 客户端接口
		func(c *clientv3.Client) task.EtcdClient {
			return c
		},
	),
)
