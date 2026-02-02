package config

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Node struct {
		ID       string `mapstructure:"id"`
		WorkDir  string `mapstructure:"workDir"`
		Hostname string `mapstructure:"hostname"`
	} `mapstructure:"node"`

	Etcd struct {
		Endpoints []string `mapstructure:"endpoints"`
		Username  string   `mapstructure:"username"`
		Password  string   `mapstructure:"password"`
	} `mapstructure:"etcd"`

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
