package task

import (
	"context"
	"fmt"
	"time"

	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/utils"

	"gorm.io/gorm"
)

// UpdateTaskStatus 更新任务状态（供 Agent 上报）
// subTaskID: 子任务 ID
// status: 任务新状态
// errorMsg: 错误信息（如果有）
// syncedSize: 已同步大小
func (s *TaskService) UpdateTaskStatus(ctx context.Context, subTaskID string, status string, errorMsg string, syncedSize int64) error {
	var sub model.SubTask

	// 使用重试机制处理数据库锁等瞬时错误
	err := utils.Retry(ctx, 3, 100*time.Millisecond, 1*time.Second, func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			// 1. 更新 SubTask 数据库记录
			updates := map[string]interface{}{
				"status":      status,
				"error_msg":   errorMsg,
				"synced_size": syncedSize,
				"updated_at":  time.Now().Unix(),
			}

			result := tx.Model(&model.SubTask{}).Where("id = ?", subTaskID).Updates(updates)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return fmt.Errorf("sub task %s not found", subTaskID)
			}

			// 2. 查询 SubTask 用于后续逻辑（报警、聚合等）
			// 在事务内查询以确保数据一致性
			return tx.First(&sub, "id = ?", subTaskID).Error
		})
	})

	if err != nil {
		return err
	}

	// 3. 检查是否失败，触发报警
	if status == constant.TaskStatusFailed {
		utils.GetLogger("task").Warnf("Task %s failed on node %s: %s", subTaskID, sub.NodeID, errorMsg)

		param := map[string]string{
			"task":   subTaskID,
			"node":   sub.NodeID,
			"reason": fmt.Sprintf("TaskFailed: %s", errorMsg),
			"time":   time.Now().Format("15:04:05"),
		}
		// 异步发送报警，避免阻塞接口
		go func() {
			if err := s.alertService.SendAlert(param); err != nil {
				utils.GetLogger("task").Errorf("Failed to send alert for task %s: %v", subTaskID, err)
			}
		}()
	}

	// 4. 检查是否完成，且任务类型为下载后删除OSS
	if status == constant.TaskStatusCompleted {
		// TaskType 2: 下载后同步删除OSS
		if sub.TaskType == 2 {
			// 异步删除 OSS 对象
			go func(objectKey string) {
				utils.GetLogger("task").Infof("Task %s (Type 2) completed, attempting to delete OSS object: %s", subTaskID, objectKey)
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

			// 尝试删除源文件 (如果源文件也在同一个 OSS Bucket 中)
			go func(sourceURL string) {
				if sourceURL == "" {
					return
				}
				utils.GetLogger("task").Infof("Task %s (Type 2) checking source file deletion. SourceURL: %s", subTaskID, sourceURL)
				sourceKey, ok := s.oss.ParseObjectKeyFromURL(sourceURL)
				if ok {
					utils.GetLogger("task").Infof("Task %s (Type 2) source is in same OSS bucket, attempting to delete source object: %s", subTaskID, sourceKey)
					if err := s.oss.DeleteFile(sourceKey); err != nil {
						utils.GetLogger("task").Errorf("Failed to delete source OSS object %s for task %s: %v", sourceKey, subTaskID, err)
					} else {
						utils.GetLogger("task").Infof("Successfully deleted source OSS object %s for task %s", sourceKey, subTaskID)
					}
				} else {
					utils.GetLogger("task").Infof("Task %s (Type 2) source URL %s could not be parsed as a key in the current OSS bucket (provider/bucket mismatch or external link)", subTaskID, sourceURL)
				}
			}(sub.SourceURL)
		}
	}

	// 5. 聚合更新 MainTask 状态和进度
	if sub.MainTaskID != "" {
		go s.aggregateMainTask(sub.MainTaskID)
	}

	return nil
}

