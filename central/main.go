package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync-node/central/config"
	"sync-node/central/handler"
	"sync-node/central/router"
	"sync-node/central/service/alert"
	configServicePkg "sync-node/central/service/config"
	"sync-node/central/service/node"
	"sync-node/central/service/oss"
	"sync-node/central/service/task"
	"sync-node/common/lock"
	"sync-node/common/service/etcd"
	"sync-node/common/service/pg"
	"sync-node/common/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func main() {
	app := fx.New(
		// 1. Provide Dependencies
		fx.Provide(
			// Config
			NewConfig,
			// DB
			NewDB,
			// Etcd
			NewEtcdClient,
			// Lock
			lock.NewEtcdLocker,
			fx.Annotate(
				func(l *lock.EtcdLocker) lock.Locker { return l },
				fx.As(new(lock.Locker)),
			),
			// Services
			oss.NewOSSService,
			fx.Annotate(
				alert.NewAlertService,
				fx.As(new(alert.AlertService)),
			),
			// TaskService 需要的 EtcdClient 接口，这里直接使用 etcd.Client
			// 但 TaskService 定义的 EtcdClient 接口可能与 etcd.Client 不完全匹配（如果有额外方法）
			// 检查发现 TaskService 定义的接口方法 etcd.Client 都实现了
			// 需要让 Fx 知道 etcd.Client 实现了 task.EtcdClient
			fx.Annotate(
				func(c *etcd.Client) *etcd.Client { return c },
				fx.As(new(task.EtcdClient)),
			),
			// ConfigService 需要的 EtcdClient 接口
			fx.Annotate(
				func(c *etcd.Client) *etcd.Client { return c },
				fx.As(new(configServicePkg.EtcdClient)),
			),
			// 同样的，ConfigService 也需要 EtcdClient，但它直接使用了 *etcd.Client 类型（假设）
			// 如果 ConfigService 使用接口，也需要处理。
			// 检查 main.go 原代码: configService := configServicePkg.NewConfigService(etcdClient)
			// 假设 NewConfigService 接受 *etcd.Client
			task.NewTaskService,
			node.NewNodeService,
			// Monitor needs NodeProvider
			fx.Annotate(
				func(s *node.NodeService) *node.NodeService { return s },
				fx.As(new(node.NodeProvider)),
			),
			configServicePkg.NewConfigService,

			// Node Monitor & Watcher
			node.NewMonitor,
			node.NewWatcher,

			// Handlers
			handler.NewTaskHandler,
			handler.NewNodeHandler,
			handler.NewConfigHandler,
			handler.NewFileHandler,

			// Gin Engine
			gin.Default,
		),

		// 2. Invoke Startup Logic
		fx.Invoke(
			InitLogger,
			RegisterRoutes,
			StartServer,
			StartBackgroundServices,
		),
	)

	app.Run()
}

// NewConfig 加载配置
func NewConfig() (*config.Config, error) {
	if err := config.InitConfig("central/config.yaml"); err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}
	return config.GlobalConfig, nil
}

// NewDB 初始化数据库
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.Port,
		cfg.Postgres.SSLMode,
	)
	pgClient, err := pg.NewClient(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to init pg client: %w", err)
	}
	// 自动迁移
	if err := pgClient.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}
	return pgClient.GetDB(), nil
}

// NewEtcdClient 初始化 Etcd
func NewEtcdClient(cfg *config.Config) (*etcd.Client, error) {
	return etcd.NewClient(
		cfg.Etcd.Endpoints,
		cfg.Etcd.Username,
		cfg.Etcd.Password,
	)
}

// InitLogger 初始化日志
func InitLogger(cfg *config.Config) {
	utils.InitLogger(cfg.Log.Level, cfg.Log.Path)
}

// RegisterRoutes 注册路由
func RegisterRoutes(
	r *gin.Engine,
	taskH *handler.TaskHandler,
	nodeH *handler.NodeHandler,
	configH *handler.ConfigHandler,
	fileH *handler.FileHandler,
) {
	router.InitRouter(r, taskH, nodeH, configH, fileH)
}

// StartServer 启动 HTTP 服务
func StartServer(lc fx.Lifecycle, r *gin.Engine, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			port := cfg.Server.Port
			utils.GetLogger("main").Infof("Starting Central Node HTTP server on %s", port)
			addr := port
			if !strings.HasPrefix(port, ":") {
				addr = ":" + port
			}
			go func() {
				if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
					utils.GetLogger("main").Errorf("Failed to start server: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			utils.GetLogger("main").Info("Stopping HTTP server")
			return nil
		},
	})
}

// StartBackgroundServices 启动后台服务 (Watcher, Monitor, ConfigWatch)
func StartBackgroundServices(
	lc fx.Lifecycle,
	monitor *node.Monitor,
	watcher *node.Watcher,
	etcdClient *etcd.Client,
	taskService *task.TaskService,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 启动节点监控
			monitor.Start()
			// 启动节点 Watcher
			watcher.Start()
			// 启动 Etcd 配置监听
			// 注意：WatchEtcdConfig 内部是异步的，或者我们需要在这里异步调用
			// WatchEtcdConfig 本身启动了 go routine
			config.WatchEtcdConfig(context.Background(), etcdClient)

			// 异步启动任务重新发布（修复 Pending 任务丢失问题）
			go func() {
				// 稍微延迟启动，确保系统完全就绪
				// time.Sleep(5 * time.Second) // Optional
				utils.GetLogger("main").Info("Starting pending tasks recovery...")
				if err := taskService.RepublishPendingTasks(context.Background()); err != nil {
					utils.GetLogger("main").Errorf("Failed to republish pending tasks: %v", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			monitor.Stop()
			watcher.Stop()
			return nil
		},
	})
}
