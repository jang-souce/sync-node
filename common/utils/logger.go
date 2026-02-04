package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// InitLogger 初始化日志记录器
func InitLogger(level string, path string) {
	Logger = logrus.New()

	// 设置日志级别
	l, err := logrus.ParseLevel(level)
	if err != nil {
		l = logrus.InfoLevel
	}
	Logger.SetLevel(l)

	// 设置格式（默认使用 TextFormatter），并添加脱敏 Hook
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
		DisableColors:   false,
	})
	Logger.AddHook(&RedactHook{})

	// 设置输出
	writers := []io.Writer{os.Stdout}
	if path != "" {
		// 确保日志目录存在
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			Logger.Warnf("Failed to create log directory %s: %v", dir, err)
		} else {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				writers = append(writers, file)
				// 输出到文件时禁用颜色
				Logger.SetFormatter(&logrus.TextFormatter{
					FullTimestamp:   true,
					TimestampFormat: "2006-01-02 15:04:05",
					DisableColors:   true,
				})
				Logger.AddHook(&RedactHook{})
			} else {
				Logger.Warnf("Failed to open log file %s: %v", path, err)
			}
		}
	}
	Logger.SetOutput(io.MultiWriter(writers...))
}

// L 返回全局 Logger
func L() *logrus.Logger {
	if Logger == nil {
		Logger = logrus.New()
	}
	return Logger
}

// GetLogger 返回带有模块字段的 logger entry
func GetLogger(module string) *logrus.Entry {
	if Logger == nil {
		Logger = logrus.New()
	}
	return Logger.WithField("module", module)
}
