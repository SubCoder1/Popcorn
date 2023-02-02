package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// This middleware handles CORS policy for Popcorn server.
func CORSMiddleware(addr string) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		gctx.Writer.Header().Set("Access-Control-Allow-Origin", addr)
		gctx.Writer.Header().Set("Vary", "Origin")
		gctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		gctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		gctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if gctx.Request.Method == "OPTIONS" {
			gctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		gctx.Next()
	}
}
