package task

import (
	"context"
	"testing"
	"time"

	"sync-node/common/constant"
	"sync-node/common/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestTaskService_UpdateTaskStatus 测试更新任务状态的逻辑
func TestTaskService_UpdateTaskStatus(t *testing.T) {
	// 1. 设置内存数据库 (SQLite)
	// 初始化测试数据库
	db := setupTestDB(t)
	var err error

	// 2. 预置数据: 创建一个待处理的子任务
	// 先创建主任务以满足外键约束
	mainTask := &model.MainTask{
		ID:        "main-task-1",
		CreatedAt: time.Now().Unix(),
	}
	db.Create(mainTask)

	// 创建子任务，初始状态为 Pending
	subTask := &model.SubTask{
		ID:         "sub-task-1",
		NodeID:     "node-1",
		Status:     constant.TaskStatusPending,
		CreatedAt:  time.Now().Unix(),
		MainTaskID: "main-task-1",
	}
	db.Create(subTask)

	// 3. 设置 Mock 对象
	mockAlert := new(MockAlertService)
	// 设置 Alert Mock 期望: 当任务状态更新为失败时，期望调用 SendAlert 发送报警
	mockAlert.On("SendAlert", mock.Anything).Return(nil)

	// 创建 Service 实例 (只需要 DB 和 AlertService)
	service := NewTaskService(db, nil, nil, mockAlert)

	// 4. 测试场景: 成功更新状态
	// 将状态更新为 Downloading，并设置同步进度为 100
	err = service.UpdateTaskStatus(context.Background(), "sub-task-1", constant.TaskStatusDownloading, "", 100)
	assert.NoError(t, err)

	// 验证数据库更新结果
	var updated model.SubTask
	db.First(&updated, "id = ?", "sub-task-1")
	assert.Equal(t, constant.TaskStatusDownloading, updated.Status)
	assert.Equal(t, int64(100), updated.SyncedSize)

	// 5. 测试场景: 更新状态为失败 (触发报警)
	// 将状态更新为 Failed，并附带错误信息 "download error"
	err = service.UpdateTaskStatus(context.Background(), "sub-task-1", constant.TaskStatusFailed, "download error", 100)
	assert.NoError(t, err)

	// 验证数据库更新结果
	db.First(&updated, "id = ?", "sub-task-1")
	assert.Equal(t, constant.TaskStatusFailed, updated.Status)
	assert.Equal(t, "download error", updated.ErrorMsg)

	// 等待异步报警发送 (因为代码中可能是异步调用报警)
	time.Sleep(100 * time.Millisecond)
	// 验证报警是否被调用
	mockAlert.AssertExpectations(t)
}
