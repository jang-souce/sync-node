package task

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync-node/central/service/alert"
	"sync-node/central/service/oss"
	"sync-node/common/constant"
	"sync-node/common/lock"
	"sync-node/common/model"
	"sync-node/common/utils"
	"time"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

// EtcdClient 定义 TaskService 需要的 Etcd 操作接口
type EtcdClient interface {
	Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
	Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error)
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	GrantLease(ctx context.Context, ttl int64) (clientv3.LeaseID, error)
}

// TaskService 任务管理服务，处理任务的创建、查询和分发
type TaskService struct {
	db           *gorm.DB
	etcd         EtcdClient
	locker       lock.Locker
	oss          oss.OSSService
	alertService alert.AlertService
}

// NewTaskService 创建 TaskService 实例
func NewTaskService(db *gorm.DB, etcd EtcdClient, locker lock.Locker, oss oss.OSSService, alertService alert.AlertService) *TaskService {
	return &TaskService{
		db:           db,
		etcd:         etcd,
		locker:       locker,
		oss:          oss,
		alertService: alertService,
	}
}

// CreateTaskReq 创建任务请求参数
type CreateTaskReq struct {
	Files []struct {
		FileName     string `json:"fileName"`
		SourceURL    string `json:"sourceUrl"`
		TaskType     int    `json:"taskType"`
		Tag          string `json:"tag"`
		TargetNodeID string `json:"targetNodeId"` // 可选：指定目标节点ID，为空则随机分发
	} `json:"files"`
}

// CreateTask 创建新任务 (批量)
// 1. 获取在线节点
// 2. 处理文件 (下载 -> 计算哈希 -> 上传 OSS)
// 3. 创建数据库记录 (MainTask, SubTask)
// 4. 将任务分发到 Etcd
func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskReq) (*model.MainTask, error) {
	if len(req.Files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	// 1. 获取在线节点列表 (用于随机分发) - 懒加载
	var onlineNodes []string
	var fetchNodesOnce bool
	const etcdTimeout = 5 * time.Second

	getOnlineNodes := func() ([]string, error) {
		if fetchNodesOnce {
			return onlineNodes, nil
		}
		// 从 Etcd 获取所有在线节点
		tCtx, cancel := context.WithTimeout(ctx, etcdTimeout)
		defer cancel()

		resp, err := s.etcd.Get(tCtx, constant.EtcdNodePrefix, clientv3.WithPrefix())
		if err != nil {
			return nil, fmt.Errorf("failed to get online nodes: %w", err)
		}
		for _, kv := range resp.Kvs {
			// Key 格式: /file_sync/node/{node_id}
			nodeID := string(kv.Key)[len(constant.EtcdNodePrefix):]
			onlineNodes = append(onlineNodes, nodeID)
		}
		fetchNodesOnce = true
		return onlineNodes, nil
	}

	// 2. 创建 MainTask
	mainTask := &model.MainTask{
		ID:         uuid.New().String(),
		TotalCount: len(req.Files),
		Status:     constant.TaskStatusPending,
	}

	// 3. 处理文件并生成 SubTasks
	var subTasks []model.SubTask

	// 缓存已处理的文件信息，避免重复下载上传
	// key: sourceURL, value: (fileSize, fileHash, ossURL, ossKey)
	processedFiles := make(map[string]struct {
		Size int64
		Hash string
		URL  string
		Key  string
	})

	for _, fileReq := range req.Files {
		// 确定目标节点
		targetNodeID := fileReq.TargetNodeID
		if targetNodeID == "" {
			// 随机分发给在线节点
			nodes, err := getOnlineNodes()
			if err != nil {
				return nil, err
			}
			if len(nodes) == 0 {
				return nil, fmt.Errorf("no available online nodes")
			}
			targetNodeID = nodes[rand.Intn(len(nodes))]
		}

		// 检查缓存，避免重复处理相同文件
		info, ok := processedFiles[fileReq.SourceURL]
		if !ok {
			// 下载并上传 OSS
			size, hash, url, key, err := s.processFile(fileReq.SourceURL, fileReq.FileName)
			if err != nil {
				return nil, fmt.Errorf("failed to process file %s: %w", fileReq.FileName, err)
			}
			info = struct {
				Size int64
				Hash string
				URL  string
				Key  string
			}{size, hash, url, key}
			processedFiles[fileReq.SourceURL] = info
		}

		// 累加总大小
		mainTask.TotalSize += info.Size

		// 创建子任务
		subTask := model.SubTask{
			ID:         uuid.New().String(),
			MainTaskID: mainTask.ID,
			NodeID:     targetNodeID,
			FileName:   fileReq.FileName,
			FileSize:   info.Size,
			FileHash:   info.Hash,
			OssURL:     info.URL,
			OssKey:     info.Key,
			SourceURL:  fileReq.SourceURL,
			Tag:        fileReq.Tag,
			TaskType:   fileReq.TaskType,
			Status:     constant.TaskStatusPending,
			CreatedAt:  time.Now().Unix(),
		}
		subTasks = append(subTasks, subTask)
	}

	mainTask.SubTasks = subTasks

	// 4. 开启事务存 DB
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(mainTask).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create main task: %w", err)
	}

	// 5. 分布式锁 + Put 到 Etcd
	// 使用分布式锁防止重复下发（虽然这里 ID 是新的，但在高并发场景下，或者重试逻辑中可能有用）
	// 这里更典型的用法可能是对某个资源（如 Node 任务队列）加锁
	// 但根据需求 "任务下发使用分布式锁，防止重复下发"，我们可以对 MainTaskID 加锁
	etcdCtx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	mutex, err := s.locker.Lock(etcdCtx, constant.EtcdLockPrefix+mainTask.ID, 10)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to acquire mutex: %w", err)
	}
	// 使用新的 Context 确保 Unlock 能执行，即使 etcdCtx 超时
	defer func() {
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), etcdTimeout)
		defer unlockCancel()
		mutex.Unlock(unlockCtx)
	}()

	// 创建租约，确保任务数据不会永久滞留在 Etcd 中
	// 默认 24 小时过期，给予足够的时间让节点获取
	leaseID, err := s.etcd.GrantLease(etcdCtx, 3600*24)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to grant lease: %w", err)
	}

	// Key: /file_sync/task/{node_id}/{sub_task_id}
	// Agent 需要监听 /file_sync/task/{my_node_id}/
	for _, subTask := range subTasks {
		taskJSON, err := json.Marshal(subTask)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to marshal sub task: %w", err)
		}

		// Etcd Key 包含 NodeID，方便 Agent 只监听自己的任务
		etcdKey := fmt.Sprintf("%s%s/%s", constant.EtcdTaskPrefix, subTask.NodeID, subTask.ID)
		if _, err := s.etcd.Put(etcdCtx, etcdKey, string(taskJSON), clientv3.WithLease(leaseID)); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to put task to etcd: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return mainTask, nil
}

