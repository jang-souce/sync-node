package main

import (
	"os"
	"os/signal"
	"sync-node/common/utils"
	"sync-node/node/config"
	"sync-node/node/module"
	"sync-node/node/service"
	"syscall"
)

func main() {
	// 1. 初始化日志 (默认 info 级别, 输出到 stdout)
	utils.InitLogger("info", "")
	logger := utils.GetLogger("main")

	// 2. 初始化配置
	if err := config.InitConfig("config.yaml"); err != nil {
		logger.Fatalf("Failed to init config: %v", err)
	}

	// 重新初始化日志，应用配置中的级别和路径
	utils.InitLogger(config.GlobalConfig.Log.Level, config.GlobalConfig.Log.Path)
	// 更新 main 函数使用的 logger 实例
	logger = utils.GetLogger("main")

	logger.Infof("Config initialized. NodeID: %s", config.GlobalConfig.Node.ID)

	// 3. 确保工作目录存在
	if err := utils.EnsureDir(config.GlobalConfig.Node.WorkDir); err != nil {
		logger.Fatalf("Failed to ensure work dir: %v", err)
	}

	// 4. 初始化服务
	etcdService, err := service.NewEtcdService(
		config.GlobalConfig.Etcd.Endpoints,
		config.GlobalConfig.Etcd.Username,
		config.GlobalConfig.Etcd.Password,
	)
	if err != nil {
		logger.Fatalf("Failed to init etcd service: %v", err)
	}
	defer etcdService.Close()

	httpService := service.NewHttpService()

	// 5. 初始化模块
	heartbeatModule := module.NewHeartbeatModule(etcdService, config.GlobalConfig.Node.ID)
	taskModule := module.NewTaskModule(
		etcdService,
		httpService,
		config.GlobalConfig.Node.ID,
		config.GlobalConfig.Node.WorkDir,
	)

	// 6. 启动模块
	heartbeatModule.Start()
	taskModule.Start()

	// 7. 启动 HTTP 服务 (在协程中)
	go func() {
		addr := config.GlobalConfig.Node.HttpPort
		if addr == "" {
			addr = ":8081"
		}
		logger.Infof("Starting HTTP server on %s", addr)
		if err := httpService.Start(addr); err != nil {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 8. 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down node...")

	// 9. 优雅停止
	heartbeatModule.Stop()
	taskModule.Stop()

	logger.Info("Node stopped")
}
