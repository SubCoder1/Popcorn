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
		gctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat, X-Request-ID")
		gctx.Writer.Header().Set("Access-Control-Allow-Methods", "HEAD, PATCH, POST, OPTIONS, GET, PUT, DELETE")
		gctx.Writer.Header().Set("Access-Control-Expose-Headers", "Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Location, Upload-Offset, Upload-Length")

		if gctx.Request.Method == "OPTIONS" {
			gctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		gctx.Next()
	}
}

// This middleware is used to add response headers for Server-Side-Events (SSE) to work properly.
func SSECORSMiddleware() gin.HandlerFunc {
	return func(gctx *gin.Context) {
		gctx.Writer.Header().Set("Content-Type", "text/event-stream")
		gctx.Writer.Header().Set("Cache-Control", "no-cache")
		gctx.Writer.Header().Set("Connection", "keep-alive")
		gctx.Next()
	}
}