// processFile 下载文件并上传到 OSS
// 返回: fileSize, fileHash, ossURL, objectKey, error
func (s *TaskService) processFile(sourceURL, fileName string) (int64, string, string, string, error) {
	// 优化：检查 SourceURL 是否已是当前 OSS 的文件
	if key, ok := s.oss.ParseObjectKeyFromURL(sourceURL); ok {
		// 验证文件存在并获取元数据
		size, hash, err := s.oss.GetObjectInfo(key)
		if err == nil {
			// 文件存在，直接复用
			// 生成一个新的签名 URL 给 Node 使用
			url, err := s.oss.GetDownloadURL(key, 3600*24*365)
			if err != nil {
				// 如果获取签名失败，降级到下载逻辑
				utils.GetLogger("task").Warnf("Optimization hit but failed to get download url for %s: %v", key, err)
			} else {
				utils.GetLogger("task").Infof("Optimized: SourceURL is already in OSS, reusing key: %s", key)
				return size, hash, url, key, nil
			}
		} else {
			utils.GetLogger("task").Warnf("Optimization failed: Object %s not found in OSS or error: %v", key, err)
		}
	} else {
		utils.GetLogger("task").Infof("Optimization skipped: SourceURL %s not recognized as internal OSS", sourceURL)
	}

	// 下载临时文件
	tmpFile, err := os.CreateTemp("", "sync-task-*")
	if err != nil {
		return 0, "", "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := http.Get(sourceURL)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("failed to download source file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", "", "", fmt.Errorf("failed to download source file: status code %d", resp.StatusCode)
	}

	// 计算哈希并写入临时文件
	hash := md5.New()
	writer := io.MultiWriter(tmpFile, hash)
	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("failed to save temp file: %w", err)
	}

	fileHash := hex.EncodeToString(hash.Sum(nil))

	// 上传到 OSS
	// 重置文件指针
	tmpFile.Seek(0, 0)
	// 使用 hash 作为文件名一部分避免冲突？或者 uuid? 这里简单用 uuid 目录
	objectKey := fmt.Sprintf("tasks/%s/%s", uuid.New().String(), fileName)
	_, err = s.oss.UploadFile(objectKey, tmpFile, written, "application/octet-stream")
	if err != nil {
		return 0, "", "", "", fmt.Errorf("failed to upload to oss: %w", err)
	}

	// 获取下载链接 (1年有效期)
	url, err := s.oss.GetDownloadURL(objectKey, 3600*24*365)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("failed to get oss url: %w", err)
	}

	return written, fileHash, url, objectKey, nil
}

