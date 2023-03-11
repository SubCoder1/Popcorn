package middlewares

import (
	"github.com/gin-gonic/gin"
)

// This middleware is used to add headers to response for Server-Side-Events (SSE) to work properly.
func SSEMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Next()
	}
}
