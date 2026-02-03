package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigHotReload 测试配置文件热更新功能
func TestConfigHotReload(t *testing.T) {
	// 1. 创建临时配置文件
	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name()) // 清理文件

	initialConfig := `
server:
  port: ":8080"
log:
  level: "info"
`
	_, err = tmpFile.WriteString(initialConfig)
	require.NoError(t, err)
	tmpFile.Close()

	// 2. 初始化配置
	err = InitConfig(tmpFile.Name())
	require.NoError(t, err)

	// 验证初始值
	assert.Equal(t, ":8080", GlobalConfig.Server.Port)
	assert.Equal(t, "info", GlobalConfig.Log.Level)

	// 3. 修改配置文件
	newConfig := `
server:
  port: ":9090"
log:
  level: "debug"
`
	err = os.WriteFile(tmpFile.Name(), []byte(newConfig), 0644)
	require.NoError(t, err)

	// 4. 等待热更新 (Viper 是异步监控)
	time.Sleep(100 * time.Millisecond)

	// 5. 验证新值
	assert.Equal(t, ":9090", GlobalConfig.Server.Port, "Server port should be updated")
	assert.Equal(t, "debug", GlobalConfig.Log.Level, "Log level should be updated")
}
