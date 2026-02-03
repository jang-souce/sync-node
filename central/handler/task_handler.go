package handler

import (
	"strconv"
	"sync-node/central/service/task"
	"sync-node/central/util"
	"sync-node/common/constant"
	"sync-node/common/utils"

	"github.com/gin-gonic/gin"
)

// TaskHandler 任务相关的 HTTP 请求处理器
type TaskHandler struct {
	service *task.TaskService
}

// NewTaskHandler 创建 TaskHandler 实例
func NewTaskHandler(service *task.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

// GetTasks 获取任务列表
// 参数: page, pageSize
func (h *TaskHandler) GetTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	// keyword 暂时不支持，因为 MainTask 没有文件名等信息，除非联表查 SubTask

	// 分页查询任务列表
	tasks, total, err := h.service.GetTasks(page, pageSize)
	if err != nil {
		utils.Error(c, constant.CodeInternalErr, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"total": total,
		"items": tasks,
	})
}

// CreateTask 创建任务
// 入参: file_data (JSON 数组)
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req task.CreateTaskReq
	// 定义请求参数结构体，用于绑定 JSON
	var input struct {
		FileData []struct {
			FileName     string `json:"fileName" binding:"required"`
			SourceURL    string `json:"sourceUrl" binding:"required,url"`
			TaskType     int    `json:"taskType" binding:"oneof=0 1 2"` // 允许不传默认为0->1
			Tag          string `json:"tag"`
			TargetNodeID string `json:"targetNodeId"` // 可选
		} `json:"file_data" binding:"required,dive"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, constant.CodeInvalidParam, util.Translate(err))
		return
	}

	// 转换参数格式
	for _, f := range input.FileData {
		tType := f.TaskType
		if tType == 0 {
			tType = 1 // 默认为普通任务
		}
		req.Files = append(req.Files, struct {
			FileName     string `json:"fileName"`
			SourceURL    string `json:"sourceUrl"`
			TaskType     int    `json:"taskType"`
			Tag          string `json:"tag"`
			TargetNodeID string `json:"targetNodeId"`
		}{
			FileName:     f.FileName,
			SourceURL:    f.SourceURL,
			TaskType:     tType,
			Tag:          f.Tag,
			TargetNodeID: f.TargetNodeID,
		})
	}

	// 调用服务层创建任务
	mainTask, err := h.service.CreateTask(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c, constant.CodeInternalErr, err.Error())
		return
	}

	utils.Success(c, mainTask)
}

// GetTask 获取任务详情
// 路径参数: id (任务 ID)
func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	// 查询任务详情（包含子任务）
	mainTask, err := h.service.GetTaskByID(id)
	if err != nil {
		utils.Error(c, constant.CodeTaskNotFound, "task not found")
		return
	}

	utils.Success(c, mainTask)
}

// DeleteTask 删除任务
// 路径参数: id (任务 ID)
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	// 逻辑删除任务及其子任务，并清理 Etcd
	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		utils.Error(c, constant.CodeInternalErr, err.Error())
		return
	}
	utils.Success(c, nil)
}
