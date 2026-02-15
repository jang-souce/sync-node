package task

import (
	"context"
	"testing"
	"time"

	"sync-node/common/constant"
	"sync-node/common/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestTaskService_Aggregation 测试主任务状态和进度聚合逻辑
func TestTaskService_Aggregation(t *testing.T) {
	// 1. 初始化环境
	db := setupTestDB(t)
	mockEtcd := new(MockEtcdClient)
	mockOSS := new(MockOSSService)
	mockAlert := new(MockAlertService)

	// Mock 报警服务，防止失败时报错
	mockAlert.On("SendAlert", mock.Anything).Return(nil)

	// 创建 Service
	service := NewTaskService(db, mockEtcd, nil, mockOSS, mockAlert)

	// 2. 准备数据：1个主任务，3个子任务
	mainTaskID := uuid.New().String()
	mainTask := &model.MainTask{
		ID:         mainTaskID,
		TotalCount: 3,
		Status:     constant.TaskStatusPending,
		TotalSize:  3000,
		SyncedSize: 0,
	}
	db.Create(mainTask)

	subTask1 := &model.SubTask{
		ID:         uuid.New().String(),
		MainTaskID: mainTaskID,
		Status:     constant.TaskStatusPending,
		FileSize:   1000,
		SyncedSize: 0,
		NodeID:     "node-1",
	}
	subTask2 := &model.SubTask{
		ID:         uuid.New().String(),
		MainTaskID: mainTaskID,
		Status:     constant.TaskStatusPending,
		FileSize:   1000,
		SyncedSize: 0,
		NodeID:     "node-2",
	}
	subTask3 := &model.SubTask{
		ID:         uuid.New().String(),
		MainTaskID: mainTaskID,
		Status:     constant.TaskStatusPending,
		FileSize:   1000,
		SyncedSize: 0,
		NodeID:     "node-3",
	}
	db.Create(subTask1)
	db.Create(subTask2)
	db.Create(subTask3)

	ctx := context.Background()

	// 3. 场景测试

	// 场景 A: 子任务1 开始下载 -> 主任务应变为 Downloading
	err := service.UpdateTaskStatus(ctx, subTask1.ID, constant.TaskStatusDownloading, "", 100)
	assert.NoError(t, err)

	// 等待异步聚合完成
	time.Sleep(100 * time.Millisecond)

	var mt model.MainTask
	db.First(&mt, "id = ?", mainTaskID)
	assert.Equal(t, constant.TaskStatusDownloading, mt.Status, "MainTask should be Downloading")
	assert.Equal(t, int64(100), mt.SyncedSize, "SyncedSize should be 100")

	// 场景 B: 子任务2 失败 -> 主任务应变为 Failed
	err = service.UpdateTaskStatus(ctx, subTask2.ID, constant.TaskStatusFailed, "some error", 0)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	db.First(&mt, "id = ?", mainTaskID)
	assert.Equal(t, constant.TaskStatusFailed, mt.Status, "MainTask should be Failed")
	assert.Equal(t, int64(100), mt.SyncedSize, "SyncedSize should still be 100")

	// 场景 C: 子任务3 完成 -> 主任务仍为 Failed (因为子任务2失败)
	// 假设 TaskType 不是 2 (删除OSS)，避免调用 OSS Delete
	err = service.UpdateTaskStatus(ctx, subTask3.ID, constant.TaskStatusCompleted, "", 1000)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	db.First(&mt, "id = ?", mainTaskID)
	assert.Equal(t, constant.TaskStatusFailed, mt.Status, "MainTask should still be Failed due to subTask2")
	assert.Equal(t, int64(1100), mt.SyncedSize, "SyncedSize should be 100 + 1000 = 1100")

	// 场景 D: 子任务2 重试并转为 Downloading -> 主任务应变为 Downloading (不再Failed)
	err = service.UpdateTaskStatus(ctx, subTask2.ID, constant.TaskStatusDownloading, "", 500)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	db.First(&mt, "id = ?", mainTaskID)
	assert.Equal(t, constant.TaskStatusDownloading, mt.Status, "MainTask should recover to Downloading")
	assert.Equal(t, int64(1600), mt.SyncedSize, "SyncedSize should be 100 + 1000 + 500 = 1600")

	// 场景 E: 所有子任务完成 -> 主任务应变为 Completed
	service.UpdateTaskStatus(ctx, subTask1.ID, constant.TaskStatusCompleted, "", 1000)
	service.UpdateTaskStatus(ctx, subTask2.ID, constant.TaskStatusCompleted, "", 1000)

	time.Sleep(100 * time.Millisecond)

	db.First(&mt, "id = ?", mainTaskID)
	assert.Equal(t, constant.TaskStatusCompleted, mt.Status, "MainTask should be Completed")
	assert.Equal(t, int64(3000), mt.SyncedSize, "SyncedSize should be 3000")
}
