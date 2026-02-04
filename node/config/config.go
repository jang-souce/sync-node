package config

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 定义全局配置结构
type Config struct {
	// Node 节点相关配置
	Node struct {
		ID                string `mapstructure:"id"`                // 节点唯一标识
		WorkDir           string `mapstructure:"workDir"`           // 工作目录
		Hostname          string `mapstructure:"hostname"`          // 主机名
		HttpPort          string `mapstructure:"httpPort"`          // HTTP 服务端口
		HeartbeatInterval int    `mapstructure:"heartbeatInterval"` // 心跳间隔(秒)
		TaskTimeout       int    `mapstructure:"taskTimeout"`       // 任务超时时间(秒)
		RetryMax          int    `mapstructure:"retryMax"`          // 最大重试次数
		RetryDelay        int    `mapstructure:"retryDelay"`        // 初始重试延迟(秒)
	} `mapstructure:"node"`

	// Etcd 配置
	Etcd struct {
		Endpoints []string `mapstructure:"endpoints"` // Etcd 节点地址列表
		Username  string   `mapstructure:"username"`  // 用户名
		Password  string   `mapstructure:"password"`  // 密码
	} `mapstructure:"etcd"`

	// Log 日志配置
	Log struct {
		Level string `mapstructure:"level"` // 日志级别
		Path  string `mapstructure:"path"`  // 日志文件路径
	} `mapstructure:"log"`
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// InitConfig 初始化配置
func InitConfig(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 开启热更新监控
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file changed: %s\n", e.Name)
		if err := viper.Unmarshal(GlobalConfig); err != nil {
			fmt.Printf("Failed to unmarshal new config: %v\n", err)
		}
	})

	return nil
}
