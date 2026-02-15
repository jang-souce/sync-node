package oss

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"sync-node/central/config"
	"sync-node/common/utils"

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

	// DeleteFile 删除文件
	// objectName: 对象名
	DeleteFile(objectName string) error

	// GetObjectInfo 获取对象元数据 (是否存在，大小，ETag)
	// 返回: size, etag, error
	GetObjectInfo(objectName string) (int64, string, error)

	// ParseObjectKeyFromURL 从 URL 中解析 ObjectKey
	// 如果 URL 属于当前 OSS Bucket，返回 (key, true)，否则返回 ("", false)
	ParseObjectKeyFromURL(url string) (string, bool)
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
	// 设置 ACL 为 Private，确保安全性
	err = bucket.PutObject(objectName, reader, oss.ContentType(contentType), oss.ACL(oss.ACLPrivate))
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

func (s *AliyunOSS) DeleteFile(objectName string) error {
	bucket, err := s.getBucket()
	if err != nil {
		return err
	}
	return bucket.DeleteObject(objectName)
}

func (s *AliyunOSS) GetObjectInfo(objectName string) (int64, string, error) {
	bucket, err := s.getBucket()
	if err != nil {
		return 0, "", err
	}

	props, err := bucket.GetObjectDetailedMeta(objectName)
	if err != nil {
		return 0, "", err
	}

	size := props.Get("Content-Length")
	etag := strings.Trim(props.Get("ETag"), "\"")
	etag = strings.ToLower(etag)

	var sizeInt int64
	fmt.Sscanf(size, "%d", &sizeInt)

	return sizeInt, etag, nil
}

func (s *AliyunOSS) ParseObjectKeyFromURL(rawURL string) (string, bool) {
	// 简单解析逻辑：检查是否包含 Endpoint 和 BucketName
	// 阿里云 URL 格式通常为:
	// 1. https://<BucketName>.<Endpoint>/<ObjectKey> (默认)
	// 2. https://<Endpoint>/<BucketName>/<ObjectKey> (Path style)

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	// 提取 Host 和 Path
	host := u.Host
	// 针对 Path 中包含 %2F 的情况，u.Path 可能已经被解码，或者未被解码
	// 但在 OSS 中，如果 URL 是 .../test-data%2Fsmall_5.txt
	// url.Parse 得到的 Path 可能是 /test-data/small_5.txt (已解码)
	// 我们需要确保获取到的是解码后的路径
	path := strings.TrimPrefix(u.Path, "/")

	// 调试日志：打印解析到的 Host 和 Path
	utils.GetLogger("oss").Infof("ParseObjectKeyFromURL: RawURL=%s, Host=%s, Path=%s, Endpoint=%s, BucketName=%s", rawURL, host, path, s.cfg.OSS.Endpoint, s.cfg.OSS.BucketName)

	// 检查 Endpoint
	// 注意：s.cfg.OSS.Endpoint 可能包含 "https://" 前缀，而 u.Host 通常不包含 scheme
	// 例如：Host=yang-sync.oss-cn-beijing.aliyuncs.com, Endpoint=https://oss-cn-beijing.aliyuncs.com
	// 需要处理掉 scheme 才能正确比较
	endpoint := s.cfg.OSS.Endpoint
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	if !strings.Contains(host, endpoint) {
		utils.GetLogger("oss").Infof("ParseObjectKeyFromURL: Host %s does not contain Endpoint %s (processed)", host, endpoint)
		return "", false
	}

	// 情况 1: Host = <BucketName>.<Endpoint>
	expectedHost := fmt.Sprintf("%s.%s", s.cfg.OSS.BucketName, endpoint)
	if host == expectedHost {
		utils.GetLogger("oss").Infof("ParseObjectKeyFromURL: Host matches expectedHost %s. Returning path: %s", expectedHost, path)
		return path, true
	} else {
		utils.GetLogger("oss").Infof("ParseObjectKeyFromURL: Host %s does not match ExpectedHost %s", host, expectedHost)
	}

	// 情况 2: Host = <Endpoint> 且 Path 以 <BucketName>/ 开头
	if host == endpoint {
		prefix := s.cfg.OSS.BucketName + "/"
		if strings.HasPrefix(path, prefix) {
			return strings.TrimPrefix(path, prefix), true
		}
	}

	return "", false
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

func (s *MinIOOSS) DeleteFile(objectName string) error {
	return s.client.RemoveObject(context.Background(), s.bucketName, objectName, minio.RemoveObjectOptions{})
}

func (s *MinIOOSS) GetObjectInfo(objectName string) (int64, string, error) {
	info, err := s.client.StatObject(context.Background(), s.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return 0, "", err
	}
	return info.Size, info.ETag, nil
}

func (s *MinIOOSS) ParseObjectKeyFromURL(rawURL string) (string, bool) {
	// MinIO URL 格式通常为: http(s)://<Endpoint>/<BucketName>/<ObjectKey>
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	path := strings.TrimPrefix(u.Path, "/")
	prefix := s.bucketName + "/"
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix), true
	}
	return "", false
}
