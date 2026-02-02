package main

import (
	"fmt"
	"log"
	"sync-node/central/config"
	"sync-node/common/service/etcd"
	"sync-node/common/service/pg"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化配置
	if err := config.InitConfig("central/config.yaml"); err != nil {
		log.Fatalf("Failed to init config: %v", err)
	}

	// 2. 初始化底层依赖 (Etcd, PG)
	etcdClient, err := etcd.NewClient(
		config.GlobalConfig.Etcd.Endpoints,
		config.GlobalConfig.Etcd.Username,
		config.GlobalConfig.Etcd.Password,
	)
	if err != nil {
		log.Fatalf("Failed to init etcd client: %v", err)
	}
	defer etcdClient.Close()

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
	// 自动迁移数据库 (Day 1 完成模型定义后通常会包含这个)
	if err := pgClient.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate db: %v", err)
	}

	// 3. 启动基础 HTTP 服务 (仅健康检查，无业务逻辑)
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong from central"})
	})

	log.Printf("Starting Central Node HTTP server on %s", config.GlobalConfig.Server.Port)
	if err := r.Run(config.GlobalConfig.Server.Port); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
