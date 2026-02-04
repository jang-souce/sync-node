package node

import (
	"context"
	"errors"
	"sync-node/common/model"

	"gorm.io/gorm"
)

// NodeService 提供节点信息查询服务
type NodeService struct {
	db *gorm.DB
}

// NewNodeService 创建 NodeService 实例
func NewNodeService(db *gorm.DB) *NodeService {
	return &NodeService{db: db}
}

// GetNodes 获取所有已注册的节点信息
func (s *NodeService) GetNodes(ctx context.Context) ([]*model.NodeState, error) {
	var nodes []*model.NodeState
	if err := s.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNode 获取指定 ID 的节点信息
func (s *NodeService) GetNode(ctx context.Context, id string) (*model.NodeState, error) {
	var node model.NodeState
	if err := s.db.WithContext(ctx).Where("node_id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}