// GetTasks 获取主任务列表
func (s *TaskService) GetTasks(page, pageSize int) ([]model.MainTask, int64, error) {
	var tasks []model.MainTask
	var total int64
	offset := (page - 1) * pageSize

	if err := s.db.Model(&model.MainTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Preload("SubTasks").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetTaskByID 获取主任务详情
func (s *TaskService) GetTaskByID(id string) (*model.MainTask, error) {
	var task model.MainTask
	if err := s.db.Where("id = ?", id).Preload("SubTasks").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// RepublishPendingTasks 重新发布处于 Pending 状态但 Etcd 中丢失的任务
func (s *TaskService) RepublishPendingTasks(ctx context.Context) error {
	var subTasks []model.SubTask
	// 查找所有 Pending 状态的子任务
	if err := s.db.Where("status = ?", constant.TaskStatusPending).Find(&subTasks).Error; err != nil {
		return fmt.Errorf("failed to query pending tasks: %w", err)
	}

	if len(subTasks) == 0 {
		return nil
	}

	fmt.Printf("Found %d pending tasks, checking Etcd status...\n", len(subTasks))

	// 创建一个新的租约 (24h)
	leaseID, err := s.etcd.GrantLease(ctx, 3600*24)
	if err != nil {
		return fmt.Errorf("failed to grant lease for republication: %w", err)
	}

	republishedCount := 0
	for _, subTask := range subTasks {
		etcdKey := fmt.Sprintf("%s%s/%s", constant.EtcdTaskPrefix, subTask.NodeID, subTask.ID)

		// 检查 Etcd 中是否存在
		resp, err := s.etcd.Get(ctx, etcdKey)
		if err != nil {
			fmt.Printf("Failed to check etcd key %s: %v\n", etcdKey, err)
			continue
		}

		// 如果不存在，重新发布
		if resp.Count == 0 {
			taskJSON, err := json.Marshal(subTask)
			if err != nil {
				fmt.Printf("Failed to marshal task %s: %v\n", subTask.ID, err)
				continue
			}

			if _, err := s.etcd.Put(ctx, etcdKey, string(taskJSON), clientv3.WithLease(leaseID)); err != nil {
				fmt.Printf("Failed to republish task %s to etcd: %v\n", subTask.ID, err)
				continue
			}
			republishedCount++
			fmt.Printf("Republished task %s to node %s\n", subTask.ID, subTask.NodeID)
		}
	}

	if republishedCount > 0 {
		fmt.Printf("Successfully republished %d tasks\n", republishedCount)
	}
	return nil
}

// DeleteTask 删除主任务 (逻辑删除)
func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	// 1. 查出所有 SubTask
	var subTasks []model.SubTask
	if err := s.db.Where("main_task_id = ?", id).Find(&subTasks).Error; err != nil {
		return err
	}

	// 2. DB 软删除 MainTask
	if err := s.db.Where("id = ?", id).Delete(&model.MainTask{}).Error; err != nil {
		return err
	}
	// 软删除 SubTasks
	if err := s.db.Where("main_task_id = ?", id).Delete(&model.SubTask{}).Error; err != nil {
		return err
	}

	// 3. Etcd 删除
	for _, sub := range subTasks {
		etcdKey := fmt.Sprintf("%s%s/%s", constant.EtcdTaskPrefix, sub.NodeID, sub.ID)
		s.etcd.Delete(ctx, etcdKey)
	}

	return nil
}
