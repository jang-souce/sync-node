package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync-node/common/utils"
	"sync-node/node/config"
	"time"

	"github.com/gin-gonic/gin"
)

// HttpService 提供 HTTP 服务和文件下载功能
type HttpService struct {
	engine *gin.Engine
	client *http.Client
}

// NewHttpService 创建 HTTP 服务实例
func NewHttpService() *HttpService {
	return &HttpService{
		engine: gin.Default(),
		client: &http.Client{},
	}
}

// Start 启动 HTTP 服务
func (s *HttpService) Start(addr string) error {
	return s.engine.Run(addr)
}

// GetEngine 获取 Gin 引擎实例
func (s *HttpService) GetEngine() *gin.Engine {
	return s.engine
}

// ProgressReader 用于监控读取进度的 Reader
type ProgressReader struct {
	io.Reader
	Total    int64
	Current  int64
	Callback func(int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Callback != nil {
		pr.Callback(pr.Current)
	}
	return n, err
}

// DownloadFile 从 URL 下载文件到 destPath，支持断点续传和重试
func (s *HttpService) DownloadFile(ctx context.Context, url string, destPath string, progressCallback func(int64)) error {
	// 获取重试配置
	maxRetries := config.GlobalConfig.Node.RetryMax
	if maxRetries <= 0 {
		maxRetries = 3 // 默认值
	}
	initialDelay := time.Duration(config.GlobalConfig.Node.RetryDelay) * time.Second
	if initialDelay <= 0 {
		initialDelay = 1 * time.Second // 默认值
	}
	maxDelay := initialDelay * 10

	// 使用统一的重试工具
	return utils.Retry(ctx, maxRetries, initialDelay, maxDelay, func() error {
		return s.downloadOneAttempt(ctx, url, destPath, progressCallback)
	})
}

// downloadOneAttempt 执行一次下载尝试
func (s *HttpService) downloadOneAttempt(ctx context.Context, url string, destPath string, progressCallback func(int64)) error {
	// 确保目录存在
	if err := utils.EnsureDir(filepath.Dir(destPath)); err != nil {
		return fmt.Errorf("failed to ensure dir: %w", err)
	}

	// 使用临时文件路径，避免失败时留下不完整的目标文件
	tmpPath := destPath + ".tmp"

	// 检查临时文件是否存在以确定起始偏移量
	var startByte int64 = 0
	if utils.FileExists(tmpPath) {
		info, err := os.Stat(tmpPath)
		if err == nil {
			startByte = info.Size()
		}
	}

	// 立即上报一次初始进度
	if progressCallback != nil {
		progressCallback(startByte)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 如果有部分文件，设置 Range 头
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		// 如果是 Range Not Satisfiable (416)，可能文件已经下载完成了
		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			// 检查远程文件大小（如果可能），这里简单认为如果本地有数据且 416，可能是下完了
			// 但更严谨的是通过 HEAD 请求确认大小。
			// 暂时如果我们请求 Range 且返回 416，我们假设需要重新下载或者已完成。
			// 简单起见，如果 416，我们尝试删除本地文件重新下载（或者直接报错让重试逻辑处理，但重试可能还是 416）
			// 这里简单报错
			return fmt.Errorf("bad status: %s", resp.Status)
		}
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// 如果服务器返回 200 OK 但我们要的是 range，说明不支持 range或文件已更改。
	// 我们应该截断临时文件；如果是 206 Partial Content，我们追加。
	flags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		startByte = 0 // 重置起始位置
	}

	f, err := os.OpenFile(tmpPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// 包装 Body 以支持进度回调
	reader := &ProgressReader{
		Reader:   resp.Body,
		Current:  startByte,
		Callback: progressCallback,
	}

	// 复制流
	_, err = io.Copy(f, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// 写入成功后原子性地替换目标文件
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
