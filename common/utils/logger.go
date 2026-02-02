package utils

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

// InitLogger 初始化日志记录器
func InitLogger(level string, path string) {
	Logger = logrus.New()

	// 设置日志级别
	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}

	// 设置格式
	// 如果是文件输出，通常用JSON格式；如果是控制台，可以用Text格式
	if path != "" {
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z0700",
		})
	} else {
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	// 设置输出
	writers := []io.Writer{os.Stdout}
	if path != "" {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			writers = append(writers, file)
		} else {
			Logger.Warnf("Failed to open log file %s: %v", path, err)
		}
	}
	Logger.SetOutput(io.MultiWriter(writers...))
}

// L 返回全局 Logger
func L() *logrus.Logger {
	return Logger
}
