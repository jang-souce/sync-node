package router

import (
	"net/http"
	"sync-node/central/handler"
	"sync-node/common/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由
func InitRouter(r *gin.Engine, taskH *handler.TaskHandler, nodeH *handler.NodeHandler, configH *handler.ConfigHandler, fileH *handler.FileHandler) {
	// CORS 中间件配置
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// 全局限流中间件 (每秒 100 请求，突发 200)
	r.Use(middleware.RateLimitMiddleware(100, 200))

	// 静态文件服务
	r.Static("/view", "./view")
	// 根路径重定向到 /view/
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/view/")
	})

	// 健康检查接口，用于验证服务存活
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong from central"})
	})

	// 注册 API 路由组
	apiV1 := r.Group("/api/v1")
	{
		// 公开的读接口 (为了兼容 Dashboard)
		apiV1.GET("/tasks", taskH.GetTasks)                // 获取任务列表
		apiV1.GET("/tasks/:id", taskH.GetTask)             // 获取单个任务详情
		apiV1.GET("/nodes", nodeH.GetNodes)                // 获取节点列表
		apiV1.GET("/nodes/:id", nodeH.GetNode)             // 获取节点详情
		apiV1.GET("/config", configH.GetConfig)            // 获取全局配置
		apiV1.GET("/files/download", fileH.GetDownloadURL) // 获取下载链接

		// 写接口 (内部使用，无需鉴权)
		apiV1.POST("/tasks", taskH.CreateTask)       // 创建新任务
		apiV1.DELETE("/tasks/:id", taskH.DeleteTask) // 删除任务
		apiV1.PUT("/config", configH.UpdateConfig)   // 更新全局配置

		//接口测试用
		apiV1.POST("/files/upload", fileH.UploadFile) // 上传文件
		apiV1.DELETE("/files", fileH.DeleteFile)      // 删除文件
	}
}