// aggregateMainTask 聚合主任务状态和进度
func (s *TaskService) aggregateMainTask(mainTaskID string) {
	err := utils.Retry(context.Background(), 3, 100*time.Millisecond, 1*time.Second, func() error {
		return s.db.Transaction(func(tx *gorm.DB) error {
			var subTasks []model.SubTask
			if err := tx.Where("main_task_id = ?", mainTaskID).Find(&subTasks).Error; err != nil {
				return err
			}

			var totalSynced int64
			var completedCount int
			var failedCount int
			var pendingCount int
			var downloadingCount int

			for _, t := range subTasks {
				totalSynced += t.SyncedSize
				switch t.Status {
				case constant.TaskStatusCompleted:
					completedCount++
				case constant.TaskStatusFailed:
					failedCount++
				case constant.TaskStatusPending:
					pendingCount++
				case constant.TaskStatusDownloading:
					downloadingCount++
				}
			}

			newStatus := constant.TaskStatusPending
			if failedCount > 0 {
				newStatus = constant.TaskStatusFailed
			} else if downloadingCount > 0 || completedCount > 0 {
				// 只要有正在下载或已完成的，且没有失败，总体状态为下载中
				newStatus = constant.TaskStatusDownloading
				// 如果全部完成
				if completedCount == len(subTasks) {
					newStatus = constant.TaskStatusCompleted
				}
			}

			// 更新 MainTask
			updates := map[string]interface{}{
				"synced_size": totalSynced,
				"status":      newStatus,
			}

			return tx.Model(&model.MainTask{}).Where("id = ?", mainTaskID).Updates(updates).Error
		})
	})

	if err != nil {
		utils.GetLogger("task").Errorf("Failed to aggregate main task %s after retries: %v", mainTaskID, err)
	}
}

// CleanTimeoutTasks 清理超时任务
// 1. 查找长时间处于 Pending 或 Downloading 状态的任务
// 2. 标记为 Failed
// 3. 从 Etcd 中移除
// 4. 发送报警
func (s *TaskService) CleanTimeoutTasks(ctx context.Context, timeoutSeconds int) error {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300 // 默认 5 分钟
	}
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second
	threshold := time.Now().Add(-timeoutDuration).Unix()

	// 1. 查找超时任务
	var timeoutTasks []model.SubTask
	err := s.db.Where("status IN ? AND updated_at < ?",
		[]string{constant.TaskStatusPending, constant.TaskStatusDownloading},
		threshold).Find(&timeoutTasks).Error
	if err != nil {
		return fmt.Errorf("failed to query timeout tasks: %w", err)
	}

	if len(timeoutTasks) == 0 {
		return nil
	}

	utils.GetLogger("task").Infof("Found %d timeout tasks", len(timeoutTasks))

	for _, task := range timeoutTasks {
		// 2. 标记为 Failed
		updates := map[string]interface{}{
			"status":     constant.TaskStatusFailed,
			"error_msg":  "Task timeout (central cleanup)",
			"updated_at": time.Now().Unix(),
		}
		if err := s.db.Model(&task).Updates(updates).Error; err != nil {
			utils.GetLogger("task").Errorf("Failed to update timeout task %s: %v", task.ID, err)
			continue
		}

		// 3. 从 Etcd 中移除 (Key: /file_sync/task/{node_id}/{sub_task_id})
		etcdKey := fmt.Sprintf("%s%s/%s", constant.EtcdTaskPrefix, task.NodeID, task.ID)
		if _, err := s.etcd.Delete(ctx, etcdKey); err != nil {
			utils.GetLogger("task").Errorf("Failed to delete timeout task %s from etcd: %v", task.ID, err)
		}

		// 4. 发送报警
		param := map[string]string{
			"task":   task.ID,
			"node":   task.NodeID,
			"reason": "TaskTimeout",
			"time":   time.Now().Format("15:04:05"),
		}
		go func(p map[string]string) {
			if err := s.alertService.SendAlert(p); err != nil {
				utils.GetLogger("task").Errorf("Failed to send alert for timeout task: %v", err)
			}
		}(param)
	}

	return nil
}
