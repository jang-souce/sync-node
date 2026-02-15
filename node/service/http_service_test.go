package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"sync-node/node/config"
)

func TestDownloadFileSuccessAndRename(t *testing.T) {
	// 准备配置
	config.GlobalConfig = &config.Config{}
	config.GlobalConfig.Node.RetryMax = 1
	config.GlobalConfig.Node.RetryDelay = 1

	// 创建测试 HTTP 服务器
	content := []byte("hello world")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 简单返回内容，不支持 Range
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer ts.Close()

	svc := NewHttpService()
	dir := t.TempDir()
	dest := filepath.Join(dir, "file.txt")
	var progress int64

	ctx := context.Background()
	if err := svc.DownloadFile(ctx, ts.URL, dest, func(c int64) {
		atomic.StoreInt64(&progress, c)
	}); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	// 验证目标文件存在且临时文件不存在
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("dest file not exists: %v", err)
	}
	if _, err := os.Stat(dest + ".tmp"); err == nil {
		t.Fatalf("temp file should be removed after success")
	}
	// 验证内容
	data, _ := os.ReadFile(dest)
	if string(data) != string(content) {
		t.Fatalf("content mismatch: %s", string(data))
	}
	// 进度至少达到内容长度
	if atomic.LoadInt64(&progress) < int64(len(content)) {
		t.Fatalf("progress not reached file size")
	}
}

func TestDownloadFileTimeout(t *testing.T) {
	config.GlobalConfig = &config.Config{}
	config.GlobalConfig.Node.RetryMax = 1
	config.GlobalConfig.Node.RetryDelay = 1

	// 服务端延迟，模拟超时
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow response"))
	}))
	defer ts.Close()

	svc := NewHttpService()
	dir := t.TempDir()
	dest := filepath.Join(dir, "slow.txt")

	// 使用已取消的上下文，立即触发超时/取消
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.DownloadFile(ctx, ts.URL, dest, nil)
	if err == nil {
		t.Fatalf("expected error due to context cancel")
	}
}
