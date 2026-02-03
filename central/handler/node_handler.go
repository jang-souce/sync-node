package handler

import (
	"sync-node/central/service/node"
	"sync-node/common/constant"
	"sync-node/common/utils"

	"github.com/gin-gonic/gin"
)

// NodeHandler 节点相关的 HTTP 请求处理器
type NodeHandler struct {
	service *node.NodeService
}

// NewNodeHandler 创建 NodeHandler 实例
func NewNodeHandler(service *node.NodeService) *NodeHandler {
	return &NodeHandler{service: service}
}

// GetNodes 获取节点列表
// 响应: []NodeInfo 结构体数组 JSON
func (h *NodeHandler) GetNodes(c *gin.Context) {
	// 调用 Service 层获取所有在线节点
	nodes, err := h.service.GetNodes(c.Request.Context())
	if err != nil {
		utils.Error(c, constant.CodeEtcdError, err.Error())
		return
	}
	utils.Success(c, nodes)
}

// GetNode 获取指定 ID 的节点信息
// 路径参数: id (节点 ID)
func (h *NodeHandler) GetNode(c *gin.Context) {
	id := c.Param("id")
	// 调用 Service 层获取单个节点详情
	node, err := h.service.GetNode(c.Request.Context(), id)
	if err != nil {
		utils.Error(c, constant.CodeEtcdError, err.Error())
		return
	}
	if node == nil {
		utils.Error(c, constant.CodeNodeNotFound, "node not found")
		return
	}
	utils.Success(c, node)
}
