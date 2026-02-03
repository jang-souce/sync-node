package handler

import (
	"sync-node/central/service/config"
	"sync-node/central/util"
	"sync-node/common/constant"
	"sync-node/common/model"
	"sync-node/common/utils"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 配置相关的 HTTP 请求处理器
type ConfigHandler struct {
	service *config.ConfigService
}

// NewConfigHandler 创建 ConfigHandler 实例
func NewConfigHandler(service *config.ConfigService) *ConfigHandler {
	return &ConfigHandler{service: service}
}

// GetConfig 获取全局配置
// 响应: GlobalConfig 结构体 JSON
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	// 调用 Service 层获取配置，支持 Context 取消
	cfg, err := h.service.GetGlobalConfig(c.Request.Context())
	if err != nil {
		utils.Error(c, constant.CodeEtcdError, err.Error())
		return
	}
	utils.Success(c, cfg)
}

// UpdateConfig 更新全局配置
// 入参: GlobalConfig 结构体 JSON
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var req model.GlobalConfig
	// 绑定 JSON 参数
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, constant.CodeInvalidParam, util.Translate(err))
		return
	}

	// 调用 Service 层更新配置到 Etcd
	if err := h.service.UpdateGlobalConfig(c.Request.Context(), &req); err != nil {
		utils.Error(c, constant.CodeEtcdError, err.Error())
		return
	}
	utils.Success(c, nil)
}
