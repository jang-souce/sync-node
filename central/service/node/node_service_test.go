package node

import (
	"context"
	"sync-node/common/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Use a unique DB name for each test to avoid shared cache collisions
	dbName := "file::memory:?cache=shared&_query_id=" + t.Name()
	db, _ := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	db.AutoMigrate(&model.NodeState{})
	return db
}

func TestNodeService_GetNodes(t *testing.T) {
	db := setupTestDB(t)
	service := NewNodeService(db)

	// Mock Data
	node1 := model.NodeState{NodeID: "node-1", Status: "online"}
	node2 := model.NodeState{NodeID: "node-2", Status: "offline"}
	db.Create(&node1)
	db.Create(&node2)

	nodes, err := service.GetNodes(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Sort or find to verify specific nodes, as order isn't guaranteed
	nodeMap := make(map[string]*model.NodeState)
	for _, n := range nodes {
		nodeMap[n.NodeID] = n
	}

	assert.Contains(t, nodeMap, "node-1")
	assert.Equal(t, "online", nodeMap["node-1"].Status)
	assert.Contains(t, nodeMap, "node-2")
	assert.Equal(t, "offline", nodeMap["node-2"].Status)
}

func TestNodeService_GetNode(t *testing.T) {
	db := setupTestDB(t)
	service := NewNodeService(db)

	// Mock Data
	node := model.NodeState{NodeID: "node-1", Status: "online"}
	db.Create(&node)

	result, err := service.GetNode(context.Background(), "node-1")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "node-1", result.NodeID)
	assert.Equal(t, "online", result.Status)

	// Test Not Found
	result, err = service.GetNode(context.Background(), "node-unknown")
	assert.NoError(t, err)
	assert.Nil(t, result)
}
