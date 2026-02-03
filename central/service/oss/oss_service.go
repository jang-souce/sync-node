package oss

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"sync-node/central/config"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// OSSService 定义对象存储服务接口
type OSSService interface {
	// UploadFile 上传文件
	// objectName: 对象名（包含路径）
	// reader: 文件内容
	// size: 文件大小
	// contentType: 文件类型
	UploadFile(objectName string, reader io.Reader, size int64, contentType string) (string, error)

	// GetDownloadURL 获取文件下载链接
	// objectName: 对象名
	// expiry: 过期时间（秒）
	GetDownloadURL(objectName string, expiry int) (string, error)
}

// AliyunOSS 阿里云 OSS 实现
type AliyunOSS struct {
	cfg *config.Config
}

// MinIOOSS MinIO 实现
type MinIOOSS struct {
	client     *minio.Client
	bucketName string
}

// NewOSSService 创建 OSS 服务实例
func NewOSSService(cfg *config.Config) (OSSService, error) {
	// 默认为 aliyun，除非明确指定 minio
	if strings.ToLower(cfg.OSS.Provider) == "minio" {
		return newMinIOOSS(cfg)
	}
	return newAliyunOSS(cfg)
}

func newAliyunOSS(cfg *config.Config) (*AliyunOSS, error) {
	// 验证配置有效性 (尝试连接一次)
	client, err := oss.New(cfg.OSS.Endpoint, cfg.OSS.AccessKeyID, cfg.OSS.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun oss client: %w", err)
	}
	// 验证 Bucket 是否存在
	_, err = client.Bucket(cfg.OSS.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket %s: %w", cfg.OSS.BucketName, err)
	}

	return &AliyunOSS{cfg: cfg}, nil
}

func (s *AliyunOSS) getBucket() (*oss.Bucket, error) {
	// 每次调用都重新创建 Client，以支持配置热更新
	// 注意：在高并发场景下可能需要优化（如缓存 Client 并监听配置变更）
	client, err := oss.New(s.cfg.OSS.Endpoint, s.cfg.OSS.AccessKeyID, s.cfg.OSS.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun oss client: %w", err)
	}

	bucket, err := client.Bucket(s.cfg.OSS.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket %s: %w", s.cfg.OSS.BucketName, err)
	}
	return bucket, nil
}

func (s *AliyunOSS) UploadFile(objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	bucket, err := s.getBucket()
	if err != nil {
		return "", err
	}

	// 阿里云 SDK 的 PutObject 默认接受 reader
	// 注意：aliyun oss sdk 对于 io.Reader 不会自动计算长度，建议使用 PutObject 结合 Option
	// 但这里我们简单处理
	err = bucket.PutObject(objectName, reader, oss.ContentType(contentType))
	if err != nil {
		return "", err
	}
	// 返回完整 URL (仅供参考，实际下载通常用签名 URL)
	// 格式: https://bucket.endpoint/object
	// 这里不方便拼装，直接返回 objectName 或者空
	return objectName, nil
}

func (s *AliyunOSS) GetDownloadURL(objectName string, expiry int) (string, error) {
	bucket, err := s.getBucket()
	if err != nil {
		return "", err
	}

	// 生成签名 URL，允许在有效期内访问私有对象
	signedURL, err := bucket.SignURL(objectName, oss.HTTPGet, int64(expiry))
	if err != nil {
		return "", err
	}
	return signedURL, nil
}

func newMinIOOSS(cfg *config.Config) (*MinIOOSS, error) {
	// MinIO 需要去掉 endpoint 中的 http:// 前缀
	endpoint := strings.TrimPrefix(cfg.OSS.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	useSSL := strings.HasPrefix(cfg.OSS.Endpoint, "https://")

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.OSS.AccessKeyID, cfg.OSS.AccessKeySecret, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinIOOSS{
		client:     minioClient,
		bucketName: cfg.OSS.BucketName,
	}, nil
}

func (s *MinIOOSS) UploadFile(objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	info, err := s.client.PutObject(context.Background(), s.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	// 返回 Object Name
	return info.Key, nil
}

func (s *MinIOOSS) GetDownloadURL(objectName string, expiry int) (string, error) {
	reqParams := make(url.Values)
	// 生成预签名 URL
	presignedURL, err := s.client.PresignedGetObject(context.Background(), s.bucketName, objectName, time.Duration(expiry)*time.Second, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}
