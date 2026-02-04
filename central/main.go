package main

import (
	"fmt"
	"sync-node/central/config"
	"sync-node/central/handler"
	"sync-node/central/router"
	"sync-node/central/service/alert"
	configServicePkg "sync-node/central/service/config"
	"sync-node/central/service/node"
	"sync-node/central/service/oss"
	"sync-node/central/service/task"
	"sync-node/common/service/etcd"
	"sync-node/common/service/pg"
	"sync-node/common/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	// 加载配置文件，如果失败则直接 panic
	if err := config.InitConfig("central/config.yaml"); err != nil {
		panic(fmt.Sprintf("Failed to init config: %v", err))
	}

	// 初始化日志
	// 根据配置设置日志级别和输出路径
	utils.InitLogger(config.GlobalConfig.Log.Level, config.GlobalConfig.Log.Path)
	log := utils.GetLogger("main")

	// 2. 初始化底层依赖 (Etcd, PG)
	// 初始化 Etcd 客户端，用于服务发现和分布式协调
	etcdClient, err := etcd.NewClient(
		config.GlobalConfig.Etcd.Endpoints,
		config.GlobalConfig.Etcd.Username,
		config.GlobalConfig.Etcd.Password,
	)
	if err != nil {
		log.Fatalf("Failed to init etcd client: %v", err)
	}
	defer etcdClient.Close()

	// 初始化 PostgreSQL 客户端，用于持久化数据
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.GlobalConfig.Postgres.Host,
		config.GlobalConfig.Postgres.User,
		config.GlobalConfig.Postgres.Password,
		config.GlobalConfig.Postgres.DBName,
		config.GlobalConfig.Postgres.Port,
		config.GlobalConfig.Postgres.SSLMode,
	)
	pgClient, err := pg.NewClient(dsn)
	if err != nil {
		log.Fatalf("Failed to init pg client: %v", err)
	}
	// 自动迁移数据库结构，确保表结构最新
	if err := pgClient.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate db: %v", err)
	}

	// 3. 初始化核心服务
	// 初始化 OSS 服务，用于文件存储
	ossService, err := oss.NewOSSService(config.GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to init oss service: %v", err)
	}

	// 初始化报警服务，用于发送系统告警
	alertService, err := alert.NewAlertService(config.GlobalConfig)
	if err != nil {
		log.Fatalf("Failed to init alert service: %v", err)
	}

	// 初始化各业务服务
	taskService := task.NewTaskService(pgClient.GetDB(), etcdClient, ossService, alertService)
	nodeService := node.NewNodeService(pgClient.GetDB())
	configService := configServicePkg.NewConfigService(etcdClient)

	// 启动节点监控
	// 定期检查节点健康状态
	monitor := node.NewMonitor(nodeService, alertService)
	monitor.Start()
	defer monitor.Stop()

	// 启动节点 Watcher (监听状态变化、断联报警、在线统计)
	// 实时监听 Etcd 事件
	watcher := node.NewWatcher(etcdClient, pgClient.GetDB(), alertService, taskService)
	watcher.Start()
	defer watcher.Stop()

	// 4. 初始化 Handlers
	// 绑定服务到 HTTP 处理层
	taskHandler := handler.NewTaskHandler(taskService)
	nodeHandler := handler.NewNodeHandler(nodeService)
	configHandler := handler.NewConfigHandler(configService)

	// 5. 启动 HTTP 服务
	r := gin.Default()

	// 初始化路由规则
	router.InitRouter(r, taskHandler, nodeHandler, configHandler)

	log.Infof("Starting Central Node HTTP server on %s", config.GlobalConfig.Server.Port)
	if err := r.Run(config.GlobalConfig.Server.Port); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
