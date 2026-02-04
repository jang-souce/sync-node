package task

import (
	"context"
	"fmt"
	"time"

	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/utils"
)

// UpdateTaskStatus 更新任务状态（供 Agent 上报）
// subTaskID: 子任务 ID
// status: 任务新状态
// errorMsg: 错误信息（如果有）
// syncedSize: 已同步大小
func (s *TaskService) UpdateTaskStatus(ctx context.Context, subTaskID string, status string, errorMsg string, syncedSize int64) error {
	// 1. 更新 SubTask 数据库记录
	// 只需要更新 Status, ErrorMsg, SyncedSize, UpdatedAt
	updates := map[string]interface{}{
		"status":      status,
		"error_msg":   errorMsg,
		"synced_size": syncedSize,
		"updated_at":  time.Now().Unix(),
	}

	result := s.db.Model(&model.SubTask{}).Where("id = ?", subTaskID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("sub task %s not found", subTaskID)
	}

	// 2. 检查是否失败，触发报警
	if status == constant.TaskStatusFailed {
		// 查询出 SubTask 以获取 NodeID 等信息用于报警
		var sub model.SubTask
		s.db.First(&sub, "id = ?", subTaskID)

		utils.GetLogger("task").Warnf("Task %s failed on node %s: %s", subTaskID, sub.NodeID, errorMsg)

		param := map[string]string{
			"task":   subTaskID,
			"node":   sub.NodeID,
			"reason": fmt.Sprintf("TaskFailed: %s", errorMsg),
			"time":   time.Now().Format("15:04:05"),
		}
		// 异步发送报警，避免阻塞接口
		go func() {
			// 任务失败报警
			if err := s.alertService.SendAlert(param); err != nil {
				utils.GetLogger("task").Errorf("Failed to send alert for task %s: %v", subTaskID, err)
			}
		}()
	}

	// 3. 检查是否完成，且任务类型为下载后删除OSS
	if status == constant.TaskStatusCompleted {
		var sub model.SubTask
		if err := s.db.First(&sub, "id = ?", subTaskID).Error; err != nil {
			utils.GetLogger("task").Errorf("Failed to query subtask %s for completion check: %v", subTaskID, err)
		} else {
			// TaskType 2: 下载后同步删除OSS
			if sub.TaskType == 2 {
				// 异步删除 OSS 对象
				go func(objectKey string) {
					if objectKey == "" {
						utils.GetLogger("task").Warnf("Task %s is set to delete OSS object but OssKey is empty", subTaskID)
						return
					}
					if err := s.oss.DeleteFile(objectKey); err != nil {
						utils.GetLogger("task").Errorf("Failed to delete OSS object %s for task %s: %v", objectKey, subTaskID, err)
					} else {
						utils.GetLogger("task").Infof("Successfully deleted OSS object %s for task %s", objectKey, subTaskID)
					}
				}(sub.OssKey)
			}
		}
	}

	return nil
}
