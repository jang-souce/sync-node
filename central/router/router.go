package router

import (
	"sync-node/central/handler"

	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由
func InitRouter(r *gin.Engine, taskH *handler.TaskHandler, nodeH *handler.NodeHandler, configH *handler.ConfigHandler) {
	// 健康检查接口，用于验证服务存活
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong from central"})
	})

	// 注册 API 路由组
	apiV1 := r.Group("/api/v1")
	{
		// 任务管理接口
		apiV1.GET("/tasks", taskH.GetTasks)          // 获取任务列表
		apiV1.POST("/tasks", taskH.CreateTask)       // 创建新任务
		apiV1.GET("/tasks/:id", taskH.GetTask)       // 获取单个任务详情
		apiV1.DELETE("/tasks/:id", taskH.DeleteTask) // 删除任务

		// 节点管理接口
		apiV1.GET("/nodes", nodeH.GetNodes)    // 获取节点列表
		apiV1.GET("/nodes/:id", nodeH.GetNode) // 获取节点详情

		// 配置管理接口
		apiV1.GET("/config", configH.GetConfig)    // 获取全局配置
		apiV1.PUT("/config", configH.UpdateConfig) // 更新全局配置
	}
}
