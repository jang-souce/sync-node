package task

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sync-node/common/constant"
	"sync-node/common/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// MockEtcdClientIntegration 模拟 EtcdClient 接口，用于集成测试
type MockEtcdClientIntegration struct {
	mock.Mock
}

func (m *MockEtcdClientIntegration) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	args := m.Called(ctx, key, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientv3.GetResponse), args.Error(1)
}

func (m *MockEtcdClientIntegration) Put(ctx context.Context, key, value string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	args := m.Called(ctx, key, value, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientv3.PutResponse), args.Error(1)
}

func (m *MockEtcdClientIntegration) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	args := m.Called(ctx, key, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clientv3.DeleteResponse), args.Error(1)
}

// MockOSSIntegration 模拟 oss.OSSService 接口，用于集成测试
type MockOSSIntegration struct {
	mock.Mock
}

func (m *MockOSSIntegration) UploadFile(objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	// Mock success
	// 模拟上传成功，返回模拟 URL
	return "http://mock-oss/" + objectName, nil
}

func (m *MockOSSIntegration) GetDownloadURL(objectName string, expiry int) (string, error) {
	// 模拟获取下载链接成功
	return "http://mock-oss/" + objectName, nil
}

func (m *MockOSSIntegration) DeleteFile(objectName string) error {
	// 模拟删除文件成功
	return nil
}

// TestTaskService_Integration 验证完整流程（集成测试）
// 使用 DB（默认 SQLite，如果设置了环境变量则使用 PostgreSQL）
// 并将插入的数据打印到控制台以进行验证。
func TestTaskService_Integration(t *testing.T) {
	// 初始化测试数据库
	db := setupTestDB(t)

	// 初始化 Mock 对象
	mockEtcd := new(MockEtcdClientIntegration)
	mockOSS := new(MockOSSIntegration)
	mockAlert := new(MockAlertService)

	// 启动一个测试 HTTP 服务器，用于模拟文件下载源
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This is a test file content"))
	}))
	defer ts.Close()

	// 模拟 Etcd 获取节点列表（虽然我们会指定目标节点，但为了防止逻辑变更，这里也进行模拟）
	mockEtcd.On("Get", mock.Anything, constant.EtcdNodePrefix, mock.Anything).Return(&clientv3.GetResponse{
		Kvs: []*mvccpb.KeyValue{
			{Key: []byte(constant.EtcdNodePrefix + "node-1"), Value: []byte("")},
		},
	}, nil).Maybe()

	// Mock Etcd Put (Task Dispatch)
	// 模拟 Etcd 任务分发
	mockEtcd.On("Put", mock.Anything, mock.MatchedBy(func(key string) bool {
		// Expect key like /file_sync/task/node-1/UUID
		// 期望 key 格式类似 /file_sync/task/node-1/UUID
		return len(key) > len(constant.EtcdTaskPrefix)
	}), mock.Anything, mock.Anything).Return(nil, nil)

	// 3. Init Service
	// 初始化 TaskService
	service := NewTaskService(db, mockEtcd, mockOSS, mockAlert)

	// 4. Create Task
	// 创建任务请求
	req := &CreateTaskReq{
		Files: []struct {
			FileName     string `json:"fileName"`
			SourceURL    string `json:"sourceUrl"`
			TaskType     int    `json:"taskType"`
			Tag          string `json:"tag"`
			TargetNodeID string `json:"targetNodeId"`
		}{
			{
				FileName:     "test_file_1.txt",
				SourceURL:    ts.URL + "/file1",
				TaskType:     1,
				Tag:          "v1",
				TargetNodeID: "node-1",
			},
			{
				FileName:     "test_file_2.txt",
				SourceURL:    ts.URL + "/file2",
				TaskType:     1,
				Tag:          "v1",
				TargetNodeID: "node-1",
			},
		},
	}

	// 执行创建任务
	mainTask, err := service.CreateTask(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, mainTask)
	assert.Equal(t, 2, mainTask.TotalCount)

	// 5. Verify Data in DB and Print it
	// 验证数据库中的数据并打印
	fmt.Println("\n=== Database Verification ===")

	var storedMainTask model.MainTask
	err = db.Preload("SubTasks").First(&storedMainTask, "id = ?", mainTask.ID).Error
	assert.NoError(t, err)

	fmt.Printf("MainTask ID: %s\n", storedMainTask.ID)
	fmt.Printf("Total Count: %d\n", storedMainTask.TotalCount)
	fmt.Printf("Created At:  %s\n", time.Unix(storedMainTask.CreatedAt, 0).Format(time.RFC3339))
	fmt.Println("-----------------------------")
	fmt.Println("SubTasks:")
	for i, sub := range storedMainTask.SubTasks {
		fmt.Printf("  [%d] ID: %s\n", i, sub.ID)
		fmt.Printf("      NodeID:    %s\n", sub.NodeID)
		fmt.Printf("      FileName:  %s\n", sub.FileName)
		fmt.Printf("      Status:    %s\n", sub.Status)
		fmt.Printf("      SourceURL: %s\n", sub.SourceURL)
		fmt.Printf("      OssURL:    %s\n", sub.OssURL)
		fmt.Println("")
	}
	fmt.Println("=============================")

	// 验证 Etcd 调用是否符合预期
	mockEtcd.AssertExpectations(t)
}
