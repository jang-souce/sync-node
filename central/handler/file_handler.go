package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"sync-node/central/service/oss"
	"sync-node/common/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FileHandler 处理文件上传下载请求
type FileHandler struct {
	ossService oss.OSSService
}

// NewFileHandler 创建 FileHandler 实例
func NewFileHandler(ossService oss.OSSService) *FileHandler {
	return &FileHandler{
		ossService: ossService,
	}
}

// UploadFile 上传文件到 OSS
// POST /api/v1/files/upload
func (h *FileHandler) UploadFile(c *gin.Context) {
	// 1. 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		utils.GetLogger("file").Errorf("Failed to get file from request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
		return
	}

	// 2. 打开文件流
	src, err := file.Open()
	if err != nil {
		utils.GetLogger("file").Errorf("Failed to open uploaded file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// 3. 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	objectName := fmt.Sprintf("uploads/%s%s", uuid.New().String(), ext)

	// 4. 上传到 OSS
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url, err := h.ossService.UploadFile(objectName, src, file.Size, contentType)
	if err != nil {
		utils.GetLogger("file").Errorf("Failed to upload file to OSS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// 5. 返回结果
	c.JSON(http.StatusOK, gin.H{
		"url":         url,
		"object_name": objectName,
		"filename":    file.Filename,
		"size":        file.Size,
	})
}

// GetDownloadURL 获取文件下载链接
// GET /api/v1/files/download?object_name=xxx
func (h *FileHandler) GetDownloadURL(c *gin.Context) {
	objectName := c.Query("object_name")
	if objectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_name is required"})
		return
	}

	// 默认过期时间 1 小时
	expiry := 3600
	if expStr := c.Query("expiry"); expStr != "" {
		if exp, err := strconv.Atoi(expStr); err == nil && exp > 0 {
			expiry = exp
		}
	}

	url, err := h.ossService.GetDownloadURL(objectName, expiry)
	if err != nil {
		utils.GetLogger("file").Errorf("Failed to get download url: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"expires_at": time.Now().Add(time.Duration(expiry) * time.Second).Unix(),
	})
}

// DeleteFile 删除文件
// DELETE /api/v1/files?object_name=xxx
func (h *FileHandler) DeleteFile(c *gin.Context) {
	objectName := c.Query("object_name")
	if objectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_name is required"})
		return
	}

	if err := h.ossService.DeleteFile(objectName); err != nil {
		utils.GetLogger("file").Errorf("Failed to delete file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
