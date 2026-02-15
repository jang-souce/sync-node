package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// AuthHeader 认证头
	AuthHeader = "Authorization"
	// AuthTokenPrefix Bearer Token 前缀
	AuthTokenPrefix = "Bearer "
	// DefaultSecret 默认密钥 (实际生产中应从配置读取)
	DefaultSecret = "sync-node-secret-key"
)

// AuthMiddleware 简单的 Token 认证中间件
// 这里为了演示，使用简单的静态 Token 验证
// 生产环境应结合 JWT 或数据库验证
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		
		// 1. 尝试从 Header 获取
		authHeader := c.GetHeader(AuthHeader)
		if authHeader != "" && strings.HasPrefix(authHeader, AuthTokenPrefix) {
			token = strings.TrimPrefix(authHeader, AuthTokenPrefix)
		}

		// 2. 尝试从 Query 获取 (方便测试和简单客户端)
		if token == "" {
			token = c.Query("token")
		}

		// 验证 Token
		// 这里简化处理，只要 Token 等于预设密钥即可
		// 实际项目中请替换为真实的验证逻辑
		if token != DefaultSecret {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Unauthorized: Invalid or missing token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
