package config

import (
	"fmt"
	"strings"
	"sync-node/common/utils"

	"bytes"
	"context"
	"sync-node/common/constant"
	"sync-node/common/service/etcd"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.etcd.io/etcd/api/v3/mvccpb"
)

// Config 定义应用程序的配置结构，映射 config.yaml 文件
type Config struct {
	Server struct {
		Port string `mapstructure:"port"` // HTTP 服务监听端口
	} `mapstructure:"server"`

	Etcd struct {
		Endpoints []string `mapstructure:"endpoints"` // Etcd 集群地址列表
		Username  string   `mapstructure:"username"`  // Etcd 用户名
		Password  string   `mapstructure:"password"`  // Etcd 密码
	} `mapstructure:"etcd"`

	Postgres struct {
		Host     string `mapstructure:"host"`     // 数据库主机地址
		Port     int    `mapstructure:"port"`     // 数据库端口
		User     string `mapstructure:"user"`     // 数据库用户名
		Password string `mapstructure:"password"` // 数据库密码
		DBName   string `mapstructure:"dbname"`   // 数据库名称
		SSLMode  string `mapstructure:"sslmode"`  // SSL 连接模式
	} `mapstructure:"postgres"`

	OSS struct {
		Provider        string `mapstructure:"provider"`        // 对象存储提供商 (aliyun 或 minio)
		Endpoint        string `mapstructure:"endpoint"`        // OSS 访问端点
		AccessKeyID     string `mapstructure:"accessKeyId"`     // Access Key ID
		AccessKeySecret string `mapstructure:"accessKeySecret"` // Access Key Secret
		BucketName      string `mapstructure:"bucketName"`      // 存储桶名称
	} `mapstructure:"oss"`

	SMS struct {
		AccessKeyID     string   `mapstructure:"accessKeyId"`     // 短信服务 Access Key ID
		AccessKeySecret string   `mapstructure:"accessKeySecret"` // 短信服务 Access Key Secret
		SignName        string   `mapstructure:"signName"`        // 短信签名
		TemplateCode    string   `mapstructure:"templateCode"`    // 短信模版 ID
		PhoneNumbers    []string `mapstructure:"phoneNumbers"`    // 接收报警的手机号列表
	} `mapstructure:"sms"`

	Log struct {
		Level string `mapstructure:"level"` // 日志级别 (debug, info, warn, error)
		Path  string `mapstructure:"path"`  // 日志文件路径
	} `mapstructure:"log"`
}

var GlobalConfig *Config

// InitConfig 初始化配置
// path: 配置文件路径
// 使用 Viper 加载配置并开启热更新监控
func InitConfig(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv() // 支持环境变量覆盖
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 开启本地文件热更新监控
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		utils.GetLogger("config").Infof("Config file changed: %s", e.Name)
		if err := viper.Unmarshal(GlobalConfig); err != nil {
			utils.GetLogger("config").Errorf("Failed to unmarshal new config: %v", err)
		}
	})

	return nil
}

// WatchEtcdConfig 监听 Etcd 中的全局配置变化并热更新
// 监听 Key: /file_sync/config/global
func WatchEtcdConfig(ctx context.Context, client *etcd.Client) {
	key := constant.EtcdConfigPrefix + "global"
	utils.GetLogger("config").Infof("Start watching etcd config key: %s", key)

	// 1. 首次获取配置
	resp, err := client.Get(ctx, key)
	if err == nil && len(resp.Kvs) > 0 {
		updateConfigFromEtcd(resp.Kvs[0].Value)
	}

	// 2. 监听变化
	watchChan := client.Watch(ctx, key)
	go func() {
		for resp := range watchChan {
			for _, ev := range resp.Events {
				if ev.Type == mvccpb.PUT {
					utils.GetLogger("config").Infof("Etcd config changed: %s", key)
					updateConfigFromEtcd(ev.Kv.Value)
				}
			}
		}
	}()
}

func updateConfigFromEtcd(data []byte) {
	// 使用 Viper 合并配置
	viper.SetConfigType("yaml")
	if err := viper.MergeConfig(bytes.NewBuffer(data)); err != nil {
		utils.GetLogger("config").Errorf("Failed to merge etcd config: %v", err)
		return
	}

	// 更新 GlobalConfig
	// 注意：这里需要确保并发安全，或者接受短暂的不一致
	// 简单起见，我们直接 Unmarshal 到 GlobalConfig
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		utils.GetLogger("config").Errorf("Failed to unmarshal new config from etcd: %v", err)
		return
	}
	utils.GetLogger("config").Info("Global config updated from Etcd")
}
