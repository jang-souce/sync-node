package config

import (
	"fmt"
	"strings"
	"sync-node/common/utils"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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

	// 开启热更新监控，配置文件变更时自动重新加载
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		utils.GetLogger("config").Infof("Config file changed: %s", e.Name)
		if err := viper.Unmarshal(GlobalConfig); err != nil {
			utils.GetLogger("config").Errorf("Failed to unmarshal new config: %v", err)
		}
	})

	return nil
}
