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

	// 设置格式
	// 默认使用 TextFormatter，兼顾控制台阅读
	// 如果需要结构化日志，可以考虑 JSONFormatter
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true, // 即使输出到文件也保持颜色代码可能会导致乱码，通常根据 Output 自动判断
		DisableColors:   false,
	})

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
				// 如果有文件输出，且希望文件里是 JSON 格式，这里比较难办，因为 logrus 只能设置一个 Formatter。
				// 通常如果同时输出控制台和文件，要么都 JSON，要么都 Text。
				// 这里为了开发体验，保留 TextFormatter。
				// 生产环境建议只输出到文件或 stdout，并使用 JSONFormatter。

				// 如果确实需要文件 JSON 而控制台 Text，需要使用 Hook，例如 lfshook (需要额外依赖)
				// 这里保持简单，统一使用 TextFormatter，但去掉了 ForceColors 以避免文件乱码，
				// 不过 logrus 默认对 TTY 开启颜色，对文件关闭颜色。
				// 如果使用 MultiWriter，logrus 无法自动判断，会认为不是 TTY。
				// 所以这里显式设置 DisableColors: false (允许颜色)，但其实 MultiWriter 会导致 CheckIfTerminal 返回 false

				// 修正策略：为了文件可读性，如果配置了文件输出，我们禁用颜色或者接受文件里有颜色代码
				// 或者使用 JSONFormatter
				Logger.SetFormatter(&logrus.TextFormatter{
					FullTimestamp:   true,
					TimestampFormat: "2006-01-02 15:04:05",
					DisableColors:   true, // 输出到文件时禁用颜色
				})
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
