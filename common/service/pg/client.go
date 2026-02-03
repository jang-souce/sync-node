package pg

import (
	"fmt"
	"log"
	"sync-node/common/model"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Client 是 PostgreSQL 数据库客户端封装
type Client struct {
	db *gorm.DB
}

// NewClient 初始化一个新的 PostgreSQL 客户端连接
// dsn 示例: "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
func NewClient(dsn string) (*Client, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(10)
	// 设置打开数据库连接的最大数量
	sqlDB.SetMaxOpenConns(100)
	// 设置了连接可复用的最大时间
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Client{db: db}, nil
}

// AutoMigrate 自动迁移数据库结构
func (c *Client) AutoMigrate() error {
	log.Println("初始化数据库自动迁移...")
	err := c.db.AutoMigrate(
		&model.MainTask{},
		&model.SubTask{},
		&model.NodeState{},
		&model.GlobalConfig{},
	)
	if err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	log.Println("数据库自动迁移完成")
	return nil
}

// GetDB 获取底层的 gorm.DB 实例
func (c *Client) GetDB() *gorm.DB {
	return c.db
}
