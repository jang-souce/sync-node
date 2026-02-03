package task

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"sync-node/common/constant"
	"sync-node/common/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 根据环境变量初始化测试数据库
// 如果设置了 TEST_DB_DSN，则使用 PostgreSQL；否则使用 SQLite 内存数据库
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("TEST_DB_DSN")
	var db *gorm.DB
	var err error

	if dsn != "" {
		t.Log("Using PostgreSQL for testing...")
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to connect to postgres: %v", err)
		}
	} else {
		t.Log("Using SQLite (In-Memory) for testing...")
		// 使用不同的内存数据库名避免冲突，或者使用 file::memory:?cache=shared
		// 这里为了简单统一，使用 shared cache
		db, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to connect to sqlite: %v", err)
		}
	}

	// 清理旧表以确保环境纯净 (无论是 Postgres 还是 SQLite)
	err = db.Migrator().DropTable(&model.MainTask{}, &model.SubTask{})
	if err != nil {
		t.Fatalf("failed to drop tables: %v", err)
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&model.MainTask{}, &model.SubTask{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

// MockEtcdClient 模拟 Etcd 客户端，用于单元测试
type MockEtcdClient struct {
	mock.Mock
}

// Put 模拟 Etcd 的 Put 操作
func (m *MockEtcdClient) Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	args := m.Called(ctx, key, value, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientv3.PutResponse), args.Error(1)
}

// Delete 模拟 Etcd 的 Delete 操作
func (m *MockEtcdClient) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	args := m.Called(ctx, key, opts)
	return args.Get(0).(*clientv3.DeleteResponse), args.Error(1)
}

// Get 模拟 Etcd 的 Get 操作
func (m *MockEtcdClient) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	args := m.Called(ctx, key, opts)
	return args.Get(0).(*clientv3.GetResponse), args.Error(1)
}

// MockOSSService 模拟 OSS 服务，用于单元测试
type MockOSSService struct {
	mock.Mock
}

// UploadFile 模拟文件上传
func (m *MockOSSService) UploadFile(objectName string, reader io.Reader, objectSize int64, contentType string) (string, error) {
	args := m.Called(objectName, reader, objectSize, contentType)
	return args.String(0), args.Error(1)
}

// DownloadFile 模拟文件下载
func (m *MockOSSService) DownloadFile(objectName string, filePath string) error {
	args := m.Called(objectName, filePath)
	return args.Error(0)
}

// GetDownloadURL 模拟获取下载链接
func (m *MockOSSService) GetDownloadURL(objectName string, expiration int) (string, error) {
	args := m.Called(objectName, expiration)
	return args.String(0), args.Error(1)
}

// MockAlertService 模拟报警服务
type MockAlertService struct {
	mock.Mock
}

// SendAlert 模拟发送报警
func (m *MockAlertService) SendAlert(param map[string]string) error {
	args := m.Called(param)
	return args.Error(0)
}

