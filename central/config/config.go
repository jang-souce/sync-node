package config

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`

	Etcd struct {
		Endpoints []string `mapstructure:"endpoints"`
		Username  string   `mapstructure:"username"`
		Password  string   `mapstructure:"password"`
	} `mapstructure:"etcd"`

	Postgres struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"postgres"`

	OSS struct {
		Endpoint        string `mapstructure:"endpoint"`
		AccessKeyID     string `mapstructure:"accessKeyId"`
		AccessKeySecret string `mapstructure:"accessKeySecret"`
		BucketName      string `mapstructure:"bucketName"`
	} `mapstructure:"oss"`

	Log struct {
		Level string `mapstructure:"level"`
		Path  string `mapstructure:"path"`
	} `mapstructure:"log"`
}

var GlobalConfig *Config

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