// TestTaskService_CreateTask 测试创建任务的逻辑
func TestTaskService_CreateTask(t *testing.T) {
	// 1. 设置模拟文件下载服务器 (模拟源文件地址)
	// 模拟一个 HTTP 服务器，返回 "test content" 作为文件内容
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// 2. 设置内存数据库 (SQLite 或 Postgres)
	// 初始化测试数据库
	db := setupTestDB(t)

	// 3. 设置 Mock 对象 (Etcd 和 OSS)
	// 创建 Etcd 和 OSS 的 Mock 对象
	mockEtcd := new(MockEtcdClient)
	mockOSS := new(MockOSSService)

	// 设置 OSS Mock 期望: 期望 UploadFile 被调用，并返回模拟 URL
	mockOSS.On("UploadFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("http://mock-oss/test", nil)
	// 设置 OSS Mock 期望: 期望 GetDownloadURL 被调用，并返回签名 URL
	mockOSS.On("GetDownloadURL", mock.Anything, mock.Anything).Return("http://signed-url", nil)

	// 设置 Etcd Mock 期望: 期望 Put 被调用 (任务分发)，并返回成功
	mockEtcd.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	// 4. 创建 TaskService 实例
	service := NewTaskService(db, mockEtcd, mockOSS, nil)

	// 5. 构造测试请求数据
	req := &CreateTaskReq{
		Files: []struct {
			FileName     string `json:"fileName"`
			SourceURL    string `json:"sourceUrl"`
			TaskType     int    `json:"taskType"`
			Tag          string `json:"tag"`
			TargetNodeID string `json:"targetNodeId"`
		}{
			{
				FileName:     "test.txt",
				SourceURL:    server.URL,
				TaskType:     1,
				Tag:          "",
				TargetNodeID: "node-1", // 指定目标节点，避免触发 Etcd Get 获取节点列表逻辑
			},
		},
	}

	// 执行创建任务
	createdTask, err := service.CreateTask(context.Background(), req)

	// 6. 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, createdTask)
	assert.Equal(t, 1, createdTask.TotalCount)
	assert.Len(t, createdTask.SubTasks, 1)
	assert.Equal(t, "test.txt", createdTask.SubTasks[0].FileName)
	assert.Equal(t, "http://signed-url", createdTask.SubTasks[0].OssURL)
	assert.NotEmpty(t, createdTask.ID)

	// 验证数据库数据
	var dbTask model.MainTask
	err = db.First(&dbTask, "id = ?", createdTask.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, createdTask.ID, dbTask.ID)
	var dbSub model.SubTask
	err = db.First(&dbSub, "main_task_id = ?", createdTask.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "test.txt", dbSub.FileName)

	// 验证所有 Mock 期望是否满足
	mockOSS.AssertExpectations(t)
	mockEtcd.AssertExpectations(t)
}

// TestTaskService_GetTasks 测试获取任务列表 (分页查询)
func TestTaskService_GetTasks(t *testing.T) {
	// 设置内存数据库
	db := setupTestDB(t)

	// 预置测试数据
	// 插入两条主任务记录
	db.Create(&model.MainTask{ID: "1", CreatedAt: 100})
	db.Create(&model.MainTask{ID: "2", CreatedAt: 200})

	// 创建 Service (查询不需要 Etcd/OSS)
	service := NewTaskService(db, nil, nil, nil)

	// 测试列表查询
	// 查询第 1 页，每页 10 条
	tasks, total, err := service.GetTasks(1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "2", tasks[0].ID) // 验证排序: 按 CreatedAt 倒序

	// 测试分页逻辑 (虽然这里数据少，但逻辑一致)
	tasks, total, err = service.GetTasks(1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, tasks, 2)
}

// TestTaskService_DeleteTask 测试删除任务
func TestTaskService_DeleteTask(t *testing.T) {
	// 设置内存数据库
	db := setupTestDB(t)
	var err error

	// 预置数据: 主任务和子任务
	main := &model.MainTask{ID: "task-to-delete"}
	db.Create(main)
	sub := &model.SubTask{ID: "sub-1", MainTaskID: main.ID, NodeID: "node-1"}
	db.Create(&sub)

	// 设置 Mock
	mockEtcd := new(MockEtcdClient)
	// 设置 Etcd Delete 期望: 删除任务时会清理 Etcd 中的任务 key
	etcdKey := fmt.Sprintf("%s%s/%s", constant.EtcdTaskPrefix, sub.NodeID, sub.ID)
	mockEtcd.On("Delete", mock.Anything, etcdKey, mock.Anything).Return(&clientv3.DeleteResponse{Deleted: 1}, nil)

	service := NewTaskService(db, mockEtcd, nil, nil)

	// 执行删除
	err = service.DeleteTask(context.Background(), "task-to-delete")

	// 验证结果
	assert.NoError(t, err)

	// 验证数据库 (软删除检查)
	var count int64
	db.Model(&model.MainTask{}).Where("id = ?", "task-to-delete").Count(&count)
	assert.Equal(t, int64(0), count) // 普通查询应查不到 (Count 为 0)

	var deletedTask model.MainTask
	// 使用 Unscoped 查询已被软删除的记录
	err = db.Unscoped().Where("id = ?", "task-to-delete").First(&deletedTask).Error
	assert.NoError(t, err)
	assert.NotZero(t, deletedTask.DeletedAt) // DeletedAt 应该有值

	// 验证 Etcd Mock
	mockEtcd.AssertExpectations(t)
}
